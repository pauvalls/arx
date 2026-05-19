package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DeepMerge merges two YAML configurations. Override values replace base values
// at the top level. For nested maps, merge recursively. For arrays, override
// replaces entirely. Base-only keys survive untouched.
// Returns merged YAML bytes.
func DeepMerge(base, override []byte) ([]byte, error) {
	// If override is nil or empty, return base as-is
	if len(override) == 0 {
		return base, nil
	}

	// Parse both documents into generic maps
	var baseMap map[string]interface{}
	if err := yaml.Unmarshal(base, &baseMap); err != nil {
		return nil, fmt.Errorf("parsing base config: %w", err)
	}

	var overrideMap map[string]interface{}
	if err := yaml.Unmarshal(override, &overrideMap); err != nil {
		return nil, fmt.Errorf("parsing override config: %w", err)
	}

	// Merge override into base
	deepMergeMaps(baseMap, overrideMap)

	// Serialize back to YAML
	result, err := yaml.Marshal(baseMap)
	if err != nil {
		return nil, fmt.Errorf("serializing merged config: %w", err)
	}

	return result, nil
}

// deepMergeMaps recursively merges override into base.
func deepMergeMaps(base, override map[string]interface{}) {
	for key, overrideVal := range override {
		baseVal, exists := base[key]
		if !exists {
			// New key from override — add it
			base[key] = overrideVal
			continue
		}

		// Both are maps → merge recursively
		baseMap, baseIsMap := baseVal.(map[string]interface{})
		overrideMap, overrideIsMap := overrideVal.(map[string]interface{})
		if baseIsMap && overrideIsMap {
			deepMergeMaps(baseMap, overrideMap)
			continue
		}

		// Override replaces base (arrays, scalars, etc.)
		base[key] = overrideVal
	}
}
