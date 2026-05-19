package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// Compile-time check: SchemaGeneratorImpl implements ports.SchemaGenerator
var _ ports.SchemaGenerator = (*SchemaGeneratorImpl)(nil)

// SchemaGeneratorImpl generates JSON Schema from Go structs using reflection.
type SchemaGeneratorImpl struct{}

// Generate generates a JSON Schema document from the provided struct value.
// schemaName is used as the $id of the schema.
func (g *SchemaGeneratorImpl) Generate(schemaName string, v interface{}) ([]byte, error) {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil, fmt.Errorf("cannot generate schema from nil")
	}

	// Dereference pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft-07/schema#",
		"$id":     schemaName,
		"title":   t.Name(),
		"type":    "object",
	}

	properties := make(map[string]interface{})
	var required []string

	buildProperties(t, properties, &required)
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	result, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}

	return result, nil
}

// buildProperties populates the properties map with JSON Schema representations
// of all fields in the struct type t.
func buildProperties(t reflect.Type, props map[string]interface{}, required *[]string) {
	for i := range t.NumField() {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() && !field.Anonymous {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			// No json tag — use field name
			jsonTag = field.Name
		}

		// Parse json tag
		name, opts := parseJSONTag(jsonTag)
		if name == "-" {
			continue
		}

		// Build the schema for this field's type
		fieldSchema := typeToSchema(field.Type)

		// Check required — fields without omitempty are required
		if !opts["omitempty"] {
			*required = append(*required, name)
		}

		props[name] = fieldSchema
	}
}

// typeToSchema converts a Go reflect.Type to a JSON Schema representation.
func typeToSchema(t reflect.Type) map[string]interface{} {
	// Dereference pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		schema := map[string]interface{}{"type": "string"}

		// Check if this is a Severity type (enum)
		if t == reflect.TypeOf(domain.Severity("")) {
			schema["enum"] = []interface{}{"error", "warning", "info", ""}
		}

		return schema

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}

	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}

	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}

	case reflect.Slice, reflect.Array:
		items := typeToSchema(t.Elem())
		return map[string]interface{}{
			"type":  "array",
			"items": items,
		}

	case reflect.Map:
		additionalProps := typeToSchema(t.Elem())
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": additionalProps,
		}

	case reflect.Struct:
		props := make(map[string]interface{})
		var required []string
		buildProperties(t, props, &required)

		schema := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}

		if len(required) > 0 {
			schema["required"] = required
		}

		// Add title for named structs
		if t.Name() != "" {
			schema["title"] = t.Name()
		}

		return schema

	default:
		// For interface{}, maps with interface{} values, etc.
		return map[string]interface{}{}
	}
}

// parseJSONTag splits a json field tag into the field name and options map.
func parseJSONTag(tag string) (string, map[string]bool) {
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = parts[0]
	}

	opts := make(map[string]bool)
	for _, opt := range parts[1:] {
		opts[strings.TrimSpace(opt)] = true
	}

	return name, opts
}
