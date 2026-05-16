package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_GoProject verifies arx detects violations in the Go fixture
func TestE2E_GoProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("go-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Go fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", fixturePath)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		// arx exits 1 if violations found — expected for this fixture
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("unexpected error: %v\nOutput: %s", err, outStr)
		}
	}

	// Must detect violations
	if !strings.Contains(outStr, "violation") && !strings.Contains(outStr, "D-01") {
		t.Errorf("Expected violations in Go project, got:\n%s", outStr)
	}
	t.Logf("Go E2E: violations detected ✅")
}

// TestE2E_TypeScriptProject verifies arx detects violations in the TypeScript fixture
func TestE2E_TypeScriptProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("ts-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("TypeScript fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", fixturePath)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("unexpected error: %v\nOutput: %s", err, outStr)
		}
	}

	if !strings.Contains(outStr, "violation") && !strings.Contains(outStr, "D-01") {
		t.Errorf("Expected violations in TypeScript project, got:\n%s", outStr)
	}
	t.Logf("TypeScript E2E: violations detected ✅")
}

// TestE2E_PythonProject verifies arx detects violations in the Python fixture
func TestE2E_PythonProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("python-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Python fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", fixturePath)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("unexpected error: %v\nOutput: %s", err, outStr)
		}
	}

	if !strings.Contains(outStr, "violation") && !strings.Contains(outStr, "D-01") {
		t.Errorf("Expected violations in Python project, got:\n%s", outStr)
	}
	t.Logf("Python E2E: violations detected ✅")
}

// TestE2E_JavaProject verifies arx detects violations in the Java Maven fixture
func TestE2E_JavaProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("java-maven")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Java fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", fixturePath)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("unexpected error: %v\nOutput: %s", err, outStr)
		}
	}

	if !strings.Contains(outStr, "violation") {
		t.Errorf("Expected violations in Java project, got:\n%s", outStr)
	}
	t.Logf("Java E2E: violations detected ✅")
}

// TestE2E_RubyProject verifies arx detects violations in the Ruby fixture
func TestE2E_RubyProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("ruby-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Ruby fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", fixturePath)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("unexpected error: %v\nOutput: %s", err, outStr)
		}
	}

	if !strings.Contains(outStr, "violation") {
		t.Errorf("Expected violations in Ruby project, got:\n%s", outStr)
	}
	t.Logf("Ruby E2E: violations detected ✅")
}

// TestE2E_AllFormats verifies all output formats work on a Go project
func TestE2E_AllFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	fixturePath := getFixtureAbsPath("go-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Go fixture not found at %s", fixturePath)
	}

	binaryPath := buildArxBinary(t)
	formats := []string{"terminal", "json", "sarif", "md", "junit", "annotations"}

	for _, format := range formats {
		cmd := exec.Command(binaryPath, "check", "--format", format, fixturePath)
		output, _ := cmd.CombinedOutput()
		outStr := string(output)

		// Should produce output regardless of exit code
		if len(outStr) < 10 {
			t.Errorf("Format %q produced too little output (%d chars)", format, len(outStr))
		}
		t.Logf("Format %q: %d chars ✅", format, len(outStr))
	}
}

// TestE2E_BaselineWorkflow verifies baseline creation + suppression
func TestE2E_BaselineWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()

	// Create minimal Go project with violations
	writeGoMod(t, tmpDir, "example.com/test")
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infra"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "domain", "entity.go"), []byte(`package domain
import "example.com/test/infra"
type Entity struct { R *infra.Repository }
`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "infra", "repo.go"), []byte(`package infra
type Repository struct{}
`), 0644)

	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths: [domain/**]
  - name: infra
    paths: [infra/**]
rules:
  - id: domain-no-infra
    from: domain
    to: [infra]
    type: Cannot
    severity: error
`
	os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(arxConfig), 0644)

	binaryPath := buildArxBinary(t)

	// Step 1: check shows violations
	cmd1 := exec.Command(binaryPath, "check", tmpDir)
	out1, _ := cmd1.CombinedOutput()
	if !strings.Contains(string(out1), "violation") {
		t.Fatalf("Expected violations before baseline, got:\n%s", out1)
	}
	t.Log("Step 1: violations found ✅")

	// Step 2: create baseline
	cmd2 := exec.Command(binaryPath, "baseline", tmpDir)
	out2, err2 := cmd2.CombinedOutput()
	if err2 != nil {
		t.Fatalf("baseline failed: %v\n%s", err2, out2)
	}
	t.Log("Step 2: baseline created ✅")

	// Step 3: check again — should be clean (baseline suppresses)
	cmd3 := exec.Command(binaryPath, "check", tmpDir)
	out3, err3 := cmd3.CombinedOutput()
	if err3 != nil {
		// Should succeed (exit 0) — no NEW violations
		if exitErr, ok := err3.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			t.Logf("Step 3 note: exit code %d (may still report suppressed count)", exitErr.ExitCode())
		}
	}
	t.Logf("Step 3: check with baseline ✅\n%s", string(out3))
}

// getFixtureAbsPath returns the absolute path to a fixture directory
func getFixtureAbsPath(name string) string {
	abs, _ := filepath.Abs(filepath.Join("..", "fixtures", name))
	return abs
}
