package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/infrastructure/config"
)

func TestYAMLReader_Read(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	configContent := `
version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain
rules:
  - id: test-rule
    from: domain
    to: [infrastructure]
    type: cannot
    severity: error
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	// Read config
	reader := config.NewYAMLReader()
	config, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Validate
	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %q", config.Version)
	}

	if len(config.Layers) != 1 {
		t.Errorf("Expected 1 layer, got %d", len(config.Layers))
	}

	if config.Layers[0].Name != "domain" {
		t.Errorf("Expected layer name 'domain', got %q", config.Layers[0].Name)
	}

	if len(config.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(config.Rules))
	}
}

func TestYAMLReader_Read_FileNotFound(t *testing.T) {
	reader := config.NewYAMLReader()
	_, err := reader.Read("/nonexistent/arx.yaml")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestYAMLReader_Validate(t *testing.T) {
	reader := config.NewYAMLReader()

	// Valid config
	validConfig := &domain.Config{
		Version: "1.0",
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "test",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     "cannot",
				Severity: "error",
			},
		},
	}

	if err := reader.Validate(validConfig); err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}

	// Invalid config (missing version)
	invalidConfig := &domain.Config{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
	}

	if err := reader.Validate(invalidConfig); err == nil {
		t.Error("Invalid config should fail validation")
	}
}
