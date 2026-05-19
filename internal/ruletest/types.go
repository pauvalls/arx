package ruletest

import (
	"fmt"
)

// MatchMode represents the different ways to match violations
type MatchMode int

const (
	MatchModeCount    MatchMode = iota // Exact violation count
	MatchModeFiles                     // Glob match on violation file paths
	MatchModeLayers                    // Match source/target layer combinations
	MatchModePatterns                  // Regex match on violation messages
)

// String returns the string representation of a MatchMode
func (m MatchMode) String() string {
	switch m {
	case MatchModeCount:
		return "count"
	case MatchModeFiles:
		return "files"
	case MatchModeLayers:
		return "layers"
	case MatchModePatterns:
		return "patterns"
	default:
		return "unknown"
	}
}

// LayerExpectation defines expected source/target layer combinations in violations
type LayerExpectation struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// Expectation defines what a test case expects from EvaluateRules results
type Expectation struct {
	Violations *int              `yaml:"violations,omitempty"`
	Files      []string          `yaml:"files,omitempty"`
	Layers     []LayerExpectation `yaml:"layers,omitempty"`
	Patterns   []string          `yaml:"patterns,omitempty"`
}

// HasExpectations returns true if at least one expectation field is set
func (e Expectation) HasExpectations() bool {
	if e.Violations != nil {
		return true
	}
	if len(e.Files) > 0 {
		return true
	}
	if len(e.Layers) > 0 {
		return true
	}
	if len(e.Patterns) > 0 {
		return true
	}
	return false
}

// IntPtr returns a pointer to the given int value
func IntPtr(n int) *int { return &n }

// TestCase represents a single test case in a test suite
type TestCase struct {
	Name    string      `yaml:"name"`
	Fixture string      `yaml:"fixture,omitempty"`
	RuleID  string      `yaml:"rule,omitempty"`
	Expect  Expectation `yaml:"expect"`
}

// Validate checks that the test case has valid configuration
func (tc TestCase) Validate() error {
	if tc.Name == "" {
		return fmt.Errorf("test case name is required")
	}
	if !tc.Expect.HasExpectations() {
		return fmt.Errorf("test case %q: at least one expectation must be set (violations, files, layers, or patterns)", tc.Name)
	}
	return nil
}

// TestSuite is a collection of test cases
type TestSuite struct {
	Name  string     `yaml:"name"`
	Tests []TestCase `yaml:"tests"`
}

// Validate checks that the test suite has valid configuration
func (ts TestSuite) Validate() error {
	if len(ts.Tests) == 0 {
		return fmt.Errorf("test suite %q: at least one test case is required", ts.Name)
	}
	seen := make(map[string]bool)
	for _, tc := range ts.Tests {
		if err := tc.Validate(); err != nil {
			return err
		}
		if seen[tc.Name] {
			return fmt.Errorf("test suite %q: duplicate test name %q", ts.Name, tc.Name)
		}
		seen[tc.Name] = true
	}
	return nil
}
