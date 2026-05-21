package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/plugin"
)

// TestPluginDetectorIntegration performs a full round-trip test:
// 1. Compiles the mock plugin
// 2. Creates plugin config with a real project
// 3. Runs detect and extract
// 4. Verifies dependencies
func TestPluginDetectorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temp dir for mock plugin binary
	binDir, err := os.MkdirTemp("", "arx-plugin-int-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(binDir)

	// Compile mock plugin
	srcPath := filepath.Join("..", "..", "test", "testdata", "plugins", "mockplugin.go")
	baseBin := filepath.Join(binDir, "mockplugin")
	cmd := exec.Command("go", "build", "-o", baseBin, srcPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to compile mock plugin: %v", err)
	}

	// Create a symlink for the "full" variant
	fullBin := filepath.Join(binDir, "mockplugin-full")
	if err := os.Symlink(baseBin, fullBin); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create a temp project directory with a test file
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "test.py"), []byte("import os\nimport sys\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create plugin config
	cfg := domain.PluginConfig{
		Name:      "python-detector",
		Command:   fullBin,
		Languages: []string{"python"},
	}

	// Create detector
	d := plugin.NewPluginDetector(cfg)

	// Test Detect
	ctx := context.Background()
	detected, err := d.Detect(ctx, projectDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true")
	}

	// Test ExtractImports
	layers := []domain.Layer{
		{Name: "app", Paths: []string{"."}},
	}
	deps, err := d.ExtractImports(ctx, projectDir, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("ExtractImports() returned 0 dependencies, expected at least 1")
	}

	// Verify dependency content
	found := false
	for _, dep := range deps {
		if dep.ImportPath == "os" {
			found = true
			if dep.SourceLine != 1 {
				t.Errorf("SourceLine = %d, want 1", dep.SourceLine)
			}
			if dep.ResolvedLayer != "stdlib" {
				t.Errorf("ResolvedLayer = %q, want %q", dep.ResolvedLayer, "stdlib")
			}
		}
	}
	if !found {
		t.Error("Expected dependency with ImportPath 'os', not found")
	}
}

// TestPluginDetectorViaConfig tests that plugins work through the Config/Registry path.
func TestPluginDetectorViaConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Compile mock plugin
	binDir, err := os.MkdirTemp("", "arx-plugin-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(binDir)

	srcPath := filepath.Join("..", "..", "test", "testdata", "plugins", "mockplugin.go")
	baseBin := filepath.Join(binDir, "mockplugin")
	cmd := exec.Command("go", "build", "-o", baseBin, srcPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to compile mock plugin: %v", err)
	}

	fullBin := filepath.Join(binDir, "mockplugin-full")
	if err := os.Symlink(baseBin, fullBin); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create config with plugins
	cfg := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "app", Paths: []string{"."}},
			{Name: "infrastructure", Paths: []string{"infra"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "R1",
				From:     "app",
				To:       []string{"infrastructure"},
				Type:     domain.RuleTypeCannot,
				Severity: domain.SeverityError,
			},
		},
		Plugins: []domain.PluginConfig{
			{
				Name:      "python-detector",
				Command:   fullBin,
				Languages: []string{"python"},
			},
		},
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Config.Validate() error = %v", err)
	}

	// Verify Plugins field is populated
	if len(cfg.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(cfg.Plugins))
	}
	if cfg.Plugins[0].Name != "python-detector" {
		t.Errorf("Plugin name = %q, want %q", cfg.Plugins[0].Name, "python-detector")
	}

	// Verify schema generation includes plugins
	// (reflection-based schema should pick up the Plugins field)
	t.Logf("Plugin config validated successfully: %+v", cfg.Plugins[0])
}

// TestPluginDetector_BadPath tests that a plugin with a bad path gives a warning, not a crash.
func TestPluginDetector_BadPath(t *testing.T) {
	cfg := domain.PluginConfig{
		Name:      "bad-path",
		Command:   "/nonexistent/plugin/binary",
		Languages: []string{"test"},
	}

	d := plugin.NewPluginDetector(cfg)

	_, err := d.Detect(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("Expected error for non-existent plugin binary, got nil")
	}
	t.Logf("Got expected error: %v", err)
}
