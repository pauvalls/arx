package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

// mockFileWriter implements ports.FileWriter for testing
type mockFileWriter struct {
	files  map[string][]byte
	exists map[string]bool
	err    error
}

func newMockFileWriter() *mockFileWriter {
	return &mockFileWriter{
		files:  make(map[string][]byte),
		exists: make(map[string]bool),
	}
}

func (m *mockFileWriter) Write(path string, content []byte) error {
	if m.err != nil {
		return m.err
	}
	m.files[path] = content
	return nil
}

func (m *mockFileWriter) Exists(path string) bool {
	return m.exists[path]
}

func TestScanProject_DetectsGo(t *testing.T) {
	// Create a temporary directory with go.mod
	tmpDir := t.TempDir()
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	info, err := ScanProject(tmpDir)
	if err != nil {
		t.Fatalf("ScanProject() error = %v", err)
	}

	found := false
	for _, lang := range info.Languages {
		if lang == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ScanProject() did not detect Go; languages = %v", info.Languages)
	}
}

func TestScanProject_DetectsTypeScript(t *testing.T) {
	// Create a temporary directory with package.json
	tmpDir := t.TempDir()
	packageJSONPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}

	info, err := ScanProject(tmpDir)
	if err != nil {
		t.Fatalf("ScanProject() error = %v", err)
	}

	found := false
	for _, lang := range info.Languages {
		if lang == "typescript" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ScanProject() did not detect TypeScript; languages = %v", info.Languages)
	}
}

func TestScanProject_DetectsTypeScriptWithTSConfig(t *testing.T) {
	// Create a temporary directory with tsconfig.json
	tmpDir := t.TempDir()
	tsConfigPath := filepath.Join(tmpDir, "tsconfig.json")
	if err := os.WriteFile(tsConfigPath, []byte(`{"compilerOptions":{}}`), 0644); err != nil {
		t.Fatalf("failed to create tsconfig.json: %v", err)
	}

	info, err := ScanProject(tmpDir)
	if err != nil {
		t.Fatalf("ScanProject() error = %v", err)
	}

	found := false
	for _, lang := range info.Languages {
		if lang == "typescript" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ScanProject() did not detect TypeScript via tsconfig.json; languages = %v", info.Languages)
	}
}

func TestScanProject_SuggestsLayers(t *testing.T) {
	// Create a temporary directory with typical Go structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create layer directories
	for _, dir := range []string{"internal/domain", "internal/application", "internal/infrastructure"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	info, err := ScanProject(tmpDir)
	if err != nil {
		t.Fatalf("ScanProject() error = %v", err)
	}

	if len(info.SuggestedLayers) == 0 {
		t.Errorf("ScanProject() did not suggest any layers")
	}

	// Check that domain layer is suggested
	hasDomain := false
	for _, layer := range info.SuggestedLayers {
		if layer.Name == "domain" {
			hasDomain = true
			break
		}
	}
	if !hasDomain {
		t.Errorf("ScanProject() did not suggest domain layer; layers = %v", layerNames(info.SuggestedLayers))
	}
}

func TestScanProject_NoLanguages(t *testing.T) {
	tmpDir := t.TempDir()

	info, err := ScanProject(tmpDir)
	if err != nil {
		t.Fatalf("ScanProject() error = %v", err)
	}

	if len(info.Languages) != 0 {
		t.Errorf("ScanProject() detected languages in empty dir: %v", info.Languages)
	}
}

func TestGenerateConfig_CreatesDefaultRules(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
			{Name: "presentation", Paths: []string{"cmd/**"}},
		},
	}

	config, err := GenerateConfig(info)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if len(config.Rules) < 5 {
		t.Errorf("GenerateConfig() created %d rules, want at least 5", len(config.Rules))
	}

	// Validate the config
	if err := config.Validate(); err != nil {
		t.Errorf("Generated config is invalid: %v", err)
	}
}

func TestGenerateConfig_FallbackLayers(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{},
	}

	config, err := GenerateConfig(info)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if len(config.Layers) == 0 {
		t.Errorf("GenerateConfig() did not create fallback layers")
	}

	if len(config.Rules) < 5 {
		t.Errorf("GenerateConfig() created %d rules, want at least 5", len(config.Rules))
	}
}

func TestGenerateConfig_NilProjectInfo(t *testing.T) {
	_, err := GenerateConfig(nil)
	if err == nil {
		t.Errorf("GenerateConfig(nil) should return error")
	}
}

func TestGenerateConfig_Excludes(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
		},
	}

	config, err := GenerateConfig(info)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if len(config.Exclude) == 0 {
		t.Errorf("GenerateConfig() did not set excludes")
	}

	expectedExcludes := []string{"vendor/**", "node_modules/**", ".git/**"}
	for _, expected := range expectedExcludes {
		found := false
		for _, ex := range config.Exclude {
			if ex == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GenerateConfig() missing exclude: %s", expected)
		}
	}
}

func TestGenerateConfig_LanguageOverrides(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go", "typescript"},
		SuggestedLayers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
		},
	}

	config, err := GenerateConfig(info)
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	if config.LanguageOverrides == nil {
		t.Fatalf("GenerateConfig() did not create language overrides")
	}

	if _, ok := config.LanguageOverrides["go"]; !ok {
		t.Errorf("GenerateConfig() missing Go language override")
	}
	if _, ok := config.LanguageOverrides["typescript"]; !ok {
		t.Errorf("GenerateConfig() missing TypeScript language override")
	}
}

func TestWriteConfig_Success(t *testing.T) {
	writer := newMockFileWriter()
	config := &domain.Config{
		Version: "1.0",
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
		},
		Rules: []domain.Rule{
			{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: domain.RuleTypeCannot},
		},
	}

	err := WriteConfig(config, "arx.yaml", writer)
	if err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	content, ok := writer.files["arx.yaml"]
	if !ok {
		t.Fatalf("WriteConfig() did not write to arx.yaml")
	}

	// Verify it's valid YAML
	var parsed domain.Config
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Errorf("WriteConfig() wrote invalid YAML: %v", err)
	}

	if parsed.Version != "1.0" {
		t.Errorf("WriteConfig() wrote version %q, want %q", parsed.Version, "1.0")
	}

	// Verify header comment
	if !strings.HasPrefix(string(content), "# Arx Architecture Configuration") {
		t.Errorf("WriteConfig() missing header comment")
	}
}

func TestWriteConfig_NilConfig(t *testing.T) {
	writer := newMockFileWriter()
	err := WriteConfig(nil, "arx.yaml", writer)
	if err == nil {
		t.Errorf("WriteConfig(nil) should return error")
	}
}

func TestWriteConfig_WriterError(t *testing.T) {
	writer := newMockFileWriter()
	writer.err = os.ErrPermission

	config := &domain.Config{
		Version: "1.0",
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	}

	err := WriteConfig(config, "arx.yaml", writer)
	if err == nil {
		t.Errorf("WriteConfig() with writer error should return error")
	}
}

func TestWriteConfig_UsesPortInterface(t *testing.T) {
	// Verify WriteConfig accepts the ports.FileWriter interface
	var _ ports.FileWriter = (*mockFileWriter)(nil)
}

// Helper function
func layerNames(layers []domain.Layer) []string {
	names := make([]string, len(layers))
	for i, l := range layers {
		names[i] = l.Name
	}
	return names
}
