package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
)

// TestConfigSchema_Acceptance verifies that the generated JSON Schema for domain.Config
// contains all expected structural properties. This test breaks if Config struct changes
// without a corresponding schema update, acting as an acceptance guard against schema drift.
func TestConfigSchema_Acceptance(t *testing.T) {
	gen := &config.SchemaGeneratorImpl{}
	schema, err := gen.Generate("arx-config", domain.Config{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	// Verify $schema field
	if doc["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Errorf("$schema = %v, want https://json-schema.org/draft-07/schema#", doc["$schema"])
	}

	// Verify properties exist
	props, ok := doc["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("missing properties in schema")
	}

	expectedProps := []string{"version", "layers", "rules"}
	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("missing property: %s", prop)
		}
	}

	// Verify layers items have 'name' and 'paths' properties
	layers, ok := props["layers"].(map[string]interface{})
	if !ok {
		t.Fatal("layers property is not an object")
	}
	if layers["type"] != "array" {
		t.Errorf("layers type = %v, want array", layers["type"])
	}

	layersItems, ok := layers["items"].(map[string]interface{})
	if !ok {
		t.Fatal("layers missing items")
	}
	layerProps, ok := layersItems["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("layers items missing properties")
	}
	for _, field := range []string{"name", "paths"} {
		if _, exists := layerProps[field]; !exists {
			t.Errorf("layer item missing property: %s", field)
		}
	}

	// Verify version is required
	required, ok := doc["required"].([]interface{})
	if !ok {
		t.Fatal("missing required array")
	}

	hasVersion := false
	for _, r := range required {
		if r.(string) == "version" {
			hasVersion = true
			break
		}
	}
	if !hasVersion {
		t.Error("version is not in required list")
	}
}
