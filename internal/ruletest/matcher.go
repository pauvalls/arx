package ruletest

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// ViolationMatcher defines the interface for matching violations against expectations
type ViolationMatcher interface {
	// Match checks violations against the matcher's criteria.
	// Returns (matched bool, detail string).
	Match(violations []domain.Violation) (bool, string)
}

// CountMatcher matches the exact number of violations
type CountMatcher struct {
	Expected int
}

// Match checks if the violation count matches the expected value
func (m *CountMatcher) Match(violations []domain.Violation) (bool, string) {
	got := len(violations)
	if got == m.Expected {
		return true, fmt.Sprintf("expected %d violations, got %d", m.Expected, got)
	}
	return false, fmt.Sprintf("expected %d violations, got %d", m.Expected, got)
}

// FilesMatcher matches violations by file path globs
type FilesMatcher struct {
	Patterns []string
}

// Match checks if any violation file matches the glob patterns
func (m *FilesMatcher) Match(violations []domain.Violation) (bool, string) {
	if len(m.Patterns) == 0 {
		return false, "no file patterns specified"
	}
	for _, v := range violations {
		for _, pattern := range m.Patterns {
			matched, err := filepath.Match(pattern, v.File)
			if err != nil {
				continue
			}
			if matched {
				return true, fmt.Sprintf("found matching file: %s (pattern: %s)", v.File, pattern)
			}
			// Also try matching with ** support via strings.Contains
			if strings.Contains(pattern, "**") {
				// Convert ** glob to prefix/suffix match
				parts := strings.SplitN(pattern, "**", 2)
				prefix := parts[0]
				suffix := ""
				if len(parts) > 1 {
					suffix = parts[1]
				}
				if strings.HasPrefix(v.File, prefix) && (suffix == "" || strings.HasSuffix(v.File, suffix)) {
					return true, fmt.Sprintf("found matching file: %s (pattern: %s)", v.File, pattern)
				}
			}
		}
	}
	return false, "no violations matched any file pattern"
}

// LayersMatcher matches violations by source/target layer combinations
type LayersMatcher struct {
	Expectations []LayerExpectation
}

// Match checks if any violation matches the expected layer combinations
func (m *LayersMatcher) Match(violations []domain.Violation) (bool, string) {
	if len(m.Expectations) == 0 {
		return false, "no layer expectations specified"
	}
	for _, v := range violations {
		for _, exp := range m.Expectations {
			if exp.Source != "" && exp.Target != "" {
				if v.SourceLayer == exp.Source && v.TargetLayer == exp.Target {
					return true, fmt.Sprintf("found matching layer: %s → %s", v.SourceLayer, v.TargetLayer)
				}
			} else if exp.Source != "" {
				if v.SourceLayer == exp.Source {
					return true, fmt.Sprintf("found matching source layer: %s", v.SourceLayer)
				}
			} else if exp.Target != "" {
				if v.TargetLayer == exp.Target {
					return true, fmt.Sprintf("found matching target layer: %s", v.TargetLayer)
				}
			}
		}
	}
	return false, "no violations matched any layer expectation"
}

// PatternsMatcher matches violations by regex on message
type PatternsMatcher struct {
	Patterns []string
}

// Match checks if any violation message matches the regex patterns
func (m *PatternsMatcher) Match(violations []domain.Violation) (bool, string) {
	if len(m.Patterns) == 0 {
		return false, "no patterns specified"
	}
	for _, v := range violations {
		for _, pattern := range m.Patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			if re.MatchString(v.Message) {
				return true, fmt.Sprintf("found matching violation: %q (pattern: %s)", v.Message, pattern)
			}
		}
	}
	return false, "no violations matched any pattern"
}
