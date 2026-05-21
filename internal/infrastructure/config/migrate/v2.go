package migrate

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// V2Migration migrates a v1 config to v2.
// Currently a no-op that only updates the version field from "1.0" to "2.0".
func V2Migration(input []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(input, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("expected YAML document node")
	}

	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected YAML mapping node")
	}

	// Find and update the version field
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		key := mapping.Content[i]
		if key.Value == "version" {
			val := mapping.Content[i+1]
			if val.Value == "1.0" || val.Value == "1" {
				val.Value = "2.0"
			}
			break
		}
	}

	output, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshaling YAML: %w", err)
	}

	return output, nil
}
