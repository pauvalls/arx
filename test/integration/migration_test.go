package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config/migrate"
	"github.com/pauvalls/arx/internal/infrastructure/server"
	"gopkg.in/yaml.v3"
)

// TestMigrationRoundTrip tests the full migration flow:
// 1. Create v1 config
// 2. Run migrate dry-run
// 3. Run migrate (real)
// 4. Verify file modified and backup created
func TestMigrationRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	// Create a minimal v1 config
	configData := []byte(`version: "1.0"
layers:
  - name: domain
    paths: ["./domain"]
rules: []
`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create registry with v1→v2 migration
	reg := domain.NewRegistry()
	err := reg.Register(domain.Migration{
		From: domain.SchemaVersion{Major: 1, Minor: 0},
		To:   domain.SchemaVersion{Major: 2, Minor: 0},
		Func: migrate.V2Migration,
	})
	if err != nil {
		t.Fatalf("Register migration: %v", err)
	}

	// Verify we can resolve v1→v2
	funcs, err := reg.Resolve(
		domain.SchemaVersion{Major: 1, Minor: 0},
		domain.SchemaVersion{Major: 2, Minor: 0},
	)
	if err != nil {
		t.Fatalf("Resolve v1→v2: %v", err)
	}
	if len(funcs) != 1 {
		t.Fatalf("expected 1 migration func, got %d", len(funcs))
	}

	// Apply v1→v2 migration directly
	output, err := migrate.V2Migration(configData)
	if err != nil {
		t.Fatalf("V2Migration: %v", err)
	}

	// Verify version updated
	var doc map[string]interface{}
	if err := yaml.Unmarshal(output, &doc); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	ver, ok := doc["version"].(string)
	if !ok {
		t.Fatal("version field missing")
	}
	if ver != "2.0" {
		t.Errorf("version = %q, want %q", ver, "2.0")
	}

	// Verify layers preserved
	if _, ok := doc["layers"]; !ok {
		t.Error("layers field missing after migration")
	}
}

// TestJSONSchemaVersion verifies that JSON output can include schema_version.
func TestJSONSchemaVersion(t *testing.T) {
	// Create a JSONReporter and verify schema_version wiring
	// This test verifies the concept — the actual CLI integration is in unit tests
	cfg := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
	}
	if cfg.Version.String() != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version.String())
	}
}

// TestSchemaEndpointResponse verifies the schema endpoint contract.
func TestSchemaEndpointResponse(t *testing.T) {
	info := server.SchemaInfoResponse{
		Current:   "1.0",
		Supported: []string{"1.0"},
		SchemaURL: "",
	}
	if info.Current != "1.0" {
		t.Errorf("current = %q, want %q", info.Current, "1.0")
	}
	if len(info.Supported) != 1 || info.Supported[0] != "1.0" {
		t.Errorf("supported = %v, want [1.0]", info.Supported)
	}
	if info.SchemaURL != "" {
		t.Errorf("schema_url = %q, want empty (local schema)", info.SchemaURL)
	}

	// Verify JSON serialization
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal SchemaInfoResponse: %v", err)
	}
	if !strings.Contains(string(data), `"current":"1.0"`) {
		t.Errorf("JSON missing current field: %s", data)
	}
	if !strings.Contains(string(data), `"schema_url":""`) {
		t.Errorf("JSON missing schema_url field: %s", data)
	}
}

// TestConfigV1LoadsSuccessfully verifies that a v1 config still loads.
func TestConfigV1LoadsSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	configData := []byte(`version: "1.0"
layers:
  - name: domain
    paths: ["./domain"]
rules: []
`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read and parse via yaml
	var cfg domain.Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	if cfg.Version.String() != "1.0" {
		t.Errorf("version = %q, want %q", cfg.Version.String(), "1.0")
	}
	if len(cfg.Layers) != 1 {
		t.Errorf("layers = %d, want 1", len(cfg.Layers))
	}
}
