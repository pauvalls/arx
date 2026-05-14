package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/ports"
)

// Walker implements the ports.FileWriter interface for file system operations
type Walker struct {
	excludePatterns []string
}

// NewWalker creates a new file system walker
func NewWalker(excludePatterns []string) *Walker {
	return &Walker{
		excludePatterns: excludePatterns,
	}
}

// Write writes content to a file at the specified path
// Creates parent directories if they don't exist
func (w *Walker) Write(path string, content []byte) error {
	// Create parent directories
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// Exists checks if a file exists at the specified path
func (w *Walker) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Walk walks the directory tree and returns files matching layer path patterns
// Respects exclude patterns and skips hidden files, vendor, node_modules, etc.
func (w *Walker) Walk(root string, layerPatterns []string) ([]string, error) {
	var matchedFiles []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip common directories that should not be scanned
			name := d.Name()
			if name == "node_modules" ||
				name == "vendor" ||
				name == ".git" ||
				name == "dist" ||
				name == "build" ||
				name == ".next" ||
				name == "out" ||
				strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}

			// Check if directory matches any exclude pattern
			relPath, err := filepath.Rel(root, path)
			if err == nil && w.shouldExclude(relPath) {
				return filepath.SkipDir
			}

			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes for consistent matching
		relPath = filepath.ToSlash(relPath)

		// Check if file should be excluded
		if w.shouldExclude(relPath) {
			return nil
		}

		// Check if file matches any layer pattern
		for _, pattern := range layerPatterns {
			matched, err := filepath.Match(pattern, relPath)
			if err == nil && matched {
				matchedFiles = append(matchedFiles, relPath)
				break
			}

			// Also check if the file is under a directory pattern
			if strings.HasSuffix(pattern, "/**") {
				dirPattern := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(relPath, dirPattern+"/") {
					matchedFiles = append(matchedFiles, relPath)
					break
				}
			} else if strings.HasSuffix(pattern, "**") {
				dirPattern := strings.TrimSuffix(pattern, "**")
				if strings.HasPrefix(relPath, dirPattern) {
					matchedFiles = append(matchedFiles, relPath)
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return matchedFiles, nil
}

// shouldExclude checks if a path matches any exclude pattern
func (w *Walker) shouldExclude(path string) bool {
	for _, pattern := range w.excludePatterns {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}

		// Handle glob patterns with **
		if strings.Contains(pattern, "**") {
			// Convert ** to regex-like matching
			regexPattern := strings.ReplaceAll(pattern, "**", ".*")
			matched, err := filepath.Match(regexPattern, path)
			if err == nil && matched {
				return true
			}

			// Simple prefix/suffix checks for common patterns
			if strings.HasPrefix(pattern, "**/") {
				suffix := strings.TrimPrefix(pattern, "**/")
				if strings.Contains(path, "/"+suffix) || strings.HasSuffix(path, suffix) {
					return true
				}
			}
			if strings.HasSuffix(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "/**")
				if strings.HasPrefix(path, prefix+"/") {
					return true
				}
			}
		}
	}

	return false
}

// Ensure Walker implements ports.FileWriter interface
var _ ports.FileWriter = (*Walker)(nil)
