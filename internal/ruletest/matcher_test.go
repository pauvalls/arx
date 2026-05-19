package ruletest

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestCountMatcher_Match(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/service.go", SourceLayer: "domain", TargetLayer: "infrastructure", Message: "domain depends on infrastructure"},
		{ID: "D-02", File: "internal/application/handler.go", SourceLayer: "application", TargetLayer: "infrastructure", Message: "app depends on infra"},
	}

	tests := []struct {
		name     string
		expected int
		passed   bool
	}{
		{"exact match", 2, true},
		{"too few", 1, false},
		{"too many", 3, false},
		{"zero", 0, false},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &CountMatcher{Expected: tt.expected}
			passed, detail := m.Match(violations)
			if passed != tt.passed {
				t.Errorf("CountMatcher.Match() = %v, want %v (detail: %s)", passed, tt.passed, detail)
			}
			if detail == "" {
				t.Error("CountMatcher.Match() returned empty detail")
			}
		})
	}

	// Empty violations
	t.Run("empty violations", func(t *testing.T) {
		m := &CountMatcher{Expected: 0}
		passed, _ := m.Match(nil)
		if !passed {
			t.Error("CountMatcher.Match() with nil and Expected=0 should pass")
		}
	})
}

func TestFilesMatcher_Match(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/service.go"},
		{ID: "D-02", File: "internal/infrastructure/db.go"},
	}

	tests := []struct {
		name     string
		patterns []string
		passed   bool
	}{
		{"single match", []string{"internal/domain/**"}, true},
		{"multiple patterns one match", []string{"cmd/**", "internal/domain/**"}, true},
		{"no match", []string{"cmd/**"}, false},
		{"empty patterns", []string{}, false},
		{"exact file match", []string{"internal/domain/service.go"}, true},
		{"star glob", []string{"internal/*/service.go"}, true},
		{"no match wrong dir", []string{"external/**"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &FilesMatcher{Patterns: tt.patterns}
			passed, detail := m.Match(violations)
			if passed != tt.passed {
				t.Errorf("FilesMatcher.Match() = %v, want %v (detail: %s)", passed, tt.passed, detail)
			}
			if detail == "" {
				t.Error("FilesMatcher.Match() returned empty detail")
			}
		})
	}

	t.Run("nil violations", func(t *testing.T) {
		m := &FilesMatcher{Patterns: []string{"internal/**"}}
		passed, _ := m.Match(nil)
		if passed {
			t.Error("FilesMatcher.Match() with nil violations should not pass")
		}
	})
}

func TestLayersMatcher_Match(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", SourceLayer: "domain", TargetLayer: "infrastructure"},
		{ID: "D-02", SourceLayer: "application", TargetLayer: "infrastructure"},
	}

	tests := []struct {
		name         string
		expectations []LayerExpectation
		passed       bool
	}{
		{"single match", []LayerExpectation{{Source: "domain", Target: "infrastructure"}}, true},
		{"multiple layers one match", []LayerExpectation{{Source: "application", Target: "infrastructure"}}, true},
		{"no match wrong source", []LayerExpectation{{Source: "presentation", Target: "infrastructure"}}, false},
		{"no match wrong target", []LayerExpectation{{Source: "domain", Target: "database"}}, false},
		{"empty expectations", []LayerExpectation{}, false},
		{"multiple expectations one match", []LayerExpectation{
			{Source: "presentation", Target: "infrastructure"},
			{Source: "domain", Target: "infrastructure"},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &LayersMatcher{Expectations: tt.expectations}
			passed, detail := m.Match(violations)
			if passed != tt.passed {
				t.Errorf("LayersMatcher.Match() = %v, want %v (detail: %s)", passed, tt.passed, detail)
			}
			if detail == "" {
				t.Error("LayersMatcher.Match() returned empty detail")
			}
		})
	}
}

func TestPatternsMatcher_Match(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", Message: "domain layer depends on infrastructure layer"},
		{ID: "D-02", Message: "import cycle detected between user and auth"},
	}

	tests := []struct {
		name     string
		patterns []string
		passed   bool
	}{
		{"single match", []string{"import cycle"}, true},
		{"regex match", []string{"domain.*infrastructure"}, true},
		{"no match", []string{"database connection"}, false},
		{"empty patterns", []string{}, false},
		{"partial word match", []string{"cycle"}, true},
		{"case sensitive no match", []string{"IMPORT CYCLE"}, false},
		{"multiple patterns one match", []string{"database", "cycle"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PatternsMatcher{Patterns: tt.patterns}
			passed, detail := m.Match(violations)
			if passed != tt.passed {
				t.Errorf("PatternsMatcher.Match() = %v, want %v (detail: %s)", passed, tt.passed, detail)
			}
			if detail == "" {
				t.Error("PatternsMatcher.Match() returned empty detail")
			}
		})
	}
}

func TestViolationMatcher_Combined(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/service.go", SourceLayer: "domain", TargetLayer: "infrastructure", Message: "domain depends on infrastructure"},
		{ID: "D-02", File: "internal/application/handler.go", SourceLayer: "application", TargetLayer: "infrastructure", Message: "app depends on infra"},
	}

	tests := []struct {
		name     string
		matchers []ViolationMatcher
		passed   bool
	}{
		{
			name: "all pass",
			matchers: []ViolationMatcher{
				&CountMatcher{Expected: 2},
				&FilesMatcher{Patterns: []string{"internal/domain/**"}},
				&LayersMatcher{Expectations: []LayerExpectation{{Source: "domain", Target: "infrastructure"}}},
				&PatternsMatcher{Patterns: []string{"infrastructure"}},
			},
			passed: true,
		},
		{
			name: "count pass but files fail",
			matchers: []ViolationMatcher{
				&CountMatcher{Expected: 2},
				&FilesMatcher{Patterns: []string{"cmd/**"}},
			},
			passed: false,
		},
		{
			name: "single matcher passes",
			matchers: []ViolationMatcher{
				&CountMatcher{Expected: 2},
			},
			passed: true,
		},
		{
			name:     "no matchers",
			matchers: []ViolationMatcher{},
			passed:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, m := range tt.matchers {
				passed, _ := m.Match(violations)
				if !passed && tt.passed {
					t.Errorf("matcher %T reported fail but expected all pass", m)
				}
			}
		})
	}
}
