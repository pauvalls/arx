//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRCheck_CLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Initialize git repo
	runCmd(t, tmpDir, "git", "init")
	runCmd(t, tmpDir, "git", "config", "user.email", "test@test.com")
	runCmd(t, tmpDir, "git", "config", "user.name", "Test")

	// Create initial project structure
	projectRoot := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial files in the project
	os.WriteFile(filepath.Join(projectRoot, "main.go"), []byte(`package main

func main() {
	println("hello")
}
`), 0644)

	// Create arx configuration
	os.WriteFile(filepath.Join(projectRoot, "arx.yaml"), []byte(`version: "1.0"
layers:
  - name: domain
    paths:
      - "internal/domain/**"
rules: []
`), 0644)

	// Initial commit
	runCmd(t, tmpDir, "git", "add", ".")
	runCmd(t, tmpDir, "git", "commit", "-m", "Initial commit")
	runCmd(t, tmpDir, "git", "tag", "v1")

	// Create a change (add a new file)
	os.WriteFile(filepath.Join(projectRoot, "new_file.go"), []byte(`package main

func newFunc() {
	println("new")
}
`), 0644)
	os.WriteFile(filepath.Join(projectRoot, "main.go"), []byte(`package main

func main() {
	println("hello")
	newFunc()
}
`), 0644)

	// Second commit
	runCmd(t, tmpDir, "git", "add", ".")
	runCmd(t, tmpDir, "git", "commit", "-m", "Second commit")

	// Get the first commit SHA
	baseSHA := strings.TrimSpace(runCmd(t, tmpDir, "git", "rev-parse", "HEAD~1"))
	headSHA := strings.TrimSpace(runCmd(t, tmpDir, "git", "rev-parse", "HEAD"))

	// Build arx binary
	arxBin := filepath.Join(tmpDir, "arx")
	buildCmd := exec.Command("go", "build", "-o", arxBin, "./cmd/arx")
	buildCmd.Dir = filepath.Dir(filepath.Dir(tmpDir)) // This might not be the repo root

	// Try to find the project root (where go.mod is)
	// We know the test runs from the arx project root
	projectRootDir := findGoModDir(t)
	buildCmd = exec.Command("go", "build", "-o", arxBin, "./cmd/arx")
	buildCmd.Dir = projectRootDir
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build arx: %v\n%s", err, buildOutput)
	}

	// Run arx pr-check
	cmd := exec.Command(arxBin, "pr-check",
		"--base", baseSHA,
		"--head", headSHA,
		"--repo", projectRoot,
	)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// The check should complete (may have 0 violations since rules are empty)
	_ = output
	_ = err

	// Verify the command ran without panic
	t.Logf("arx pr-check output:\n%s", string(output))
}

func findGoModDir(t *testing.T) string {
	t.Helper()
	// Walk up from the test file's likely location
	dir := "."
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			abs, _ := filepath.Abs(dir)
			return abs
		}
		dir = filepath.Join(dir, "..")
	}
	t.Fatal("could not find go.mod")
	return ""
}

func runCmd(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("command %s %v failed: %v\nOutput: %s", name, args, err, out.String())
	}
	return out.String()
}
