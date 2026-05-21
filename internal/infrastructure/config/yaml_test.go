package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPipeline_WithIncludesAndEnvVars(t *testing.T) {
	dir := t.TempDir()

	// Create included layers file
	includeData := []byte("- name: ${LAYER_NAME}\n  paths: [\"./${LAYER_PATH}\"]\n")
	includePath := filepath.Join(dir, "layers.yaml")
	if err := os.WriteFile(includePath, includeData, 0644); err != nil {
		t.Fatalf("failed to write include: %v", err)
	}

	// Create main config with !include and env vars
	configData := []byte("version: \"${VERSION}\"\nlayers: !include layers.yaml\nrules: []\n")
	configPath := filepath.Join(dir, "arx.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Set env vars
	os.Setenv("VERSION", "1.0")
	os.Setenv("LAYER_NAME", "infra")
	os.Setenv("LAYER_PATH", "infrastructure")
	defer func() {
		os.Unsetenv("VERSION")
		os.Unsetenv("LAYER_NAME")
		os.Unsetenv("LAYER_PATH")
	}()

	reader := NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if cfg.Version.String() != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version.String(), "1.0")
	}
	if len(cfg.Layers) != 1 || cfg.Layers[0].Name != "infra" {
		t.Errorf("unexpected layers: %+v", cfg.Layers)
	}
}

func TestReadPipeline_NoIncludesNoEnvVars(t *testing.T) {
	dir := t.TempDir()

	configData := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\nrules: []\n")
	configPath := filepath.Join(dir, "arx.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reader := NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if cfg.Version.String() != "1.0" {
		t.Errorf("Version = %q, want %q", cfg.Version.String(), "1.0")
	}
	if len(cfg.Layers) != 1 || cfg.Layers[0].Name != "domain" {
		t.Errorf("unexpected layers: %+v", cfg.Layers)
	}
}

func TestReadPipeline_ErrorPropagation(t *testing.T) {
	dir := t.TempDir()

	// Config using an unset env var without default — should error
	configData := []byte("version: \"${MUST_EXIST}\"\nlayers: []\n")
	configPath := filepath.Join(dir, "arx.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	os.Unsetenv("MUST_EXIST")

	reader := NewYAMLReader()
	_, err := reader.Read(configPath)
	if err == nil {
		t.Fatal("Read() expected error for unset env var, got nil")
	}
	if !strings.Contains(err.Error(), "MUST_EXIST") {
		t.Errorf("error should mention the missing var, got: %v", err)
	}
}

func TestReadPipeline_SkipsIncludeResolutionWhenNoExclamation(t *testing.T) {
	dir := t.TempDir()

	configData := []byte("version: \"1.5\"\nlayers:\n  - name: infra\n    paths: [\"./infra\"]\n")
	configPath := filepath.Join(dir, "arx.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Set an env var that exists
	os.Setenv("EXISTING", "yes")
	defer os.Unsetenv("EXISTING")

	reader := NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if cfg.Version.String() != "1.5" {
		t.Errorf("Version = %q, want %q", cfg.Version.String(), "1.5")
	}
}
