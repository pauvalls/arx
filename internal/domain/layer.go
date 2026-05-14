package domain

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Layer represents an architectural layer in the system
type Layer struct {
	Name        string   `json:"name" yaml:"name"`
	Paths       []string `json:"paths" yaml:"paths"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// MatchesPath checks if a given file path matches any of the layer's path patterns
func (l *Layer) MatchesPath(filePath string) bool {
	for _, pattern := range l.Paths {
		// Handle glob patterns
		if strings.Contains(pattern, "*") {
			matched, err := filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		}
		// Handle directory prefix matching
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(filePath, pattern) {
				return true
			}
		}
		// Handle exact or prefix matching
		if filepath.HasPrefix(filePath, pattern) {
			return true
		}
	}
	return false
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
