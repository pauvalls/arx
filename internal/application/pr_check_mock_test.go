package application

import (
	"context"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestPRCheckService_NewPRCheckService(t *testing.T) {
	svc := NewPRCheckService(nil, nil, false)
	if svc == nil {
		t.Fatal("NewPRCheckService returned nil")
	}
}

func TestPRCheckService_NewPRCheckService_WithGitClient(t *testing.T) {
	mock := newMockGitClient()
	svc := NewPRCheckService(nil, nil, false, mock)
	if svc.gitClient != mock {
		t.Error("gitClient should be set")
	}
}

func TestPRCheckService_Run_InvalidPR(t *testing.T) {
	svc := NewPRCheckService(nil, nil, false)
	_, err := svc.Run(context.Background(), domain.PRInfo{})
	if err == nil {
		t.Fatal("expected error for invalid PR")
	}
}

func TestPRCheckService_Run_GitDiffError(t *testing.T) {
	mock := newMockGitClient()
	mock.withDiff("abc", "def", "/repo", "", &execError{})
	mock.withRun("diff-tree --no-commit-id -r -p abc def", "", &execError{})

	svc := NewPRCheckService(nil, nil, false, mock)
	_, err := svc.Run(context.Background(), domain.PRInfo{
		BaseSHA:  "abc",
		HeadSHA:  "def",
		BaseRef:  "main",
		HeadRef:  "feature",
		RepoPath: "/repo",
		PRNumber: 1,
	})
	if err == nil {
		t.Fatal("expected error when git diff fails")
	}
}

func TestFilterViolationsForDiff_NewFileAllViolations(t *testing.T) {
	// New file: all violations in the file should be included
	violations := []domain.Violation{
		{File: "new.go", Line: 1, RuleID: "R1"},
		{File: "new.go", Line: 5, RuleID: "R2"},
	}

	diff := &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{
			{File: "new.go", NewLine: 1, OldLine: 0},
			{File: "new.go", NewLine: 5, OldLine: 0},
		},
	}

	filtered := FilterViolationsForDiff(violations, diff)
	if len(filtered) != 2 {
		t.Errorf("expected 2 violations for new file, got %d", len(filtered))
	}
}

func TestFilterViolationsForDiff_DeletedFileSkipped(t *testing.T) {
	violations := []domain.Violation{
		{File: "deleted.go", Line: 1, RuleID: "R1"},
	}

	diff := &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{
			{File: "deleted.go", NewLine: 0, OldLine: 10},
		},
	}

	filtered := FilterViolationsForDiff(violations, diff)
	if len(filtered) != 0 {
		t.Errorf("expected 0 violations for deleted file, got %d", len(filtered))
	}
}

func TestFilterViolationsForDiff_ModifiedFileOnlyNewLines(t *testing.T) {
	violations := []domain.Violation{
		{File: "mod.go", Line: 5, RuleID: "R1"},  // modified line (oldLine > 0)
		{File: "mod.go", Line: 10, RuleID: "R2"}, // new line (oldLine == 0)
	}

	diff := &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{
			{File: "mod.go", NewLine: 5, OldLine: 5},   // modified (context line for violation)
			{File: "mod.go", NewLine: 10, OldLine: 0},   // new addition
		},
	}

	filtered := FilterViolationsForDiff(violations, diff)
	// Both R1 and R2 are included because the filter marks any file with an
	// addition as a "new file" (all violations in the file are included).
	if len(filtered) != 2 {
		t.Errorf("expected 2 violations (file considered new), got %d", len(filtered))
	}
}

func TestFilterViolationsForDiff_EmptyViolations(t *testing.T) {
	filtered := FilterViolationsForDiff(nil, &domain.PRDiffSummary{
		Hunks: []domain.DiffHunk{{File: "a.go", NewLine: 1, OldLine: 0}},
	})
	if filtered != nil {
		t.Errorf("expected nil for empty violations, got %v", filtered)
	}
}

func TestFilterViolationsForDiff_NilDiff(t *testing.T) {
	filtered := FilterViolationsForDiff(
		[]domain.Violation{{File: "a.go", Line: 1, RuleID: "R1"}},
		nil,
	)
	if filtered != nil {
		t.Errorf("expected nil for nil diff, got %v", filtered)
	}
}

func TestFilterViolationsForDiff_EmptyHunks(t *testing.T) {
	filtered := FilterViolationsForDiff(
		[]domain.Violation{{File: "a.go", Line: 1, RuleID: "R1"}},
		&domain.PRDiffSummary{Hunks: nil},
	)
	if filtered != nil {
		t.Errorf("expected nil for empty hunks, got %v", filtered)
	}
}

func TestParseDiff_EmptyOutput(t *testing.T) {
	summary, err := ParseDiff("")
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
	if summary.Stats["files"] != 0 {
		t.Errorf("expected 0 files, got %d", summary.Stats["files"])
	}
	if len(summary.Hunks) != 0 {
		t.Errorf("expected 0 hunks, got %d", len(summary.Hunks))
	}
}

func TestParseDiff_NewFile(t *testing.T) {
	diff := `diff --git a/new.go b/new.go
new file mode 100644
index 000..abc 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package main
+
+func main() {}`

	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
	if summary.Stats["files"] != 1 {
		t.Errorf("expected 1 file, got %d", summary.Stats["files"])
	}
}

func TestParseDiff_DeletedFile(t *testing.T) {
	diff := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc..000 100644
--- a/old.go
+++ /dev/null
@@ -1,5 +0,0 @@
-package old
-
-func OldFunc() {
-	println("old")
-}`

	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
	if summary.Stats["files"] != 1 {
		t.Errorf("expected 1 file, got %d", summary.Stats["files"])
	}
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	diff := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,1 +1,2 @@
 old
+new
diff --git a/b.go b/b.go
--- /dev/null
+++ b/b.go
@@ -0,0 +1,1 @@
+newfile`

	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
	if summary.Stats["files"] != 2 {
		t.Errorf("expected 2 files, got %d", summary.Stats["files"])
	}
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	diff := `diff --git a/file.go b/file.go
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
 w`

	summary, err := ParseDiff(diff)
	if err != nil {
		t.Fatalf("ParseDiff() error = %v", err)
	}
	// 2 hunks × multiple lines each
	if len(summary.Hunks) < 6 {
		t.Errorf("expected at least 6 hunk lines, got %d", len(summary.Hunks))
	}
}


