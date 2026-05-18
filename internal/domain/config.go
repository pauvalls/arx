package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// LanguageOverride allows language-specific configuration overrides
type LanguageOverride struct {
	Extensions []string `json:"extensions" yaml:"extensions"`
	Comment    string   `json:"comment,omitempty" yaml:"comment,omitempty"`
	Import     string   `json:"import,omitempty" yaml:"import,omitempty"`
}

// SeverityConfig allows customization of severity behavior
type SeverityConfig struct {
	FailBuild bool `json:"fail_build" yaml:"fail_build"`
	ShowInUI  bool `json:"show_in_ui" yaml:"show_in_ui"`
}

// Config represents the complete Arx configuration
type Config struct {
	Schema         string                      `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Version        string                      `json:"version" yaml:"version"`
	Layers         []Layer                     `json:"layers" yaml:"layers"`
	Rules          []Rule                      `json:"rules" yaml:"rules"`
	LanguageOverrides map[string]LanguageOverride `json:"language_overrides,omitempty" yaml:"language_overrides,omitempty"`
	Exclude        []string                    `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	SeverityConfig map[Severity]SeverityConfig `json:"severity_config,omitempty" yaml:"severity_config,omitempty"`
	MaxViolations  int                         `json:"max_violations,omitempty" yaml:"max_violations,omitempty"`
	SeverityMapping map[string]string           `json:"severity_mapping,omitempty" yaml:"severity_mapping,omitempty"`
	Functions      map[string]string            `json:"functions,omitempty" yaml:"functions,omitempty"`

	userFunctions map[string]Expr `json:"-" yaml:"-"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("config version is required")
	}

	if len(c.Layers) == 0 {
		return fmt.Errorf("at least one layer must be defined")
	}

	// Validate max_violations threshold
	if c.MaxViolations < 0 {
		return fmt.Errorf("max_violations cannot be negative (got %d)", c.MaxViolations)
	}

	// Validate and apply severity mapping
	if err := c.applySeverityMapping(); err != nil {
		return err
	}

	// Compile and validate user-defined functions
	if err := c.compileFunctions(); err != nil {
		return err
	}

	// Validate all layers
	layerNames := make(map[string]bool)
	for i := range c.Layers {
		if err := c.Layers[i].Validate(); err != nil {
			return fmt.Errorf("layer[%d]: %w", i, err)
		}
		if layerNames[c.Layers[i].Name] {
			return fmt.Errorf("duplicate layer name: %q", c.Layers[i].Name)
		}
		layerNames[c.Layers[i].Name] = true
	}

	// Validate all rules
	for i := range c.Rules {
		if err := c.Rules[i].Validate(); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
		// Expression rules must be standalone (no from/to/template/pattern mixing)
		if c.Rules[i].Check.Raw != "" {
			if c.Rules[i].From != "" {
				return fmt.Errorf("rule[%d]: 'check' expression rules cannot have 'from' field", i)
			}
			if len(c.Rules[i].To) > 0 {
				return fmt.Errorf("rule[%d]: 'check' expression rules cannot have 'to' field", i)
			}
			if c.Rules[i].Template != "" {
				return fmt.Errorf("rule[%d]: 'check' expression rules cannot have 'template' field", i)
			}
			if c.Rules[i].Pattern != "" {
				return fmt.Errorf("rule[%d]: 'check' expression rules cannot have 'pattern' field", i)
			}
		}

		// Check that rule references valid layers (skip pattern-only and template-only rules)
		if c.Rules[i].From != "" {
			if !layerNames[c.Rules[i].From] {
				return fmt.Errorf("rule[%d]: 'from' references unknown layer %q", i, c.Rules[i].From)
			}
		}
		for _, to := range c.Rules[i].To {
			if !layerNames[to] {
				return fmt.Errorf("rule[%d]: 'to' references unknown layer %q", i, to)
			}
		}
		// Validate template params that reference layer names
		if c.Rules[i].Template != "" && c.Rules[i].Params != nil {
			if err := validateTemplateLayerRefs(c.Rules[i].Template, c.Rules[i].Params, layerNames, i); err != nil {
				return err
			}
		}
	}

	return nil
}

// Hash returns a SHA-256 hex digest of the marshaled JSON config.
// Used for cache invalidation and baseline staleness checks.
func (c *Config) Hash() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// ViolationThreshold returns the maximum number of violations allowed before failing.
// Returns 0 if no threshold is set (unlimited, backward-compatible).
func (c *Config) ViolationThreshold() int {
	return c.MaxViolations
}

// UserFunctions returns the compiled user-defined function expressions,
// or nil if no functions are defined or if compileFunctions has not been called.
func (c *Config) UserFunctions() map[string]Expr {
	return c.userFunctions
}

// compileFunctions parses, validates, and compiles all user-defined functions.
// It checks identifier validity, builtin shadowing, and detects cycles in the
// function call graph using Kahn's algorithm.
func (c *Config) compileFunctions() error {
	if len(c.Functions) == 0 {
		c.userFunctions = nil
		return nil
	}

	// Phase 1: parse and validate each function
	parsed := make(map[string]Expr, len(c.Functions))
	for name, body := range c.Functions {
		if !IsValidIdentifier(name) {
			return fmt.Errorf("function %q: invalid identifier", name)
		}
		if IsBuiltinName(name) {
			return fmt.Errorf("function %q: cannot shadow builtin %q", name, name)
		}
		expr, err := Parse(body)
		if err != nil {
			return fmt.Errorf("function %q: %w", name, err)
		}
		parsed[name] = expr
	}

	// Phase 2: build call-graph adjacency list and check for cycles
	inDegree := make(map[string]int, len(parsed))
	graph := make(map[string][]string, len(parsed))

	// Initialize in-degrees
	for name := range parsed {
		inDegree[name] = 0
	}

	// Build edges: for each function, find calls to other user functions
	for name, expr := range parsed {
		calls := CollectFuncCalls(expr)
		for _, callee := range calls {
			if _, isUser := parsed[callee]; isUser {
				graph[name] = append(graph[name], callee)
				inDegree[callee]++
			}
		}
	}

	// Kahn's algorithm for topological sort + cycle detection
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	visited := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++
		for _, neighbor := range graph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visited != len(parsed) {
		// Find one node still in a cycle for the error message
		for name, deg := range inDegree {
			if deg > 0 {
				return fmt.Errorf("function %q: circular reference detected", name)
			}
		}
		return fmt.Errorf("circular reference detected in function definitions")
	}

	// Phase 3: store compiled expressions
	c.userFunctions = parsed
	return nil
}

// applySeverityMapping validates all mapping values and rewrites rule severities.
// Empty mapping has no effect (backward compatible).
func (c *Config) applySeverityMapping() error {
	if len(c.SeverityMapping) == 0 {
		return nil
	}

	// Validate all mapped values are valid severities
	validSeverities := map[string]bool{
		string(SeverityError):   true,
		string(SeverityWarning): true,
		string(SeverityInfo):    true,
	}

	for from, to := range c.SeverityMapping {
		if !validSeverities[to] {
			return fmt.Errorf("severity_mapping: %q maps to invalid severity %q (must be error, warning, or info)", from, to)
		}
	}

	// Apply mapping to all rules
	for i := range c.Rules {
		if mapped, ok := c.SeverityMapping[string(c.Rules[i].Severity)]; ok {
			c.Rules[i].Severity = Severity(mapped)
		}
		// Apply mapping to overrides
		for j := range c.Rules[i].Overrides {
			if mapped, ok := c.SeverityMapping[string(c.Rules[i].Overrides[j].Severity)]; ok {
				c.Rules[i].Overrides[j].Severity = Severity(mapped)
			}
		}
	}

	return nil
}

// validateTemplateLayerRefs checks that template params referencing layer names
// point to valid configured layers. Only checks params known to be layer names
// (from, to, layer, forbidden) — skips numeric params like max, min.
func validateTemplateLayerRefs(templateName string, params map[string]interface{}, layerNames map[string]bool, ruleIndex int) error {
	// Params that contain layer name references per template schema
	layerRefParams := map[string][]string{
		"max-deps":      {"from", "to"},
		"no-leak":       {"layer", "forbidden"},
		"layer-balance": {}, // no layer refs in layer-balance
	}

	refParams, ok := layerRefParams[templateName]
	if !ok {
		return nil // unknown template (already caught by Rule.Validate)
	}

	for _, param := range refParams {
		val, exists := params[param]
		if !exists {
			continue // missing param caught by ValidateTemplateParams
		}

		switch v := val.(type) {
		case string:
			if !layerNames[v] {
				return fmt.Errorf("rule[%d]: template param %q references unknown layer %q", ruleIndex, param, v)
			}
		case []interface{}:
			for j, elem := range v {
				if s, ok := elem.(string); ok && !layerNames[s] {
					return fmt.Errorf("rule[%d]: template param %q[%d] references unknown layer %q", ruleIndex, param, j, s)
				}
			}
		case []string:
			for j, s := range v {
				if !layerNames[s] {
					return fmt.Errorf("rule[%d]: template param %q[%d] references unknown layer %q", ruleIndex, param, j, s)
				}
			}
		}
	}

	return nil
}
