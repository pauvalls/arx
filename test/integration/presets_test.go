package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitWithPreset_Clean tests that arx init --preset clean generates a valid config
func TestInitWithPreset_Clean(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with clean preset
	cmd := exec.Command(binaryPath, "init", "--preset", "clean")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("arx init failed: %v\nOutput: %s", err, output)
	}

	// Verify arx.yaml was created
	arxYamlPath := filepath.Join(tmpDir, "arx.yaml")
	content, err := os.ReadFile(arxYamlPath)
	if err != nil {
		t.Fatalf("failed to read arx.yaml: %v", err)
	}

	// Verify content has expected layers
	contentStr := string(content)
	expectedLayers := []string{"domain", "application", "infrastructure", "presentation"}
	for _, layer := range expectedLayers {
		if !strings.Contains(contentStr, "- name: "+layer) {
			t.Errorf("arx.yaml missing layer: %s", layer)
		}
	}

	// Verify header comment
	if !strings.HasPrefix(contentStr, "# Arx Architecture Configuration") {
		t.Error("arx.yaml missing header comment")
	}
}

// TestInitWithPreset_Hexagonal tests that arx init --preset hexagonal generates a valid config
func TestInitWithPreset_Hexagonal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with hexagonal preset
	cmd := exec.Command(binaryPath, "init", "--preset", "hexagonal")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("arx init failed: %v\nOutput: %s", err, output)
	}

	// Verify arx.yaml was created
	arxYamlPath := filepath.Join(tmpDir, "arx.yaml")
	content, err := os.ReadFile(arxYamlPath)
	if err != nil {
		t.Fatalf("failed to read arx.yaml: %v", err)
	}

	// Verify content has expected layers (hexagonal should have ports and adapters)
	contentStr := string(content)
	expectedLayers := []string{"domain", "ports", "adapters", "infrastructure"}
	for _, layer := range expectedLayers {
		if !strings.Contains(contentStr, "- name: "+layer) {
			t.Errorf("arx.yaml missing layer: %s", layer)
		}
	}
}

// TestInitWithPreset_Ddd tests that arx init --preset ddd generates a valid config
func TestInitWithPreset_Ddd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with ddd preset
	cmd := exec.Command(binaryPath, "init", "--preset", "ddd")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("arx init failed: %v\nOutput: %s", err, output)
	}

	// Verify arx.yaml was created
	arxYamlPath := filepath.Join(tmpDir, "arx.yaml")
	content, err := os.ReadFile(arxYamlPath)
	if err != nil {
		t.Fatalf("failed to read arx.yaml: %v", err)
	}

	// Verify content has expected layers (DDD should have interfaces layer)
	contentStr := string(content)
	expectedLayers := []string{"domain", "application", "infrastructure", "interfaces"}
	for _, layer := range expectedLayers {
		if !strings.Contains(contentStr, "- name: "+layer) {
			t.Errorf("arx.yaml missing layer: %s", layer)
		}
	}
}

// TestInitWithPreset_InvalidPreset tests that invalid preset names are rejected
func TestInitWithPreset_InvalidPreset(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with invalid preset
	cmd := exec.Command(binaryPath, "init", "--preset", "invalid")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("arx init with invalid preset should have failed")
	}

	// Verify error message mentions available presets
	outputStr := string(output)
	if !strings.Contains(outputStr, "unknown preset") {
		t.Errorf("expected 'unknown preset' error, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "clean") || !strings.Contains(outputStr, "hexagonal") || !strings.Contains(outputStr, "ddd") {
		t.Errorf("error message should list available presets, got: %s", outputStr)
	}
}

// TestInitWithPreset_WithOutputFlag tests that --output flag works with presets
func TestInitWithPreset_WithOutputFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create config directory
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with custom output path
	outputPath := filepath.Join("config", "arx.yaml")
	cmd := exec.Command(binaryPath, "init", "--preset", "clean", "--output", outputPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("arx init failed: %v\nOutput: %s", err, output)
	}

	// Verify arx.yaml was created at custom path
	arxYamlPath := filepath.Join(tmpDir, outputPath)
	if _, err := os.Stat(arxYamlPath); os.IsNotExist(err) {
		t.Fatalf("arx.yaml not created at custom path: %s", outputPath)
	}
}

// TestInitWithPreset_ForceOverwrite tests that --force flag works with presets
func TestInitWithPreset_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create initial arx.yaml
	initialContent := []byte("# initial config\nversion: \"1.0\"\nlayers: []\n")
	arxYamlPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(arxYamlPath, initialContent, 0644); err != nil {
		t.Fatalf("failed to create initial arx.yaml: %v", err)
	}

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx init with --force
	cmd := exec.Command(binaryPath, "init", "--preset", "clean", "--force")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("arx init --force failed: %v\nOutput: %s", err, output)
	}

	// Verify arx.yaml was overwritten (should have clean preset content)
	content, err := os.ReadFile(arxYamlPath)
	if err != nil {
		t.Fatalf("failed to read arx.yaml: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "domain") {
		t.Error("arx.yaml was not overwritten with preset content")
	}
}

// buildArxBinary builds the arx CLI binary and returns its path
func buildArxBinary(t *testing.T) string {
	t.Helper()

	// Get project root (parent of test directory)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("failed to get project root: %v", err)
	}

	// Create temp dir for binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "arx")

	// Build binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/arx")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build arx: %v\nOutput: %s", err, output)
	}

	return binaryPath
}
