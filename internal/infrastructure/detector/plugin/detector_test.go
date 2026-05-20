package plugin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

// compileMockPluginForDetector compiles the mock plugin and returns the binary dir.
func compileMockPluginForDetector(t *testing.T) (binaryDir string, cleanup func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "arx-plugin-detector-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	srcPath := filepath.Join("..", "..", "..", "..", "test", "testdata", "plugins", "mockplugin.go")
	baseBin := filepath.Join(dir, "mockplugin-base")
	cmd := exec.Command("go", "build", "-o", baseBin, srcPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to compile mock plugin: %v", err)
	}

	variants := []string{"detect-only", "full", "slow", "error"}
	for _, v := range variants {
		linkPath := filepath.Join(dir, "mockplugin-"+v)
		if err := os.Symlink(baseBin, linkPath); err != nil {
			os.RemoveAll(dir)
			t.Fatalf("failed to create symlink %s: %v", linkPath, err)
		}
	}

	cleanup = func() { os.RemoveAll(dir) }
	return dir, cleanup
}

func TestPluginDetector_Detect_Success(t *testing.T) {
	dir, cleanup := compileMockPluginForDetector(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "mock-detect",
		Command:   filepath.Join(dir, "mockplugin-detect-only"),
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	detected, err := d.Detect(context.Background(), "/tmp/test")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true")
	}
}

func TestPluginDetector_Detect_FromFullPlugin(t *testing.T) {
	dir, cleanup := compileMockPluginForDetector(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "mock-full",
		Command:   filepath.Join(dir, "mockplugin-full"),
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	detected, err := d.Detect(context.Background(), "/tmp/test")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true")
	}
}

func TestPluginDetector_Extract_Success(t *testing.T) {
	dir, cleanup := compileMockPluginForDetector(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "mock-full",
		Command:   filepath.Join(dir, "mockplugin-full"),
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	layers := []domain.Layer{
		{Name: "app", Paths: []string{"src"}},
	}
	deps, err := d.ExtractImports(context.Background(), "/tmp/test", layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("ExtractImports() returned %d deps, want 1", len(deps))
	}
	if deps[0].SourceFile != "test.py" {
		t.Errorf("SourceFile = %q, want %q", deps[0].SourceFile, "test.py")
	}
	if deps[0].ImportPath != "os" {
		t.Errorf("ImportPath = %q, want %q", deps[0].ImportPath, "os")
	}
	if deps[0].ResolvedLayer != "stdlib" {
		t.Errorf("ResolvedLayer = %q, want %q", deps[0].ResolvedLayer, "stdlib")
	}
}

func TestPluginDetector_Name(t *testing.T) {
	cfg := domain.PluginConfig{
		Name:      "my-detector",
		Command:   "/bin/true",
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	if name := d.Name(); name != "my-detector" {
		t.Errorf("Name() = %q, want %q", name, "my-detector")
	}
}

func TestPluginDetector_ErrorResponse(t *testing.T) {
	dir, cleanup := compileMockPluginForDetector(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "mock-error",
		Command:   filepath.Join(dir, "mockplugin-error"),
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	_, err := d.Detect(context.Background(), "/tmp/test")
	if err == nil {
		t.Fatal("Detect() expected error, got nil")
	}
	if !containsStr(err.Error(), "mock plugin error") {
		t.Errorf("Detect() error = %q, want it to contain 'mock plugin error'", err.Error())
	}
}

func TestPluginDetector_Timeout(t *testing.T) {
	dir, cleanup := compileMockPluginForDetector(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "mock-slow",
		Command:   filepath.Join(dir, "mockplugin-slow"),
		Languages: []string{"mock"},
		Timeout:   "1s",
	}
	d := NewPluginDetector(cfg)

	_, err := d.Detect(context.Background(), "/tmp/test")
	if err == nil {
		t.Fatal("Detect() expected timeout error, got nil")
	}
	if !containsStr(err.Error(), "timeout") {
		t.Errorf("Detect() error = %q, want it to contain 'timeout'", err.Error())
	}
}

func TestPluginDetector_BinaryNotFound(t *testing.T) {
	cfg := domain.PluginConfig{
		Name:      "missing",
		Command:   "/nonexistent/binary",
		Languages: []string{"mock"},
	}
	d := NewPluginDetector(cfg)

	_, err := d.Detect(context.Background(), "/tmp/test")
	if err == nil {
		t.Fatal("Detect() expected error for missing binary, got nil")
	}
}
