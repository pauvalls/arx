package domain

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ArxIgnore holds project-wide file exclusion patterns loaded from .arxignore.
// When nil, no files are ignored (backward compatible).
type ArxIgnore struct {
	Patterns []string
}

// LoadArxIgnore reads .arxignore from the project root.
// Returns nil, nil if the file does not exist (backward compatible — no patterns applied).
// Returns an error only if the file exists but cannot be read.
func LoadArxIgnore(root string) (*ArxIgnore, error) {
	filePath := filepath.Join(root, ".arxignore")

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return &ArxIgnore{Patterns: patterns}, nil
}

// IsIgnored checks if a relative path matches any pattern.
// Uses path.Match for glob matching (*, ?, [...]) — cross-platform with forward slashes.
// Directory patterns (ending with /) match any path under that directory.
// Double-star patterns (**/) match at any depth.
func (a *ArxIgnore) IsIgnored(relPath string) bool {
	if a == nil || len(a.Patterns) == 0 {
		return false
	}

	// Normalize to forward slashes for cross-platform matching.
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range a.Patterns {
		if matchPattern(pattern, relPath) {
			return true
		}
	}
	return false
}

// matchPattern checks a single pattern against a path.
// Handles directory prefixes (trailing /), double-star (**), and standard globs.
func matchPattern(pattern, relPath string) bool {
	// Directory pattern: "vendor/" matches "vendor/..." at any depth.
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		if strings.HasPrefix(relPath, dir+"/") || relPath == dir {
			return true
		}
		return false
	}

	// Double-star pattern: "build/**" or "**/test/**" matches at any depth.
	if strings.Contains(pattern, "**") {
		return matchDoubleStar(pattern, relPath)
	}

	// Standard glob: match against full path and basename.
	matched, _ := path.Match(pattern, relPath)
	if matched {
		return true
	}

	// Also match against basename (e.g., "*.go" matches "pkg/foo.go").
	base := path.Base(relPath)
	matched, _ = path.Match(pattern, base)
	return matched
}

// matchDoubleStar handles ** glob patterns.
// "build/**" matches anything under build/.
// "**/test/**" matches any path containing test/ at any level.
func matchDoubleStar(pattern, relPath string) bool {
	// Convert ** to a form that path.Match can handle.
	// "build/**" → prefix match on "build/"
	// "**/foo" → match basename "foo" or any path ending with "/foo"
	// "a/**/b" → match "a/" prefix and "/b" suffix

	if pattern == "**" || pattern == "**/*" {
		return true
	}

	if strings.HasPrefix(pattern, "**/") {
		rest := strings.TrimPrefix(pattern, "**/")
		// Match against full path or any suffix after a slash.
		matched, _ := path.Match(rest, relPath)
		if matched {
			return true
		}
		// Try matching against path suffixes.
		for i := 0; i < len(relPath); i++ {
			if relPath[i] == '/' {
				matched, _ := path.Match(rest, relPath[i+1:])
				if matched {
					return true
				}
			}
		}
		return false
	}

	if strings.HasSuffix(pattern, "/**") || strings.HasSuffix(pattern, "/**/*") {
		prefix := strings.TrimSuffix(pattern, "**")
		prefix = strings.TrimSuffix(prefix, "*")
		prefix = strings.TrimSuffix(prefix, "/")
		return strings.HasPrefix(relPath, prefix+"/") || relPath == prefix
	}

	// General case: a/**/b — split on ** and check prefix/suffix.
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) == 2 {
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")

		if prefix != "" && !strings.HasPrefix(relPath, prefix+"/") && relPath != prefix {
			return false
		}
		if suffix != "" {
			matched, _ := path.Match(suffix, path.Base(relPath))
			if matched {
				return true
			}
			// Try matching suffix against the remaining path after prefix.
			remaining := strings.TrimPrefix(relPath, prefix+"/")
			matched, _ = path.Match(suffix, remaining)
			return matched
		}
		return true
	}

	return false
}
