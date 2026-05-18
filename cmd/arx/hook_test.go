package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// T9: hook install
// ---------------------------------------------------------------------------

func TestHookInstall_CreatesHookFile(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", out)
	}

	err := hookInstallCmd.RunE(hookInstallCmd, []string{tmpDir})
	if err != nil {
		t.Fatalf("hook install failed: %v", err)
	}

	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not created: %v", err)
	}

	// Must be executable
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable (mode 0755)")
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("cannot read hook: %v", err)
	}
	script := string(content)
	if !strings.Contains(script, "arx check") {
		t.Error("hook script must contain 'arx check'")
	}
	if !strings.Contains(script, "SKIP") {
		t.Error("hook script must contain SKIP guard")
	}
	if !strings.Contains(script, "exit 0") {
		t.Error("hook script must contain 'exit 0'")
	}
}

func TestHookInstall_NonGitDir_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()

	err := hookInstallCmd.RunE(hookInstallCmd, []string{tmpDir})
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error must mention 'not a git repository', got: %v", err)
	}
}

func TestHookInstall_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", out)
	}

	// Install twice — must succeed both times
	if err := hookInstallCmd.RunE(hookInstallCmd, []string{tmpDir}); err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if err := hookInstallCmd.RunE(hookInstallCmd, []string{tmpDir}); err != nil {
		t.Fatalf("second install (idempotent) failed: %v", err)
	}

	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("hook should exist after idempotent install")
	}
}

// ---------------------------------------------------------------------------
// T10: hook uninstall
// ---------------------------------------------------------------------------

func TestHookUninstall_RemovesHookFile(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", out)
	}

	// Install first
	if err := hookInstallCmd.RunE(hookInstallCmd, []string{tmpDir}); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		t.Fatal("hook must exist after install")
	}

	// Uninstall
	if err := hookUninstallCmd.RunE(hookUninstallCmd, []string{tmpDir}); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook must be removed after uninstall")
	}
}

func TestHookUninstall_NoHookExists_Graceful(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", out)
	}

	// Uninstall without a prior install — must NOT error
	if err := hookUninstallCmd.RunE(hookUninstallCmd, []string{tmpDir}); err != nil {
		t.Fatalf("uninstall without hook must not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T11: hook script content verification
// ---------------------------------------------------------------------------

func TestHookScript_ContainsExpectedContent(t *testing.T) {
	checks := []struct {
		name    string
		pattern string
	}{
		{"shebang", "#!/bin/sh"},
		{"SKIP guard", `grep -q arx`},
		{"git rev-parse for root", "git rev-parse --show-toplevel"},
		{"arx check with --no-cache", "arx check --no-cache"},
		{"capture exit code", "exit_code=$?"},
		{"violation message", "Architecture violation(s) found"},
		{"pass exit code", "exit $exit_code"},
		{"shebang as first line", "#!/bin/sh"},
	}
	for _, c := range checks {
		if !strings.Contains(hookScript, c.pattern) {
			t.Errorf("hookScript must contain %q (%s)", c.pattern, c.name)
		}
	}
}
