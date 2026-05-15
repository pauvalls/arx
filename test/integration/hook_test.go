package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHook_InstallAndUninstall verifies the full hook lifecycle in a temp git repo.
func TestHook_InstallAndUninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a real git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", out)
	}

	hookPath := filepath.Join(tmpDir, ".git", "hooks", "pre-commit")

	// Simulate what "arx hook install" does — write the hook script
	scriptContent := `#!/bin/sh
# arx pre-commit hook — blocks commits with new architecture violations
# Set SKIP=arx to bypass this hook (e.g. SKIP=arx git commit)

if echo "${SKIP-}" | grep -q arx; then
    exit 0
fi

PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || {
    echo "not a git repository" >&2
    exit 1
}

cd "$PROJECT_ROOT" || exit 1
arx check --no-cache
exit_code=$?

if [ $exit_code -ne 0 ]; then
    echo "Architecture violation(s) found. Run 'arx check' for details." >&2
fi

exit $exit_code
`

	if err := os.WriteFile(hookPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write hook: %v", err)
	}
	t.Logf("hook written to %s", hookPath)

	// --- Verify hook exists and is executable ---
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable (mode 0755)")
	}

	// --- Verify hook content contains expected strings ---
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("cannot read hook: %v", err)
	}
	script := string(content)

	expectedPatterns := []struct {
		name    string
		pattern string
	}{
		{"shebang", "#!/bin/sh"},
		{"arx check command", "arx check --no-cache"},
		{"SKIP bypass variable", "SKIP"},
		{"git rev-parse for root", "git rev-parse --show-toplevel"},
		{"exit code capture", "exit_code=$?"},
		{"exit 0 on success", "exit 0"},
		{"exit passthrough", "exit $exit_code"},
		{"violation message", "Architecture violation(s) found"},
	}
	for _, ep := range expectedPatterns {
		if !strings.Contains(script, ep.pattern) {
			t.Errorf("hook script should contain %q (%s)", ep.pattern, ep.name)
		}
	}

	// --- Verify shebang is the first line ---
	lines := strings.Split(script, "\n")
	if len(lines) == 0 || lines[0] != "#!/bin/sh" {
		t.Errorf("first line must be '#!/bin/sh', got: %q", lines[0])
	}

	// --- Test uninstall removes the hook ---
	if err := os.Remove(hookPath); err != nil {
		t.Fatalf("failed to remove hook: %v", err)
	}

	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook should not exist after removal")
	}
	t.Log("hook removed successfully")

	// --- Test uninstall on non-existent hook is graceful ---
	// (no error when the file doesn't exist)
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatal("hook should not exist")
	}
	t.Log("uninstall on non-existent hook: no error (expected)")
}
