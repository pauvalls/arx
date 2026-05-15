package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Op represents a file operation type.
type Op int

const (
	Create Op = iota
	Write
	Remove
	Rename
	Chmod
)

// WatchEvent represents a single debounced file change event.
type WatchEvent struct {
	Path string    `json:"path"`
	Op   Op        `json:"op"`
	Time time.Time `json:"time"`
}

// Watcher monitors file system changes with debounce and .gitignore filtering.
type Watcher struct {
	fsnotify *fsnotify.Watcher
	dirs     []string
	debounce time.Duration
	events   chan WatchEvent
	errors   chan error
	done     chan struct{}
	ignored  []string // compiled .gitignore patterns
	mu       sync.Mutex
	closed   bool
}

// NewWatcher creates a new file watcher for the given directories.
// The watcher reads .gitignore from the first directory (project root) for pattern filtering.
func NewWatcher(dirs []string, debounce time.Duration) (*Watcher, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("at least one directory is required")
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fw.Close()
			return nil, fmt.Errorf("invalid directory %q: %w", dir, err)
		}
		if info, err := os.Stat(absDir); err != nil {
			fw.Close()
			return nil, fmt.Errorf("directory %q does not exist: %w", absDir, err)
		} else if !info.IsDir() {
			fw.Close()
			return nil, fmt.Errorf("%q is not a directory", absDir)
		}
	}

	// Read .gitignore from the first directory (treated as project root)
	var ignorePatterns []string
	gitignorePath := filepath.Join(dirs[0], ".gitignore")
	if data, err := os.ReadFile(gitignorePath); err == nil {
		ignorePatterns = parseGitignore(string(data))
	}

	// Always skip .git directory
	ignorePatterns = append(ignorePatterns, ".git/")

	return &Watcher{
		fsnotify: fw,
		dirs:     dirs[:len(dirs):len(dirs)], // copy to prevent slice mutation
		debounce: debounce,
		events:   make(chan WatchEvent, 100),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
		ignored:  ignorePatterns,
	}, nil
}

// Events returns the channel of debounced file change events.
func (w *Watcher) Events() <-chan WatchEvent {
	return w.events
}

// Errors returns the channel of watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

// Start begins watching for file changes. It blocks until the context is cancelled.
// Events are debounced: if multiple events arrive within the debounce window,
// the timer resets and the latest event per unique path is sent when the timer fires.
func (w *Watcher) Start(ctx context.Context) error {
	// Add all directories to fsnotify (recursively)
	for _, dir := range w.dirs {
		if err := w.addRecursive(dir); err != nil {
			return fmt.Errorf("failed to watch directory %q: %w", dir, err)
		}
	}

	// pending tracks the latest event per path since last debounce fire
	pending := make(map[string]WatchEvent)
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case event, ok := <-w.fsnotify.Events:
			if !ok {
				return nil
			}
			// Filter by .gitignore
			if w.isIgnored(event.Name) {
				continue
			}
			// If a new directory is created, add it to the watcher
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.fsnotify.Add(event.Name)
				}
			}
			// Convert fsnotify event to our event type, overwrite previous for same path
			we := convertEvent(event)
			pending[event.Name] = we

			// Reset debounce timer
			if timer == nil {
				timer = time.NewTimer(w.debounce)
				timerC = timer.C
			} else {
				timer.Stop()
				timer.Reset(w.debounce)
			}

		case <-timerC:
			// Debounce period ended — send latest event per changed path
			for _, evt := range pending {
				select {
				case w.events <- evt:
				default:
					// Channel full, drop event
				}
			}
			pending = make(map[string]WatchEvent)
			timer = nil
			timerC = nil

		case err, ok := <-w.fsnotify.Errors:
			if !ok {
				return nil
			}
			select {
			case w.errors <- err:
			default:
			}

		case <-ctx.Done():
			// Flush any remaining pending events
			for _, evt := range pending {
				select {
				case w.events <- evt:
				default:
				}
			}
			return ctx.Err()

		case <-w.done:
			return nil
		}
	}
}

// addRecursive adds a directory and all its subdirectories to the fsnotify watcher.
func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if !info.IsDir() {
			return nil
		}
		// Skip ignored directories
		if w.isIgnored(path) && path != root {
			return filepath.SkipDir
		}
		return w.fsnotify.Add(path)
	})
}

// Close stops the watcher and cleans up resources.
func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	close(w.done)
	return w.fsnotify.Close()
}

// convertEvent maps a fsnotify event to our WatchEvent type.
func convertEvent(e fsnotify.Event) WatchEvent {
	var op Op
	switch {
	case e.Has(fsnotify.Create):
		op = Create
	case e.Has(fsnotify.Write):
		op = Write
	case e.Has(fsnotify.Remove):
		op = Remove
	case e.Has(fsnotify.Rename):
		op = Rename
	case e.Has(fsnotify.Chmod):
		op = Chmod
	}
	return WatchEvent{
		Path: e.Name,
		Op:   op,
		Time: time.Now(),
	}
}

// isIgnored checks if a path matches any .gitignore pattern.
func (w *Watcher) isIgnored(path string) bool {
	for _, pattern := range w.ignored {
		if matchGitignorePattern(path, pattern) {
			return true
		}
	}
	return false
}

// parseGitignore reads .gitignore content and returns non-comment lines.
func parseGitignore(content string) []string {
	var patterns []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Skip negations (not supported)
		if strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// matchGitignorePattern checks if a file path matches a .gitignore pattern.
// Supports directory patterns (trailing /), glob patterns (containing *),
// and exact/prefix matches.
func matchGitignorePattern(path, pattern string) bool {
	// Normalize: use forward slashes
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Get just the filename for suffix matching
	_, filename := filepath.Split(path)

	// Remove trailing slash for directory patterns
	isDirPattern := strings.HasSuffix(pattern, "/")
	if isDirPattern {
		pattern = strings.TrimSuffix(pattern, "/")
	}

	// Glob pattern (contains *)
	if strings.Contains(pattern, "*") {
		// Simple glob: support *.ext and dir/* patterns
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(filename, suffix)
		}
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			return strings.HasPrefix(path, prefix+"/") || strings.HasPrefix(path, prefix+"\\")
		}
		// Try regex matching for more complex patterns
		re := globToPrefix(pattern)
		return strings.Contains(path, re)
	}

	// Directory pattern: match if path contains /pattern/
	if isDirPattern {
		return strings.Contains(path, "/"+pattern+"/") ||
			strings.HasPrefix(path, pattern+"/")
	}

	// Pattern with /: match as path prefix or suffix
	if strings.Contains(pattern, "/") {
		return strings.HasPrefix(path, pattern) ||
			strings.HasSuffix(path, pattern) ||
			strings.Contains(path, "/"+pattern)
	}

	// Simple name: match filename
	return filename == pattern || strings.HasSuffix(path, "/"+pattern)
}

// globToPrefix converts a simple glob pattern to a prefix string for matching.
func globToPrefix(pattern string) string {
	parts := strings.SplitN(pattern, "*", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return pattern
}
