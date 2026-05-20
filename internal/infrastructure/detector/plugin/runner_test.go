package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

// compileMockPlugin compiles the mock plugin source and creates symlinks with different names.
// Returns a cleanup function and the directory containing the binaries.
func compileMockPlugin(t *testing.T) (binaryDir string, cleanup func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "arx-plugin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Find the mock plugin source relative to this test file
	// We're in internal/infrastructure/detector/plugin/
	// The source is in test/testdata/plugins/mockplugin.go
	srcPath := filepath.Join("..", "..", "..", "..", "test", "testdata", "plugins", "mockplugin.go")

	// Compile the base binary
	baseBin := filepath.Join(dir, "mockplugin-base")
	cmd := exec.Command("go", "build", "-o", baseBin, srcPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to compile mock plugin: %v", err)
	}

	// Create symlinks for different behaviors
	variants := []string{"detect-only", "full", "slow", "error"}
	for _, v := range variants {
		linkPath := filepath.Join(dir, "mockplugin-"+v)
		if err := os.Symlink(baseBin, linkPath); err != nil {
			os.RemoveAll(dir)
			t.Fatalf("failed to create symlink %s: %v", linkPath, err)
		}
	}

	cleanup = func() {
		os.RemoveAll(dir)
	}
	return dir, cleanup
}

func TestRunPlugin_DetectOnly(t *testing.T) {
	dir, cleanup := compileMockPlugin(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "detect-test",
		Command:   filepath.Join(dir, "mockplugin-detect-only"),
		Languages: []string{"mock"},
	}

	req := domain.PluginRequest{
		Action:      "detect",
		ProjectRoot: "/tmp/test",
	}

	resp, err := RunPlugin(cfg, req)
	if err != nil {
		t.Fatalf("RunPlugin() error = %v", err)
	}
	if resp.Detect == nil {
		t.Fatal("RunPlugin() response.Detect is nil")
	}
	if !resp.Detect.Detected {
		t.Error("RunPlugin() Detect.Detected = false, want true")
	}
}

func TestRunPlugin_FullExtract(t *testing.T) {
	dir, cleanup := compileMockPlugin(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "full-test",
		Command:   filepath.Join(dir, "mockplugin-full"),
		Languages: []string{"python"},
	}

	req := domain.PluginRequest{
		Action:      "extract",
		ProjectRoot: "/tmp/test",
		Layers: []domain.LayerInfo{
			{Name: "app", Paths: []string{"src"}},
		},
	}

	resp, err := RunPlugin(cfg, req)
	if err != nil {
		t.Fatalf("RunPlugin() error = %v", err)
	}
	if resp.Extract == nil {
		t.Fatal("RunPlugin() response.Extract is nil")
	}
	if len(resp.Extract.Dependencies) != 1 {
		t.Fatalf("RunPlugin() Dependencies length = %d, want 1", len(resp.Extract.Dependencies))
	}
	dep := resp.Extract.Dependencies[0]
	if dep.SourceFile != "test.py" {
		t.Errorf("SourceFile = %q, want %q", dep.SourceFile, "test.py")
	}
	if dep.ImportPath != "os" {
		t.Errorf("ImportPath = %q, want %q", dep.ImportPath, "os")
	}
}

func TestRunPlugin_Timeout(t *testing.T) {
	dir, cleanup := compileMockPlugin(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "slow-test",
		Command:   filepath.Join(dir, "mockplugin-slow"),
		Languages: []string{"mock"},
		Timeout:   "1s",
	}

	req := domain.PluginRequest{
		Action:      "detect",
		ProjectRoot: "/tmp/test",
	}

	_, err := RunPlugin(cfg, req)
	if err == nil {
		t.Fatal("RunPlugin() expected timeout error, got nil")
	}
	if !isTimeoutError(err) {
		t.Errorf("RunPlugin() error = %v, expected timeout error", err)
	}
}

func TestRunPlugin_ErrorResponse(t *testing.T) {
	dir, cleanup := compileMockPlugin(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "error-test",
		Command:   filepath.Join(dir, "mockplugin-error"),
		Languages: []string{"mock"},
	}

	req := domain.PluginRequest{
		Action:      "detect",
		ProjectRoot: "/tmp/test",
	}

	resp, err := RunPlugin(cfg, req)
	if err != nil {
		t.Fatalf("RunPlugin() error = %v", err)
	}
	if resp.Error == nil {
		t.Fatal("RunPlugin() response.Error is nil, expected error response")
	}
	if resp.Error.Message != "mock plugin error" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "mock plugin error")
	}
}

func TestRunPlugin_NonExistentBinary(t *testing.T) {
	cfg := domain.PluginConfig{
		Name:      "missing",
		Command:   "/nonexistent/binary",
		Languages: []string{"mock"},
	}

	req := domain.PluginRequest{
		Action:      "detect",
		ProjectRoot: "/tmp/test",
	}

	_, err := RunPlugin(cfg, req)
	if err == nil {
		t.Fatal("RunPlugin() expected error for nonexistent binary, got nil")
	}
}

func TestRunPlugin_Capabilities(t *testing.T) {
	dir, cleanup := compileMockPlugin(t)
	defer cleanup()

	cfg := domain.PluginConfig{
		Name:      "full-test",
		Command:   filepath.Join(dir, "mockplugin-full"),
		Languages: []string{"python"},
	}

	caps, err := GetCapabilities(cfg)
	if err != nil {
		t.Fatalf("GetCapabilities() error = %v", err)
	}
	if caps.Name != "full-mock" {
		t.Errorf("Name = %q, want %q", caps.Name, "full-mock")
	}
	if len(caps.Languages) != 2 || caps.Languages[0] != "python" {
		t.Errorf("Languages = %v, want [python ruby]", caps.Languages)
	}
	if caps.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", caps.Version, "2.0.0")
	}
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsStr(errStr, "timeout") || containsStr(errStr, "deadline")
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
