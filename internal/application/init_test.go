package application

import (
	"fmt"
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
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
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

	if parsed.Version.String() != "1.0" {
		t.Errorf("WriteConfig() wrote version %q, want %q", parsed.Version.String(), "1.0")
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
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
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

func TestGenerateConfigWithPreset_NoPreset(t *testing.T) {
	// When no preset is specified, should fall back to detection-based logic
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

	config, err := GenerateConfigWithPreset(info, "")
	if err != nil {
		t.Fatalf("GenerateConfigWithPreset() error = %v", err)
	}

	if config == nil {
		t.Fatal("GenerateConfigWithPreset() returned nil config")
	}

	// Should have generated rules based on detected layers
	if len(config.Rules) < 5 {
		t.Errorf("GenerateConfigWithPreset() created %d rules, want at least 5", len(config.Rules))
	}
}

func TestGenerateConfigWithPreset_CleanPreset(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{},
	}

	ps := newMockPresetService()
	ps.addPreset("clean", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
			{Name: "presentation", Paths: []string{"cmd/**"}},
		},
		Rules: []domain.Rule{
			{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: domain.RuleTypeCannot},
		},
		LanguageOverrides: map[string]domain.LanguageOverride{
			"go": {Extensions: []string{".go"}, Comment: "//", Import: "import"},
		},
	})

	config, err := GenerateConfigWithPreset(info, "clean", ps)
	if err != nil {
		t.Fatalf("GenerateConfigWithPreset() error = %v", err)
	}

	if config == nil {
		t.Fatal("GenerateConfigWithPreset() returned nil config")
	}

	// Verify clean architecture layers
	layerNames := make(map[string]bool)
	for _, layer := range config.Layers {
		layerNames[layer.Name] = true
	}

	expectedLayers := []string{"domain", "application", "infrastructure", "presentation"}
	for _, expected := range expectedLayers {
		if !layerNames[expected] {
			t.Errorf("GenerateConfigWithPreset(clean) missing layer: %s", expected)
		}
	}

	// Verify config is valid
	if err := config.Validate(); err != nil {
		t.Errorf("Generated config is invalid: %v", err)
	}
}

func TestGenerateConfigWithPreset_HexagonalPreset(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{},
	}

	ps := newMockPresetService()
	ps.addPreset("hexagonal", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "ports", Paths: []string{"internal/ports/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
		},
	})

	config, err := GenerateConfigWithPreset(info, "hexagonal", ps)
	if err != nil {
		t.Fatalf("GenerateConfigWithPreset() error = %v", err)
	}

	// Verify hexagonal architecture layers (should have ports layer)
	hasPorts := false
	for _, layer := range config.Layers {
		if layer.Name == "ports" {
			hasPorts = true
			break
		}
	}
	if !hasPorts {
		t.Errorf("GenerateConfigWithPreset(hexagonal) missing 'ports' layer")
	}
}

func TestGenerateConfigWithPreset_DddPreset(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{},
	}

	ps := newMockPresetService()
	ps.addPreset("ddd", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
			{Name: "interfaces", Paths: []string{"cmd/**"}},
		},
	})

	config, err := GenerateConfigWithPreset(info, "ddd", ps)
	if err != nil {
		t.Fatalf("GenerateConfigWithPreset() error = %v", err)
	}

	// Verify DDD layers (should have interfaces layer)
	hasInterfaces := false
	for _, layer := range config.Layers {
		if layer.Name == "interfaces" {
			hasInterfaces = true
			break
		}
	}
	if !hasInterfaces {
		t.Errorf("GenerateConfigWithPreset(ddd) missing 'interfaces' layer")
	}
}

func TestGenerateConfigWithPreset_InvalidPreset(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
	}

	ps := newMockPresetService()
	// No presets added, so any lookup fails

	_, err := GenerateConfigWithPreset(info, "invalid", ps)
	if err == nil {
		t.Fatal("GenerateConfigWithPreset(invalid) should return error")
	}

	if !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("GenerateConfigWithPreset(invalid) error = %v, want 'unknown preset'", err)
	}
}

func TestGenerateConfigWithPreset_AddsLanguageOverrides(t *testing.T) {
	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go", "typescript"},
	}

	ps := newMockPresetService()
	ps.addPreset("clean", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
			{Name: "presentation", Paths: []string{"cmd/**"}},
		},
		Rules: []domain.Rule{
			{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: domain.RuleTypeCannot},
		},
		// No language overrides in preset — they should be added
	})

	config, err := GenerateConfigWithPreset(info, "clean", ps)
	if err != nil {
		t.Fatalf("GenerateConfigWithPreset() error = %v", err)
	}

	// Verify config is valid
	if err := config.Validate(); err != nil {
		t.Fatalf("Generated config is invalid: %v", err)
	}

	// Language overrides should have been added since preset had none
	if len(config.LanguageOverrides) == 0 {
		t.Error("GenerateConfigWithPreset() did not include language overrides")
	}

	if _, ok := config.LanguageOverrides["go"]; !ok {
		t.Error("GenerateConfigWithPreset() missing Go language override")
	}
	if _, ok := config.LanguageOverrides["typescript"]; !ok {
		t.Error("GenerateConfigWithPreset() missing TypeScript language override")
	}
}

// mockPresetService implements ports.PresetService for testing
type mockPresetService struct {
	presets map[string]*domain.Config
}

func newMockPresetService() *mockPresetService {
	return &mockPresetService{
		presets: make(map[string]*domain.Config),
	}
}

func (m *mockPresetService) LoadPreset(name string) (*domain.Config, error) {
	cfg, ok := m.presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q", name)
	}
	return cfg, nil
}

func (m *mockPresetService) ListPresets() []string {
	names := make([]string, 0, len(m.presets))
	for name := range m.presets {
		names = append(names, name)
	}
	return names
}

func (m *mockPresetService) addPreset(name string, cfg *domain.Config) {
	m.presets[name] = cfg
}

func TestInitService_InitWithPreset_Valid(t *testing.T) {
	writer := newMockFileWriter()
	presetService := newMockPresetService()
	presetService.addPreset("clean", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
			{Name: "presentation", Paths: []string{"cmd/**"}},
		},
		Rules: []domain.Rule{
			{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: domain.RuleTypeCannot},
		},
	})

	service := NewInitServiceWithPreset(writer, presetService)

	config, err := service.InitWithPreset("clean", "arx.yaml", false)
	if err != nil {
		t.Fatalf("InitWithPreset() error = %v", err)
	}

	if config == nil {
		t.Fatal("InitWithPreset() returned nil config")
	}

	content, ok := writer.files["arx.yaml"]
	if !ok {
		t.Fatalf("InitWithPreset() did not write to arx.yaml")
	}

	// Verify header comment
	contentStr := string(content)
	if !strings.Contains(contentStr, "# Arx Architecture Configuration") {
		t.Errorf("InitWithPreset() missing header comment")
	}
	if !strings.Contains(contentStr, "# Preset: clean") {
		t.Errorf("InitWithPreset() missing preset name in header")
	}
	if !strings.Contains(contentStr, "# Generated:") {
		t.Errorf("InitWithPreset() missing timestamp in header")
	}
	if !strings.Contains(contentStr, "⚠️  This is a starting point") {
		t.Errorf("InitWithPreset() missing warning comment")
	}

	// Verify YAML content is valid
	var parsed domain.Config
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Errorf("InitWithPreset() wrote invalid YAML: %v", err)
	}

	if parsed.Version.String() != "1.0" {
		t.Errorf("InitWithPreset() wrote version %q, want %q", parsed.Version.String(), "1.0")
	}
}

func TestInitService_InitWithPreset_FileExists_NoForce(t *testing.T) {
	writer := newMockFileWriter()
	writer.exists["arx.yaml"] = true

	presetService := newMockPresetService()
	presetService.addPreset("clean", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	})

	service := NewInitServiceWithPreset(writer, presetService)

	config, err := service.InitWithPreset("clean", "arx.yaml", false)
	if err == nil {
		t.Errorf("InitWithPreset() with existing file and force=false should return error")
	}

	if config != nil {
		t.Errorf("InitWithPreset() with existing file should return nil config")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("InitWithPreset() error = %v, want 'already exists'", err)
	}
}

func TestInitService_InitWithPreset_FileExists_WithForce(t *testing.T) {
	writer := newMockFileWriter()
	writer.exists["arx.yaml"] = true

	presetService := newMockPresetService()
	presetService.addPreset("clean", &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	})

	service := NewInitServiceWithPreset(writer, presetService)

	config, err := service.InitWithPreset("clean", "arx.yaml", true)
	if err != nil {
		t.Fatalf("InitWithPreset() with force=true error = %v", err)
	}

	if config == nil {
		t.Fatal("InitWithPreset() with force=true returned nil config")
	}

	// Verify file was written despite existing
	content, ok := writer.files["arx.yaml"]
	if !ok {
		t.Fatalf("InitWithPreset() with force=true did not write to arx.yaml")
	}

	// Verify content is valid
	var parsed domain.Config
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Errorf("InitWithPreset() wrote invalid YAML: %v", err)
	}
}

func TestInitService_InitWithPreset_InvalidPreset(t *testing.T) {
	writer := newMockFileWriter()
	presetService := newMockPresetService()
	// No presets added, so any preset name is invalid

	service := NewInitServiceWithPreset(writer, presetService)

	config, err := service.InitWithPreset("invalid", "arx.yaml", false)
	if err == nil {
		t.Errorf("InitWithPreset() with invalid preset should return error")
	}

	if config != nil {
		t.Errorf("InitWithPreset() with invalid preset should return nil config")
	}

	if !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("InitWithPreset() error = %v, want 'unknown preset'", err)
	}
}

func TestInitService_InitWithPreset_NoPresetService(t *testing.T) {
	writer := newMockFileWriter()
	// Create service without preset service
	service := NewInitService(writer)

	config, err := service.InitWithPreset("clean", "arx.yaml", false)
	if err == nil {
		t.Errorf("InitWithPreset() without preset service should return error")
	}

	if config != nil {
		t.Errorf("InitWithPreset() without preset service should return nil config")
	}

	if !strings.Contains(err.Error(), "preset service not configured") {
		t.Errorf("InitWithPreset() error = %v, want 'preset service not configured'", err)
	}
}
