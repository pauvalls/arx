package domain

import (
	"fmt"
	"strconv"
)

// TemplateFunc evaluates a rule template and returns violations.
// An empty slice means the rule passed.
//
//	params: validated YAML params map for this template
//	deps:  all detected dependencies
//	layers: all configured layers
type TemplateFunc func(params map[string]interface{}, deps []Dependency, layers []Layer) []Violation

// TemplateRegistry holds all registered rule templates.
var TemplateRegistry = map[string]TemplateFunc{
	"max-deps":      TemplateMaxDeps,
	"no-leak":       TemplateNoLeak,
	"layer-balance": TemplateLayerBalance,
}

// templateParamSchema defines the expected params for each template.
var templateParamSchema = map[string]map[string]string{
	"max-deps": {
		"from": "string",
		"to":   "[]string",
		"max":  "int",
	},
	"no-leak": {
		"layer":     "string",
		"forbidden": "[]string",
	},
	"layer-balance": {
		"min": "int",
		"max": "int",
	},
}

// ValidateTemplateParams checks that required params exist and have correct types
// for the given template name. Returns an error if validation fails.
func ValidateTemplateParams(templateName string, params map[string]interface{}) error {
	schema, ok := templateParamSchema[templateName]
	if !ok {
		return fmt.Errorf("unknown template %q", templateName)
	}

	for key, expectedType := range schema {
		val, exists := params[key]
		if !exists {
			return fmt.Errorf("missing required param %q for template %q", key, templateName)
		}
		if err := checkParamType(key, val, expectedType); err != nil {
			return fmt.Errorf("param %q for template %q: %w", key, templateName, err)
		}
	}

	return nil
}

// checkParamType validates that a param value matches the expected type string.
// Handles YAML unmarshaling quirks (ints come as float64).
func checkParamType(key string, val interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case "[]string":
		switch v := val.(type) {
		case []string:
			return nil
		case []interface{}:
			// YAML unmarshals sequences as []interface{} — validate each element is string
			for i, elem := range v {
				if _, ok := elem.(string); !ok {
					return fmt.Errorf("expected []string, element [%d] is %T", i, elem)
				}
			}
			return nil
		default:
			return fmt.Errorf("expected []string, got %T", val)
		}
	case "int":
		switch v := val.(type) {
		case int:
			return nil
		case float64:
			// YAML unmarshals numbers as float64 — accept if it's a whole number
			if v != float64(int(v)) {
				return fmt.Errorf("expected int, got float %v", v)
			}
			return nil
		case string:
			// Accept numeric strings
			if _, err := strconv.Atoi(v); err != nil {
				return fmt.Errorf("expected int, got string %q", v)
			}
			return nil
		default:
			return fmt.Errorf("expected int, got %T", val)
		}
	default:
		return fmt.Errorf("unknown expected type %q", expectedType)
	}
	return nil
}

// toInt converts a YAML-unmarshaled int param (possibly float64 or string) to int.
func toInt(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

// toStrSlice converts a YAML-unmarshaled []string param (possibly []interface{}) to []string.
func toStrSlice(val interface{}) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, elem := range v {
			if s, ok := elem.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// toStr safely extracts a string from a param value.
func toStr(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// resolveSourceLayer finds the layer that contains the given source file.
func resolveSourceLayer(filePath string, layers []Layer) string {
	for _, layer := range layers {
		if layer.MatchesPath(filePath) {
			return layer.Name
		}
	}
	return ""
}

// TemplateMaxDeps checks that a source layer does not exceed a maximum number
// of dependencies to a set of target layers.
//
//	Params: from (string), to ([]string), max (int)
//	When max=0, any dependency is a violation.
//	Multiple target layers → count all together.
func TemplateMaxDeps(params map[string]interface{}, deps []Dependency, layers []Layer) []Violation {
	from := toStr(params["from"])
	toTargets := toStrSlice(params["to"])
	maxDeps := toInt(params["max"])

	// Build target lookup set
	targetSet := make(map[string]bool, len(toTargets))
	for _, t := range toTargets {
		targetSet[t] = true
	}

	count := 0
	for _, dep := range deps {
		sourceLayer := resolveSourceLayer(dep.SourceFile, layers)
		if sourceLayer != from {
			continue
		}
		if targetSet[dep.ResolvedLayer] {
			count++
		}
	}

	if count > maxDeps {
		return []Violation{{
			Message: fmt.Sprintf("%s has %d dependencies to %v (max: %d)", from, count, toTargets, maxDeps),
		}}
	}
	return nil
}

// TemplateNoLeak checks that a layer does not leak its types into forbidden layers.
// Returns one violation per forbidden import.
//
//	Params: layer (string), forbidden ([]string)
func TemplateNoLeak(params map[string]interface{}, deps []Dependency, layers []Layer) []Violation {
	layerName := toStr(params["layer"])
	forbiddenList := toStrSlice(params["forbidden"])

	// Build forbidden lookup set
	forbiddenSet := make(map[string]bool, len(forbiddenList))
	for _, f := range forbiddenList {
		forbiddenSet[f] = true
	}

	var violations []Violation
	for _, dep := range deps {
		sourceLayer := resolveSourceLayer(dep.SourceFile, layers)
		if sourceLayer != layerName {
			continue
		}
		if forbiddenSet[dep.ResolvedLayer] {
			violations = append(violations, Violation{
				File:        dep.SourceFile,
				Line:        dep.SourceLine,
				SourceLayer: sourceLayer,
				TargetLayer: dep.ResolvedLayer,
				Import:      dep.ImportPath,
				Message:     fmt.Sprintf("%s imports %s from forbidden layer %s", dep.SourceFile, dep.ImportPath, dep.ResolvedLayer),
			})
		}
	}
	return violations
}

// TemplateLayerBalance checks that each layer's total outgoing dependency count
// falls within [min, max].
//
//	Params: min (int), max (int)
func TemplateLayerBalance(params map[string]interface{}, deps []Dependency, layers []Layer) []Violation {
	minDeps := toInt(params["min"])
	maxDeps := toInt(params["max"])

	// Count outgoing deps per source layer
	layerCounts := make(map[string]int)
	for _, dep := range deps {
		sourceLayer := resolveSourceLayer(dep.SourceFile, layers)
		if sourceLayer == "" {
			continue
		}
		layerCounts[sourceLayer]++
	}

	var violations []Violation
	for _, layer := range layers {
		count := layerCounts[layer.Name]
		if count < minDeps {
			violations = append(violations, Violation{
				SourceLayer: layer.Name,
				Message:     fmt.Sprintf("%s has %d dependencies (min: %d)", layer.Name, count, minDeps),
			})
		}
		if count > maxDeps {
			violations = append(violations, Violation{
				SourceLayer: layer.Name,
				Message:     fmt.Sprintf("%s has %d dependencies (max: %d)", layer.Name, count, maxDeps),
			})
		}
	}
	return violations
}
