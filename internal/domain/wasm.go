package domain

import (
	"context"
	"fmt"
)

// WasmConfig holds the configuration for a WASM-based policy rule.
type WasmConfig struct {
	Path   string                 `yaml:"path" json:"path"`
	Params map[string]interface{} `yaml:"params,omitempty" json:"params,omitempty"`
}

// Validate checks that the WasmConfig is valid.
func (w *WasmConfig) Validate() error {
	if w.Path == "" {
		return fmt.Errorf("wasm path is required")
	}
	return nil
}

// WasmEvaluator evaluates a WASM policy module against the current architecture state.
type WasmEvaluator interface {
	// Evaluate runs the WASM policy and returns any violations found.
	Evaluate(ctx context.Context, deps []Dependency, layers []Layer, violations []Violation, params map[string]interface{}) ([]Violation, error)
	// Close releases any resources held by the evaluator.
	Close() error
}
