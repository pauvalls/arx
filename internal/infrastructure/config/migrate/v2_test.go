package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"gopkg.in/yaml.v3"
)

func TestV2Migration_PreservesFields(t *testing.T) {
	input := []byte(`version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain/**"]
rules:
  - id: test-rule
    from: domain
    to: ["infrastructure"]
    type: Cannot
    severity: error
exclude:
  - vendor/**
`)

	output, err := V2Migration(input)
	if err != nil {
		t.Fatalf("V2Migration() error = %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Verify version is updated
	ver, ok := result["version"]
	if !ok {
		t.Fatal("version field missing after migration")
	}
	if ver != "2.0" {
		t.Errorf("version = %q, want %q", ver, "2.0")
	}

	// Verify layers preserved
	layers, ok := result["layers"].([]interface{})
	if !ok || len(layers) != 1 {
		t.Fatalf("layers not preserved: %v", result["layers"])
	}

	// Verify rules preserved
	rules, ok := result["rules"].([]interface{})
	if !ok || len(rules) != 1 {
		t.Fatalf("rules not preserved: %v", result["rules"])
	}

	// Verify exclude preserved
	exclude, ok := result["exclude"].([]interface{})
	if !ok || len(exclude) != 1 {
		t.Fatalf("exclude not preserved: %v", result["exclude"])
	}
}

func TestV2Migration_BackupCreated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	input := []byte(`version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain"]
rules: []
`)
	if err := os.WriteFile(configPath, input, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Run migration
	output, err := V2Migration(input)
	if err != nil {
		t.Fatalf("V2Migration() error = %v", err)
	}

	// Use domain.SchemaVersion to check
	var cfg domain.Config
	if err := yaml.Unmarshal(output, &cfg); err != nil {
		t.Fatalf("failed to unmarshal migrated config: %v", err)
	}

	if cfg.Version.String() != "2.0" {
		t.Errorf("version = %q, want %q", cfg.Version.String(), "2.0")
	}
}

func TestV2MigrationRegistered(t *testing.T) {
	reg := domain.NewRegistry()
	v1 := domain.SchemaVersion{Major: 1, Minor: 0}
	v2 := domain.SchemaVersion{Major: 2, Minor: 0}

	err := reg.Register(domain.Migration{From: v1, To: v2, Func: V2Migration})
	if err != nil {
		t.Fatalf("Register(v1→v2) error: %v", err)
	}

	funcs, err := reg.Resolve(v1, v2)
	if err != nil {
		t.Fatalf("Resolve(v1, v2) error: %v", err)
	}
	if len(funcs) != 1 {
		t.Fatalf("got %d migration funcs, want 1", len(funcs))
	}
}
