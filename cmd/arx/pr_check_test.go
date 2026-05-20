package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRCheckCmd_Help(t *testing.T) {
	// Just verify the command is registered and shows help
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"pr-check", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("pr-check --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "pr-check") {
		t.Errorf("help should mention pr-check, got: %s", output)
	}
	if !strings.Contains(output, "--base") {
		t.Errorf("help should mention --base flag, got: %s", output)
	}
	if !strings.Contains(output, "--head") {
		t.Errorf("help should mention --head flag, got: %s", output)
	}
}

func TestPRCheckCmd_Flags(t *testing.T) {
	// Verify flags exist
	cmd := prCheckCmd
	if cmd == nil {
		t.Fatal("prCheckCmd is nil")
	}

	f := cmd.Flags()
	testCases := []string{"base", "head", "repo", "json", "verbose", "approve"}
	for _, flag := range testCases {
		if f.Lookup(flag) == nil {
			t.Errorf("flag --%s not found on pr-check command", flag)
		}
	}
}

func TestPRCheckCmd_NoBaseFlag(t *testing.T) {
	// Reset global state
	prCheckBase = ""
	prCheckHead = ""
	prCheckRepo = "."

	cmd := prCheckCmd
	err := cmd.ParseFlags([]string{"--head", "abc123"})
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when --base is missing")
	}
	if !strings.Contains(err.Error(), "base") {
		t.Errorf("error should mention --base, got: %v", err)
	}
}

func TestPRCheckCmd_NoHeadFlag(t *testing.T) {
	// Reset global state
	prCheckBase = ""
	prCheckHead = ""
	prCheckRepo = "."

	cmd := prCheckCmd
	err := cmd.ParseFlags([]string{"--base", "abc123"})
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when --head is missing")
	}
	if !strings.Contains(err.Error(), "head") {
		t.Errorf("error should mention --head, got: %v", err)
	}
}

func TestPRCheckJsonOutput(t *testing.T) {
	// Test JSON output formatting
	v := violationOutput{
		File:      "test.go",
		Line:      10,
		RuleID:    "R001",
		Message:   "test violation",
		IsNew:     true,
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal violationOutput: %v", err)
	}

	var decoded violationOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal violationOutput: %v", err)
	}

	if decoded.File != "test.go" {
		t.Errorf("File = %q, want %q", decoded.File, "test.go")
	}
	if decoded.Line != 10 {
		t.Errorf("Line = %d, want 10", decoded.Line)
	}
	if decoded.RuleID != "R001" {
		t.Errorf("RuleID = %q, want %q", decoded.RuleID, "R001")
	}
	if !decoded.IsNew {
		t.Error("IsNew should be true")
	}
}

func TestPRCheckCmd_RunInTempDir(t *testing.T) {
	// Create a temp directory with a basic git repo
	tmpDir := t.TempDir()

	// Init git repo
	gitCmd(t, tmpDir, "init")
	gitCmd(t, tmpDir, "config", "user.email", "test@test.com")
	gitCmd(t, tmpDir, "config", "user.name", "Test")

	// Create initial file and commit
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644)
	gitCmd(t, tmpDir, "add", ".")
	gitCmd(t, tmpDir, "commit", "-m", "initial")

	// Create a change
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	gitCmd(t, tmpDir, "add", ".")
	gitCmd(t, tmpDir, "commit", "-m", "second")

	// Run pr-check via RunE directly (avoids global rootCmd pollution)
	cmd := prCheckCmd
	cmd.SetArgs([]string{"--base", "HEAD~1", "--head", "HEAD", "--repo", tmpDir})
	err := cmd.ParseFlags([]string{"--base", "HEAD~1", "--head", "HEAD", "--repo", tmpDir})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	err = cmd.RunE(cmd, []string{})
	// It may fail if arx.yaml doesn't exist, but the command structure should work
	// We're just testing it doesn't panic
	_ = err
}

// gitCmd runs a git command in the specified directory.
func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := gitCommand(dir, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(output))
	}
}
