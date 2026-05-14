package domain

import "fmt"

// Audit service evaluates architectural rules against detected dependencies

// EvaluateRules checks all dependencies against all rules and returns violations
func EvaluateRules(dependencies []Dependency, rules []Rule, layers []Layer) []Violation {
	var violations []Violation
	violationIndex := 0

	// Build a map of layer names to layer objects for quick lookup
	layerMap := make(map[string]*Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Evaluate each dependency against each rule
	for _, dep := range dependencies {
		// Resolve source and target layers
		sourceLayer := resolveLayer(dep.SourceFile, layerMap)
		targetLayer := dep.ResolvedLayer

		// Skip if we couldn't resolve layers
		if sourceLayer == "" || targetLayer == "" {
			continue
		}

		// Check against each rule
		for _, rule := range rules {
			if rule.Violates(dep.ImportPath, sourceLayer, targetLayer) {
				violationIndex++
				violations = append(violations, Violation{
					ID:          GenerateViolationID(rule, violationIndex),
					RuleID:      rule.ID,
					File:        dep.SourceFile,
					Line:        dep.SourceLine,
					SourceLayer: sourceLayer,
					TargetLayer: targetLayer,
					Import:      dep.ImportPath,
					Message:     buildViolationMessage(rule, sourceLayer, targetLayer, dep.ImportPath),
				})
			}
		}
	}

	return violations
}

// GenerateViolationID creates a sequential ID for a violation
func GenerateViolationID(rule Rule, index int) string {
	return fmt.Sprintf("D-%02d", index)
}

// resolveLayer finds the layer that matches a given file path
func resolveLayer(filePath string, layerMap map[string]*Layer) string {
	for name, layer := range layerMap {
		if layer.MatchesPath(filePath) {
			return name
		}
	}
	return ""
}

// buildViolationMessage creates a human-readable violation message
func buildViolationMessage(rule Rule, sourceLayer, targetLayer, importPath string) string {
	if rule.Explanation != "" {
		return fmt.Sprintf("%s: %s cannot import %s", rule.Explanation, sourceLayer, targetLayer)
	}
	return fmt.Sprintf("%s cannot depend on %s (import: %s)", sourceLayer, targetLayer, importPath)
}
