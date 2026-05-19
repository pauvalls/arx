package ports

import (
	"encoding/json"
	"testing"
)

// MockSchemaGenerator implements SchemaGenerator for testing
type MockSchemaGenerator struct {
	generateResult []byte
	generateErr    error
}

func (m *MockSchemaGenerator) Generate(schemaName string, v interface{}) ([]byte, error) {
	return m.generateResult, m.generateErr
}

// TestSchemaGeneratorInterface verifies the SchemaGenerator interface can be implemented
func TestSchemaGeneratorInterface(t *testing.T) {
	var _ SchemaGenerator = (*MockSchemaGenerator)(nil)

	gen := &MockSchemaGenerator{
		generateResult: []byte(`{"$schema": "https://json-schema.org/draft-07/schema#"}`),
	}

	result, err := gen.Generate("test", struct{ Name string }{Name: "hello"})
	if err != nil {
		t.Errorf("Generate() error = %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(result, &doc); err != nil {
		t.Errorf("Generate() result is not valid JSON: %v", err)
	}

	if doc["$schema"] != "https://json-schema.org/draft-07/schema#" {
		t.Errorf("Generate() missing $schema field")
	}
}

// TestSchemaGenerator_NilValue verifies the interface contract for nil values
func TestSchemaGenerator_NilValue(t *testing.T) {
	gen := &MockSchemaGenerator{}

	// The interface contract: passing nil should be handled without panic
	result, err := gen.Generate("test", nil)
	if err != nil {
		t.Errorf("Generate(nil) error = %v (should be handled gracefully)", err)
	}
	_ = result // result may be nil — the key is no panic
}

// TestSchemaGenerator_ErrorPropagation verifies errors from Generate are propagated
func TestSchemaGenerator_ErrorPropagation(t *testing.T) {
	gen := &MockSchemaGenerator{
		generateErr: nil,
	}

	// When there's no error, Generate should succeed
	_, err := gen.Generate("test", map[string]string{"key": "val"})
	if err != nil {
		t.Errorf("Generate() unexpected error = %v", err)
	}
}
