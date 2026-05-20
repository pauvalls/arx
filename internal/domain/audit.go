package domain

import (
	"context"
	"fmt"
)

// Audit service evaluates architectural rules against detected dependencies

// EvaluateRules checks all dependencies against all rules and returns violations.
// userFuncs is an optional map of compiled user-defined function expressions.
func EvaluateRules(dependencies []Dependency, rules []Rule, layers []Layer, userFuncs ...map[string]Expr) []Violation {
	var violations []Violation
	violationIndex := 0

	// Build a map of layer names to layer objects for quick lookup
	layerMap := make(map[string]*Layer)
	layerFiles := make(map[string][]string)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Build unique file list per layer from dependencies
	fileSeen := make(map[string]bool)
	for _, dep := range dependencies {
		srcLayer := resolveLayer(dep.SourceFile, layerMap)
		if srcLayer != "" && !fileSeen[dep.SourceFile] {
			fileSeen[dep.SourceFile] = true
			layerFiles[srcLayer] = append(layerFiles[srcLayer], dep.SourceFile)
		}
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
		for i := range rules {
			rule := &rules[i]

			// Expression-based rules are evaluated separately (not per-dependency)
			if rule.CheckExpressionIsStandalone() {
				continue
			}

			if rule.Violates(dep.ImportPath, sourceLayer, targetLayer) {
				// Check if rule is disabled for this file path
				if !rule.IsEnabledFor(dep.SourceFile) {
					continue
				}

				// Check if file is excluded from this rule
				// Defensive: compile exclude patterns if not already compiled
				if rule.compiledExclude == nil && len(rule.Exclude) > 0 {
					_ = rule.CompileExcludePatterns()
				}
				if rule.IsExcludedFor(dep.SourceFile) {
					continue
				}

				violationIndex++
				v := Violation{
					ID:          GenerateViolationID(*rule, violationIndex),
					RuleID:      rule.ID,
					File:        dep.SourceFile,
					Line:        dep.SourceLine,
					SourceLayer: sourceLayer,
					TargetLayer: targetLayer,
					Import:      dep.ImportPath,
					Message:     buildViolationMessage(*rule, sourceLayer, targetLayer, dep.ImportPath),
					Severity:    rule.Severity,
				}

				// Apply severity override if matching
				if overrideSev, ok := rule.GetEffectiveSeverity(dep.SourceFile); ok {
					v.OriginalSeverity = v.Severity // save original before override
					v.Severity = overrideSev
					v.Overridden = true
				}

				violations = append(violations, v)
			}
		}
	}

	// Inject user-defined functions into EvalContext (if provided via variadic)
	var userFnMap map[string]Expr
	if len(userFuncs) > 0 {
		userFnMap = userFuncs[0]
	}

	// Evaluate expression-based rules
	exprCtx := EvalContext{
		Deps:          dependencies,
		Layers:        layers,
		Violations:    violations,
		LayerFiles:    layerFiles,
		UserFunctions: userFnMap,
	}
	for i := range rules {
		rule := &rules[i]
		if !rule.CheckExpressionIsStandalone() {
			continue
		}
		matched, err := ruleCheckMatches(rule, exprCtx)
		if err != nil {
			// Log error but continue; validation should have caught this
			continue
		}
		if matched {
			violationIndex++
			v := Violation{
				ID:          GenerateViolationID(*rule, violationIndex),
				RuleID:      rule.ID,
				File:        "",
				Line:        0,
				SourceLayer: "",
				TargetLayer: "",
				Import:      "",
				Message:     buildExprViolationMessage(*rule),
				Severity:    rule.Severity,
			}
			violations = append(violations, v)
		}
	}

	// Evaluate template-based rules
	for i := range rules {
		rule := &rules[i]
		if rule.Template == "" {
			continue
		}
		fn, ok := TemplateRegistry[rule.Template]
		if !ok {
			continue // should have been caught by validation
		}
		templateViolations := fn(rule.Params, dependencies, layers)
		for _, tv := range templateViolations {
			violationIndex++
			tv.ID = GenerateTemplateViolationID(violationIndex)
			tv.RuleID = rule.ID
			if tv.Severity == "" {
				tv.Severity = rule.Severity
			}
			violations = append(violations, tv)
		}
	}

	// Check for circular dependencies
	circularViolations := EvaluateCircularDependencies(dependencies, rules, layers)
	violations = append(violations, circularViolations...)

	return violations
}

// GenerateViolationID creates a sequential ID for a violation
func GenerateViolationID(rule Rule, index int) string {
	return fmt.Sprintf("D-%02d", index)
}

// GenerateTemplateViolationID creates a sequential ID for a template-based violation
func GenerateTemplateViolationID(index int) string {
	return fmt.Sprintf("T-%02d", index)
}

// GenerateWasmViolationID creates a sequential ID for a WASM-based violation
func GenerateWasmViolationID(index int) string {
	return fmt.Sprintf("W-%02d", index)
}

// EvaluateWasmRules evaluates all rules with Wasm config and appends violations.
// It requires an external WasmEvaluator to be provided per rule's WASM module path.
// If evaluatorFn returns nil for a rule's path, the rule's policies are skipped.
// Errors are logged by incrementing the returned error count.
func EvaluateWasmRules(ctx context.Context, rules []Rule, deps []Dependency, layers []Layer, existingViolations []Violation, evaluatorFn func(wasmPath string) WasmEvaluator) ([]Violation, int) {
	var violations []Violation
	errorCount := 0

	for i := range rules {
		rule := &rules[i]
		if rule.Wasm == nil {
			continue
		}
		if !rule.IsEnabledFor("") {
			continue
		}

		eval := evaluatorFn(rule.Wasm.Path)
		if eval == nil {
			errorCount++
			continue
		}

		wasmViolations, err := eval.Evaluate(ctx, deps, layers, existingViolations, rule.Wasm.Params)
		if err != nil {
			errorCount++
			continue // graceful degradation: skip this policy on error
		}

		// Map returned violations to the rule
		for _, wv := range wasmViolations {
			if rule.IsExcludedFor(wv.File) {
				continue
			}
			if !rule.IsEnabledFor(wv.File) {
				continue
			}
			v := wv
			if v.RuleID == "" {
				v.RuleID = rule.ID
			}
			if v.Severity == "" {
				v.Severity = rule.Severity
			}
			violations = append(violations, v)
		}
	}

	return violations, errorCount
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

// EvaluateCircularDependencies detects circular dependencies and creates violations
func EvaluateCircularDependencies(dependencies []Dependency, rules []Rule, layers []Layer) []Violation {
	var violations []Violation

	// Find circular dependency rules
	var circularRules []Rule
	for _, rule := range rules {
		if rule.Type == RuleTypeMustNotCircular {
			circularRules = append(circularRules, rule)
		}
	}

	// If no circular rules defined, use default circular detection
	if len(circularRules) == 0 {
		circularRules = []Rule{
			{
				ID:       "no-circular-dependencies",
				From:     "*",
				To:       []string{"*"},
				Type:     RuleTypeMustNotCircular,
				Severity: SeverityError,
				Explanation: "Circular dependencies create tightly coupled code that is hard to test, modify, and deploy.",
			},
		}
	}

	// Detect cycles
	cycles := DetectCircularDependencies(dependencies, layers)

	// Create violations for each cycle
	for _, cycle := range cycles {
		for _, rule := range circularRules {
			violations = append(violations, CreateCircularViolations([]CircularDependency{cycle}, rule)...)
		}
	}

	return violations
}
