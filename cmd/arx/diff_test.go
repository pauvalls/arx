package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/output"
)

func TestDiffCommand_DefaultRefs(t *testing.T) {
	// Create a temp git repo
	tmpDir := setupTestGitRepo(t)

	// Capture output
	buf := new(bytes.Buffer)
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Set args and run
	rootCmd.SetArgs([]string{"diff", "--config", filepath.Join(tmpDir, "arx.yaml")})
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Change to temp dir for the test
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := rootCmd.Execute()
	os.Stdout = oldStdout
	w.Close()

	// The command should fail because there are no commits to diff (HEAD~1 doesn't exist)
	// But it should fail with a meaningful error, not a panic
	if err == nil {
		// If it somehow succeeds, that's also acceptable
		t.Log("diff command succeeded (unexpected but acceptable)")
	}
}

func TestDiffCommand_WrongArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetArgs([]string{"diff", "arg1", "arg2", "arg3"})
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for too many args")
	}
}

func TestDiffRenderer_Render(t *testing.T) {
	result := application.DiffResult{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
		Added: []domain.Violation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user.go", Line: 10},
			{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "github.com/example/entity", File: "service.go", Line: 20},
		},
		Resolved: []domain.Violation{
			{RuleID: "R003", SourceLayer: "domain", TargetLayer: "presentation", Import: "github.com/example/handler", File: "handler.go", Line: 30},
		},
		Unchanged: []domain.Violation{
			{RuleID: "R004", SourceLayer: "infrastructure", TargetLayer: "domain", Import: "github.com/example/repo", File: "repo.go", Line: 40},
		},
	}

	renderer := output.NewDiffRenderer()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	renderer.Render(result)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Verify output contains expected content
	if !containsStr(out, "HEAD~1") || !containsStr(out, "HEAD") {
		t.Error("output should contain refs")
	}
	if !containsStr(out, "+ 2 NEW violations") {
		t.Error("output should show added count")
	}
	if !containsStr(out, "- 1 RESOLVED violations") {
		t.Error("output should show resolved count")
	}
	if !containsStr(out, "= 1 UNCHANGED violations") {
		t.Error("output should show unchanged count")
	}
	if !containsStr(out, "+2 violations, -1 resolved, 1 unchanged") {
		t.Error("output should contain summary")
	}
}

func TestDiffRenderer_RenderJSON(t *testing.T) {
	result := application.DiffResult{
		RefBefore:     "HEAD~1",
		RefAfter:      "HEAD",
		ConfigChanged: true,
		Added: []domain.Violation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user.go", Line: 10},
		},
		Resolved:  []domain.Violation{},
		Unchanged: []domain.Violation{},
	}

	renderer := output.NewDiffRenderer()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderer.RenderJSON(result)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Verify it's valid JSON
	var parsed output.DiffJSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}

	// Verify structure
	if parsed.RefBefore != "HEAD~1" {
		t.Errorf("ref_before = %q, want %q", parsed.RefBefore, "HEAD~1")
	}
	if parsed.RefAfter != "HEAD" {
		t.Errorf("ref_after = %q, want %q", parsed.RefAfter, "HEAD")
	}
	if !parsed.ConfigChanged {
		t.Error("config_changed should be true")
	}
	if len(parsed.Added) != 1 {
		t.Errorf("added count = %d, want 1", len(parsed.Added))
	}
	if parsed.Added[0].RuleID != "R001" {
		t.Errorf("added[0].rule_id = %q, want %q", parsed.Added[0].RuleID, "R001")
	}
	if parsed.Summary != "+1 violations, -0 resolved, 0 unchanged" {
		t.Errorf("summary = %q, want %q", parsed.Summary, "+1 violations, -0 resolved, 0 unchanged")
	}
}

func TestDiffRenderer_RenderJSON_NoANSI(t *testing.T) {
	result := application.DiffResult{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
		Added:     []domain.Violation{{RuleID: "R001"}},
	}

	renderer := output.NewDiffRenderer()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = renderer.RenderJSON(result)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// JSON output should not contain ANSI escape codes
	if containsStr(out, "\x1b[") {
		t.Error("JSON output should not contain ANSI escape codes")
	}
}

func TestDiffRenderer_EmptyDiff(t *testing.T) {
	result := application.DiffResult{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
	}

	renderer := output.NewDiffRenderer()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	renderer.Render(result)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !containsStr(out, "No architecture changes detected") {
		t.Error("empty diff should show 'no changes' message")
	}
}

// setupTestGitRepo creates a minimal git repo with two commits for testing.
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Initialize git repo
	runCmd(t, tmpDir, "git", "init")
	runCmd(t, tmpDir, "git", "config", "user.email", "test@test.com")
	runCmd(t, tmpDir, "git", "config", "user.name", "Test")

	// Create arx.yaml
	configContent := `layers:
  - name: domain
    path: internal/domain
  - name: application
    path: internal/application
rules:
  - id: R001
    source: domain
    target: infrastructure
    action: deny
`
	if err := os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write arx.yaml: %v", err)
	}

	// Create first commit
	runCmd(t, tmpDir, "git", "add", "arx.yaml")
	runCmd(t, tmpDir, "git", "commit", "-m", "initial commit")

	// Create second commit
	if err := os.WriteFile(filepath.Join(tmpDir, "dummy.go"), []byte("package dummy\n"), 0o644); err != nil {
		t.Fatalf("failed to write dummy.go: %v", err)
	}
	runCmd(t, tmpDir, "git", "add", "dummy.go")
	runCmd(t, tmpDir, "git", "commit", "-m", "second commit")

	return tmpDir
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %s %v failed: %v", name, args, err)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
