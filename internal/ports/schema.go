package ports

// SchemaGenerator defines the interface for generating JSON Schema from Go structs.
// This allows consumers to generate JSON Schema documents that describe the
// configuration structure for validation, IDE autocompletion, and documentation.
type SchemaGenerator interface {
	// Generate generates a JSON Schema document for the given value.
	// schemaName is used as the $id of the generated schema.
	// Returns pretty-printed JSON bytes.
	Generate(schemaName string, v interface{}) ([]byte, error)
}

// Compile-time check: ensure the interface is well-defined.
// Concrete implementations will be checked at import time.
var _ SchemaGenerator = (SchemaGenerator)(nil)
