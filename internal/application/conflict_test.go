package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestDetectConflicts_EmptySuggestions(t *testing.T) {
	conflicts := DetectConflicts(nil)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for empty suggestions, got %d", len(conflicts))
	}
}

func TestDetectConflicts_SingleSuggestion(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new",
		},
	}
	conflicts := DetectConflicts(suggestions)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for single suggestion, got %d", len(conflicts))
	}
}

func TestDetectConflicts_DifferentFiles(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new",
		},
		{
			ViolationID: "D-02",
			File:        "b.go",
			Diff:        "--- a/b.go\n+++ b/b.go\n@@ -10,3 +10,3 @@\n-old\n+new",
		},
	}
	conflicts := DetectConflicts(suggestions)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for different files, got %d", len(conflicts))
	}
}

func TestDetectConflicts_ExactOverlap(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new2",
		},
	}
	conflicts := DetectConflicts(suggestions)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict for exact overlap, got %d", len(conflicts))
	}
	if conflicts[0].File != "a.go" {
		t.Errorf("expected conflict File 'a.go', got %q", conflicts[0].File)
	}
}

func TestDetectConflicts_AdjacentWithinThreeLines(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -9,3 +9,3 @@\n-old\n+new2",
		},
	}
	conflicts := DetectConflicts(suggestions)
	// Range 5-7 and 9-11 are 1 line apart (within 3-line tolerance)
	if len(conflicts) != 1 {
		t.Errorf("expected 1 conflict for adjacent ranges (within 3-line tolerance), got %d", len(conflicts))
	}
}

func TestDetectConflicts_NonOverlappingSameFile(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n-old\n+new",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -20,3 +20,3 @@\n-old\n+new2",
		},
	}
	conflicts := DetectConflicts(suggestions)
	// Range 5-7 and 20-22 are far apart (more than 3 lines between them)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for non-overlapping ranges, got %d", len(conflicts))
	}
}

func TestDetectConflicts_SuggestionWithoutDiff(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "",
		},
	}
	conflicts := DetectConflicts(suggestions)
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for suggestions without diffs, got %d", len(conflicts))
	}
}

func TestDetectConflicts_ParseHunkHeader_Malformed(t *testing.T) {
	// Malformed hunk headers should not cause crash
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -abc,xyz @@\n",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n",
		},
	}
	conflicts := DetectConflicts(suggestions)
	// Malformed header should be skipped gracefully
	_ = conflicts // just ensure no panic
}

func TestDetectConflicts_MultipleConflicts(t *testing.T) {
	suggestions := []domain.FixSuggestion{
		{
			ViolationID: "D-01",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -5,3 +5,3 @@\n",
		},
		{
			ViolationID: "D-02",
			File:        "a.go",
			Diff:        "--- a/a.go\n+++ b/a.go\n@@ -6,3 +6,3 @@\n",
		},
		{
			ViolationID: "D-03",
			File:        "b.go",
			Diff:        "--- a/b.go\n+++ b/b.go\n@@ -1,1 +1,1 @@\n",
		},
		{
			ViolationID: "D-04",
			File:        "b.go",
			Diff:        "--- a/b.go\n+++ b/b.go\n@@ -2,1 +2,1 @@\n",
		},
	}
	conflicts := DetectConflicts(suggestions)
	// a.go: D-01 and D-02 overlap, b.go: D-03 and D-04 overlap
	if len(conflicts) != 2 {
		t.Errorf("expected 2 conflicts (one per file), got %d", len(conflicts))
	}
}
