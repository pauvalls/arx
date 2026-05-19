package domain

import (
	"testing"
)

func TestConflict_TypeCreation(t *testing.T) {
	c := Conflict{
		File: "internal/domain/user.go",
		Suggestions: [2]FixSuggestion{
			{
				ViolationID: "D-01",
				RuleID:      "domain-imports-infrastructure",
				File:        "internal/domain/user.go",
				Line:        5,
				Description: "Extract interface",
				Diff:        "--- a/internal/domain/user.go\n+++ b/internal/domain/user.go\n@@ -5 +5 @@",
			},
			{
				ViolationID: "D-02",
				RuleID:      "domain-imports-infrastructure",
				File:        "internal/domain/user.go",
				Line:        10,
				Description: "Extract interface",
				Diff:        "--- a/internal/domain/user.go\n+++ b/internal/domain/user.go\n@@ -10 +10 @@",
			},
		},
		Description: "Overlapping fixes for internal/domain/user.go",
	}

	if c.File != "internal/domain/user.go" {
		t.Errorf("expected File 'internal/domain/user.go', got %q", c.File)
	}
	if len(c.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(c.Suggestions))
	}
	if c.Suggestions[0].ViolationID != "D-01" {
		t.Errorf("expected first suggestion ViolationID D-01, got %q", c.Suggestions[0].ViolationID)
	}
	if c.Suggestions[1].ViolationID != "D-02" {
		t.Errorf("expected second suggestion ViolationID D-02, got %q", c.Suggestions[1].ViolationID)
	}
	if c.Description != "Overlapping fixes for internal/domain/user.go" {
		t.Errorf("expected Description 'Overlapping fixes for ...', got %q", c.Description)
	}
}

func TestHunkRange_TypeCreation(t *testing.T) {
	r := HunkRange{
		StartLine: 5,
		EndLine:   15,
	}

	if r.StartLine != 5 {
		t.Errorf("expected StartLine 5, got %d", r.StartLine)
	}
	if r.EndLine != 15 {
		t.Errorf("expected EndLine 15, got %d", r.EndLine)
	}
}

func TestFixSuggestion_TypeCreation(t *testing.T) {
	fs := FixSuggestion{
		ViolationID: "D-01",
		RuleID:      "domain-imports-infrastructure",
		File:        "internal/domain/user.go",
		Line:        5,
		Description: "Extract an interface",
		Diff:        "--- a/file.go\n+++ b/file.go\n@@ -5 +5 @@\n-old\n+new",
		HunkRange: HunkRange{
			StartLine: 5,
			EndLine:   6,
		},
	}

	if fs.ViolationID != "D-01" {
		t.Errorf("expected ViolationID D-01, got %q", fs.ViolationID)
	}
	if fs.RuleID != "domain-imports-infrastructure" {
		t.Errorf("expected RuleID domain-imports-infrastructure, got %q", fs.RuleID)
	}
	if fs.File != "internal/domain/user.go" {
		t.Errorf("expected File internal/domain/user.go, got %q", fs.File)
	}
	if fs.Line != 5 {
		t.Errorf("expected Line 5, got %d", fs.Line)
	}
	if fs.HunkRange.StartLine != 5 || fs.HunkRange.EndLine != 6 {
		t.Errorf("expected HunkRange {5,6}, got {%d,%d}", fs.HunkRange.StartLine, fs.HunkRange.EndLine)
	}
}
