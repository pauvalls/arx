package domain

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Layer represents an architectural layer in the system
type Layer struct {
	Name        string   `json:"name" yaml:"name"`
	Paths       []string `json:"paths" yaml:"paths"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// MatchesPath checks if a given file path matches any of the layer's path patterns.
// Supports glob patterns: * (single segment), ** (zero or more segments).
func (l *Layer) MatchesPath(filePath string) bool {
	for _, pattern := range l.Paths {
		if strings.Contains(pattern, "**") {
			// Use regex-based matching for ** patterns
			if matchGlobPattern(pattern, filePath) {
				return true
			}
		} else if strings.Contains(pattern, "*") {
			// Use filepath.Match for simple glob patterns
			matched, err := filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		} else if strings.HasSuffix(pattern, "/") {
			// Directory prefix matching
			if strings.HasPrefix(filePath, pattern) {
				return true
			}
		} else if strings.HasPrefix(filePath, pattern) {
			// Exact or prefix matching
			return true
		}
	}
	return false
}

// matchGlobPattern converts a glob pattern with ** to regex and matches.
func matchGlobPattern(pattern, filePath string) bool {
	// Escape regex metacharacters
	escaped := regexp.QuoteMeta(pattern)

	// Replace /** with (/.*)? (matches zero or more path segments)
	escaped = strings.ReplaceAll(escaped, `/\*\*`, "(/.*)?")

	// Replace any remaining ** with .*
	escaped = strings.ReplaceAll(escaped, `\*\*`, ".*")

	// Replace * with [^/]* (matches anything except /)
	escaped = strings.ReplaceAll(escaped, `\*`, "[^/]*")

	patternRegex := "^" + escaped + "$"

	matched, err := regexp.MatchString(patternRegex, filePath)
	if err != nil {
		return false
	}

	return matched
}

// Validate checks if the layer configuration is valid
func (l *Layer) Validate() error {
	if l.Name == "" {
		return fmt.Errorf("layer name is required")
	}
	if len(l.Paths) == 0 {
		return fmt.Errorf("layer %q must have at least one path pattern", l.Name)
	}
	return nil
}
