package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestParseDiff(t *testing.T) {
	tests := []struct {
		name      string
		diff      string
		wantHunks int
		wantFiles int
		wantStats map[string]int
	}{
		{
			name: "single file modified",
			diff: `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 line1
-line2
+new_line2
 line3
+line4`,
			// Each line in the hunk body produces a DiffHunk (context, removed, added, context, added)
			wantHunks: 5,
			wantFiles: 1,
		},
		{
			name: "new file",
			diff: `diff --git a/new.go b/new.go
new file mode 100644
index 000..abc 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package main
+
+func main() {}`,
			// 3 added lines
			wantHunks: 3,
			wantFiles: 1,
		},
		{
			name: "deleted file",
			diff: `diff --git a/old.go b/old.go
deleted file mode 100644
index abc..000 100644
--- a/old.go
+++ /dev/null
@@ -1,5 +0,0 @@
-package old
-
-func OldFunc() {
-	println("old")
-}`,
			// 5 deleted lines
			wantHunks: 5,
			wantFiles: 1,
		},
		{
			name:      "empty diff",
			diff:      "",
			wantHunks: 0,
			wantFiles: 0,
		},
		{
			name: "multiple files",
			diff: `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,2 @@
 old
+new
diff --git a/b.go b/b.go
--- /dev/null
+++ b/b.go
@@ -0,0 +1,1 @@
+newfile`,
			// a.go: context + 1 add = 2; b.go: 1 add = 1; total = 3
			wantHunks: 3,
			wantFiles: 2,
		},
		{
			name: "multiple hunks in one file",
			diff: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 a
 b
+c
 d
@@ -10,5 +11,6 @@
 x
 y
+z
 w`,
			// First hunk: 4 lines (context, context, added, context); Second hunk: 4 lines
			wantHunks: 8,
			wantFiles: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, err := ParseDiff(tt.diff)
			if err != nil {
				t.Fatalf("ParseDiff() unexpected error: %v", err)
			}

			if len(summary.Hunks) != tt.wantHunks {
				t.Errorf("hunks = %d, want %d", len(summary.Hunks), tt.wantHunks)
			}

			// Check stats
			if summary.Stats != nil {
				if files := summary.Stats["files"]; files != tt.wantFiles {
					t.Errorf("files = %d, want %d", files, tt.wantFiles)
				}
			}
		})
	}
}

func TestParseDiff_HunkDetails(t *testing.T) {
	diff := `diff --git a/internal/service.go b/internal/service.go
index abc..def 100644
--- a/internal/service.go
+++ b/internal/service.go
@@ -10,7 +10,8 @@ package service

 import (
 	"errors"
+	"fmt"
 	"strings"
 )`

	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() unexpected error: %v", err)
	}

	if len(summary.Hunks) < 3 {
		t.Fatalf("expected at least 3 changes, got %d", len(summary.Hunks))
	}

	// Find the added line "+	\"fmt\""
	var addedHunk *domain.DiffHunk
	for _, h := range summary.Hunks {
		if h.Content == "+	\"fmt\"" {
			addedHunk = &h
			break
		}
	}
	if addedHunk == nil {
		t.Fatal("expected to find added hunk for \"fmt\"")
	}
	if addedHunk.File != "internal/service.go" {
		t.Errorf("file = %q, want %q", addedHunk.File, "internal/service.go")
	}
	if addedHunk.NewLine != 13 {
		t.Errorf("new line for added hunk = %d, want 13", addedHunk.NewLine)
	}
	if addedHunk.OldLine != 0 {
		t.Errorf("old line for added hunk = %d, want 0 (new code)", addedHunk.OldLine)
	}
}

func TestFilterViolationsForDiff(t *testing.T) {
	v1 := domain.Violation{File: "a.go", Line: 1, RuleID: "R001"}
	v2 := domain.Violation{File: "a.go", Line: 5, RuleID: "R002"}
	v3 := domain.Violation{File: "b.go", Line: 10, RuleID: "R003"}
	v4 := domain.Violation{File: "a.go", Line: 2, RuleID: "R004"}

	diff := &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{
			{File: "a.go", NewLine: 1, OldLine: 0},   // new line
			{File: "a.go", NewLine: 5, OldLine: 5},   // modified line
			{File: "b.go", NewLine: 0, OldLine: 10},  // deleted line (skip)
		},
	}

	filtered := FilterViolationsForDiff([]domain.Violation{v1, v2, v3, v4}, diff)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered violations, got %d: %+v", len(filtered), filtered)
	}

	// v1 (a.go:1) is a new line — should be included
	// v2 (a.go:5) is a modified line — should be included
	// v3 (b.go:10) is a deleted file line — should be excluded
	// v4 (a.go:2) is not in the diff — should be excluded

	found := make(map[string]bool)
	for _, v := range filtered {
		found[v.RuleID] = true
	}
	if !found["R001"] {
		t.Error("expected R001 (a.go:1, new line) to be included")
	}
	if !found["R002"] {
		t.Error("expected R002 (a.go:5, modified line) to be included")
	}
	if found["R003"] {
		t.Error("expected R003 (b.go:10, deleted line) to be excluded")
	}
	if found["R004"] {
		t.Error("expected R004 (a.go:2, not in diff) to be excluded")
	}
}

func TestFilterViolationsForDiff_EmptyDiff(t *testing.T) {
	v := []domain.Violation{{File: "a.go", Line: 1, RuleID: "R001"}}
	filtered := FilterViolationsForDiff(v, &domain.PRDiffSummary{Hunks: nil})
	if len(filtered) != 0 {
		t.Errorf("expected 0 violations for empty diff, got %d", len(filtered))
	}
}

func TestFilterViolationsForDiff_NewFile(t *testing.T) {
	// New file: old line is 0, all violations in that file should be included
	v := []domain.Violation{
		{File: "new.go", Line: 1, RuleID: "R001"},
		{File: "new.go", Line: 10, RuleID: "R002"},
		{File: "old.go", Line: 5, RuleID: "R003"},
	}

	diff := &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{
			{File: "new.go", NewLine: 1, OldLine: 0},
			{File: "new.go", NewLine: 10, OldLine: 0},
			{File: "old.go", NewLine: 0, OldLine: 5},
		},
	}

	filtered := FilterViolationsForDiff(v, diff)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 violations for new file, got %d", len(filtered))
	}
}

func TestFilterViolationsForDiff_NilInput(t *testing.T) {
	filtered := FilterViolationsForDiff(nil, &domain.PRDiffSummary{})
	if filtered != nil {
		t.Errorf("expected nil for nil input, got %v", filtered)
	}
}

func TestParseDiff_InvalidHunkHeader(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -invalid@@

content`
	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() unexpected error: %v", err)
	}
	// Invalid hunk headers are skipped
	if len(summary.Hunks) != 0 {
		t.Errorf("expected 0 hunks for invalid header, got %d", len(summary.Hunks))
	}
}

func TestPRCheckResult_Passed(t *testing.T) {
	tests := []struct {
		name string
		r    PRCheckResult
		want bool
	}{
		{
			name: "no violations",
			r:    PRCheckResult{NewViolations: nil, Passed: true},
			want: true,
		},
		{
			name: "empty violations",
			r:    PRCheckResult{NewViolations: []domain.Violation{}, Passed: true},
			want: true,
		},
		{
			name: "has violations",
			r:    PRCheckResult{NewViolations: []domain.Violation{{RuleID: "R001"}}, Passed: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.r.Passed != tt.want {
				t.Errorf("Passed = %v, want %v", tt.r.Passed, tt.want)
			}
		})
	}
}

func TestGetGitDiff(t *testing.T) {
	// Test that GetGitDiff returns an error in a non-git directory
	_, err := GetGitDiff("/nonexistent/dir", "HEAD", "HEAD~1")
	if err == nil {
		t.Fatal("expected error in non-existent directory")
	}
}
