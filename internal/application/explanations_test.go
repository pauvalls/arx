package application

import (
	"strings"
	"testing"
)

func TestGetExplanation_ExactMatch(t *testing.T) {
	tests := []struct {
		name   string
		ruleID string
		want   string
	}{
		{
			name:   "domain imports infrastructure",
			ruleID: "domain-imports-infrastructure",
			want:   "The domain layer is the heart of your business logic",
		},
		{
			name:   "domain imports application",
			ruleID: "domain-imports-application",
			want:   "The domain layer is the heart of your business logic",
		},
		{
			name:   "application imports infrastructure",
			ruleID: "application-imports-infrastructure",
			want:   "The application layer exists to orchestrate domain operations",
		},
		{
			name:   "infrastructure imports domain",
			ruleID: "infrastructure-imports-domain",
			want:   "Infrastructure implements adapters",
		},
		{
			name:   "presentation imports infrastructure",
			ruleID: "presentation-imports-infrastructure",
			want:   "The presentation layer",
		},
		{
			name:   "presentation imports domain",
			ruleID: "presentation-imports-domain",
			want:   "The presentation layer should delegate",
		},
		{
			name:   "domain circular",
			ruleID: "domain-circular",
			want:   "Circular dependencies create maintenance nightmares",
		},
		{
			name:   "application circular",
			ruleID: "application-circular",
			want:   "Circular dependencies in the application layer",
		},
		{
			name:   "infrastructure circular",
			ruleID: "infrastructure-circular",
			want:   "Circular dependencies in infrastructure",
		},
		{
			name:   "layer circular",
			ruleID: "layer-circular",
			want:   "Circular dependencies between layers",
		},
		{
			name:   "default fallback",
			ruleID: "unknown-rule-id",
			want:   "This dependency violates an architectural rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExplanation(tt.ruleID)
			if !strings.Contains(got, tt.want) {
				t.Errorf("GetExplanation(%q) = %q, want containing %q", tt.ruleID, got, tt.want)
			}
		})
	}
}

func TestGetExplanation_PatternMatching(t *testing.T) {
	tests := []struct {
		name   string
		ruleID string
		want   string
	}{
		{
			name:   "prefix match domain-*",
			ruleID: "domain-imports-external",
			want:   "The domain layer is the heart of your business logic",
		},
		{
			name:   "suffix match *-circular",
			ruleID: "custom-circular",
			want:   "Circular dependencies between layers",
		},
		{
			name:   "prefix match application-*",
			ruleID: "application-imports-external",
			want:   "The application layer exists to orchestrate domain operations",
		},
		{
			name:   "prefix match infrastructure-*",
			ruleID: "infrastructure-imports-external",
			want:   "Infrastructure implements adapters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExplanation(tt.ruleID)
			if !strings.Contains(got, tt.want) {
				t.Errorf("GetExplanation(%q) = %q, want containing %q", tt.ruleID, got, tt.want)
			}
		})
	}
}

func TestGetFixGuidance_ExactMatch(t *testing.T) {
	tests := []struct {
		name   string
		ruleID string
		minLen int
	}{
		{
			name:   "domain imports infrastructure",
			ruleID: "domain-imports-infrastructure",
			minLen: 3,
		},
		{
			name:   "application imports infrastructure",
			ruleID: "application-imports-infrastructure",
			minLen: 3,
		},
		{
			name:   "layer circular",
			ruleID: "layer-circular",
			minLen: 3,
		},
		{
			name:   "default fallback",
			ruleID: "unknown-rule",
			minLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFixGuidance(tt.ruleID)
			if len(got) < tt.minLen {
				t.Errorf("GetFixGuidance(%q) returned %d items, want at least %d", tt.ruleID, len(got), tt.minLen)
			}
			for i, step := range got {
				if strings.TrimSpace(step) == "" {
					t.Errorf("GetFixGuidance(%q)[%d] is empty", tt.ruleID, i)
				}
			}
		})
	}
}

func TestGetFixGuidance_PatternMatching(t *testing.T) {
	tests := []struct {
		name   string
		ruleID string
		want   string
	}{
		{
			name:   "prefix match domain-*",
			ruleID: "domain-imports-external",
			want:   "Move the infrastructure concern",
		},
		{
			name:   "suffix match *-circular",
			ruleID: "custom-circular",
			want:   "Draw your dependency diagram",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFixGuidance(tt.ruleID)
			found := false
			for _, step := range got {
				if strings.Contains(step, tt.want) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetFixGuidance(%q) = %v, want containing %q", tt.ruleID, got, tt.want)
			}
		})
	}
}

func TestExplanationCount(t *testing.T) {
	// Ensure we have 10+ patterns
	if len(explanations) < 10 {
		t.Errorf("Expected at least 10 explanations, got %d", len(explanations))
	}
	if len(fixGuidance) < 10 {
		t.Errorf("Expected at least 10 fix guidance entries, got %d", len(fixGuidance))
	}
}
