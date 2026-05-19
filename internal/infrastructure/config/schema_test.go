package config

import (
	"encoding/json"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

type basicTypes struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Active  bool   `json:"active"`
	Score   float64 `json:"score,omitempty"`
}

type nestedStruct struct {
	Meta  basicTypes `json:"meta"`
	Label string     `json:"label"`
}

type withArray struct {
	Tags []string `json:"tags"`
}

type withMap struct {
	Meta map[string]string `json:"meta"`
}

type withPointer struct {
	Config *basicTypes `json:"config,omitempty"`
}

type withRequired struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

func TestSchemaGenerate_BasicTypes(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("test", basicTypes{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	if doc["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Errorf("missing $schema field")
	}

	props, ok := doc["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing properties object")
	}

	for _, name := range []string{"name", "age", "active"} {
		if _, exists := props[name]; !exists {
			t.Errorf("missing property: %s", name)
		}
	}
}

func TestSchemaGenerate_NestedStruct(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("nested", nestedStruct{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	props, ok := doc["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing properties")
	}

	meta, ok := props["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing meta property")
	}
	if meta["type"] != "object" {
		t.Errorf("meta type = %v, want object", meta["type"])
	}

	metaProps, ok := meta["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing meta.properties")
	}
	for _, name := range []string{"name", "age", "active"} {
		if _, exists := metaProps[name]; !exists {
			t.Errorf("missing nested property: %s", name)
		}
	}
}

func TestSchemaGenerate_Array(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("arr", withArray{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	props := doc["properties"].(map[string]interface{})
	tags := props["tags"].(map[string]interface{})

	if tags["type"] != "array" {
		t.Errorf("tags type = %v, want array", tags["type"])
	}

	items, ok := tags["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing tags.items")
	}
	if items["type"] != "string" {
		t.Errorf("tags.items type = %v, want string", items["type"])
	}
}

func TestSchemaGenerate_Map(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("map", withMap{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	props := doc["properties"].(map[string]interface{})
	meta := props["meta"].(map[string]interface{})

	if meta["type"] != "object" {
		t.Errorf("meta type = %v, want object", meta["type"])
	}
	if _, ok := meta["additionalProperties"]; !ok {
		t.Errorf("missing meta.additionalProperties")
	}
}

func TestSchemaGenerate_Pointer(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("ptr", withPointer{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	props := doc["properties"].(map[string]interface{})
	config, ok := props["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing config property")
	}
	if config["type"] != "object" {
		t.Errorf("config type = %v, want object", config["type"])
	}
}

func TestSchemaGenerate_RequiredFields(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("req", withRequired{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	required, ok := doc["required"].([]interface{})
	if !ok {
		t.Fatalf("missing required array")
	}

	requiredSet := make(map[string]bool)
	for _, r := range required {
		requiredSet[r.(string)] = true
	}

	if !requiredSet["id"] {
		t.Errorf("'id' should be required")
	}
	if !requiredSet["name"] {
		t.Errorf("'name' should be required")
	}
	if requiredSet["version"] {
		t.Errorf("'version' with omitempty should not be required")
	}
}

func TestSchemaGenerate_SeverityEnum(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("severity", struct {
		Level domain.Severity `json:"level"`
	}{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	props := doc["properties"].(map[string]interface{})
	level := props["level"].(map[string]interface{})

	if level["type"] != "string" {
		t.Errorf("level type = %v, want string", level["type"])
	}

	enum, ok := level["enum"].([]interface{})
	if !ok {
		t.Fatalf("missing level.enum")
	}

	enumVals := make(map[string]bool)
	for _, e := range enum {
		enumVals[e.(string)] = true
	}

	for _, expected := range []string{"error", "warning", "info", ""} {
		if !enumVals[expected] {
			t.Errorf("enum missing %q", expected)
		}
	}
}

func TestSchemaGenerate_DomainConfig(t *testing.T) {
	gen := &SchemaGeneratorImpl{}
	schema, err := gen.Generate("arx-config", domain.Config{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("Generate() produced invalid JSON: %v\n%s", err, string(schema))
	}

	// Verify $schema and $id
	if doc["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Errorf("missing $schema, got %v", doc["$schema"])
	}
	if doc["$id"] != "arx-config" {
		t.Errorf("$id = %v, want arx-config", doc["$id"])
	}

	props, ok := doc["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing properties")
	}

	// Should have key config fields
	for _, key := range []string{"version", "layers", "rules", "exclude"} {
		if _, exists := props[key]; !exists {
			t.Errorf("missing property: %s", key)
		}
	}

	// version should be required
	required, ok := doc["required"].([]interface{})
	if ok {
		hasVersion := false
		for _, r := range required {
			if r.(string) == "version" {
				hasVersion = true
				break
			}
		}
		if !hasVersion {
			t.Errorf("'version' should be required")
		}
	}
}

func TestSchemaGenerate_MapsGoTypes(t *testing.T) {
	gen := &SchemaGeneratorImpl{}

	tests := []struct {
		name     string
		val      interface{}
		wantType string
	}{
		{"string", struct{ F string `json:"f"` }{}, "string"},
		{"int", struct{ F int `json:"f"` }{}, "integer"},
		{"bool", struct{ F bool `json:"f"` }{}, "boolean"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := gen.Generate("test", tt.val)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			var doc map[string]interface{}
			if err := json.Unmarshal(schema, &doc); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			props := doc["properties"].(map[string]interface{})
			prop := props["f"].(map[string]interface{})

			if prop["type"] != tt.wantType {
				t.Errorf("type mapping: got %v, want %s", prop["type"], tt.wantType)
			}
		})
	}
}
