package domain

import (
	"context"
	"errors"
	"testing"
)

// ─── T-02: WasmEvaluator interface contract ──────────────────────────────────

// mockEvaluator implements WasmEvaluator for testing.
type mockEvaluator struct {
	violations []Violation
	err        error
}

func (m *mockEvaluator) Evaluate(_ context.Context, _ []Dependency, _ []Layer, _ []Violation, _ map[string]interface{}) ([]Violation, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.violations, nil
}

func (m *mockEvaluator) Close() error { return nil }

func TestWasmEvaluator_Interface(t *testing.T) {
	// Verify that a concrete type can satisfy the WasmEvaluator interface.
	var e WasmEvaluator = &mockEvaluator{}
	_ = e // interface satisfaction at compile time; this line prevents "unused" lint
}

func TestWasmEvaluator_Evaluate(t *testing.T) {
	tests := []struct {
		name       string
		evaluator  *mockEvaluator
		wantCount  int
		wantErr    bool
	}{
		{
			name: "returns violations",
			evaluator: &mockEvaluator{
				violations: []Violation{
					{RuleID: "wasm-rule", Message: "violation from wasm"},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "returns empty when no violations",
			evaluator: &mockEvaluator{
				violations: []Violation{},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "returns error",
			evaluator: &mockEvaluator{
				err: errors.New("evaluation failed"),
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := tt.evaluator.Evaluate(context.Background(), nil, nil, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(violations) != tt.wantCount {
				t.Errorf("Evaluate() returned %d violations, want %d", len(violations), tt.wantCount)
			}
		})
	}
}

func TestWasmEvaluator_Close(t *testing.T) {
	e := &mockEvaluator{}
	if err := e.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// ─── T-01 + T-03: WasmConfig validation ───────────────────────────────────────

func TestWasmConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WasmConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with path only",
			cfg: WasmConfig{
				Path: "policies/layer-balance.wasm",
			},
			wantErr: false,
		},
		{
			name: "valid with path and params",
			cfg: WasmConfig{
				Path:   "policies/layer-balance.wasm",
				Params: map[string]interface{}{"min": int64(3), "max": int64(8)},
			},
			wantErr: false,
		},
		{
			name: "valid with empty params",
			cfg: WasmConfig{
				Path:   "policies/layer-balance.wasm",
				Params: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "invalid with empty path",
			cfg: WasmConfig{
				Path: "",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "invalid with nil params is ok",
			cfg: WasmConfig{
				Path: "policies/layer-balance.wasm",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("WasmConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("WasmConfig.Validate() expected error containing %q, got nil", tt.errMsg)
				} else if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("WasmConfig.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestRule_Validate_WasmField(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "rule with wasm only is valid",
			rule: Rule{
				ID:       "W1",
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Wasm: &WasmConfig{
					Path: "policies/layer-balance.wasm",
				},
			},
			wantErr: false,
		},
		{
			name: "rule with wasm and params is valid",
			rule: Rule{
				ID:       "W2",
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Wasm: &WasmConfig{
					Path:   "policies/layer-balance.wasm",
					Params: map[string]interface{}{"min": int64(3)},
				},
			},
			wantErr: false,
		},
		{
			name: "rule with wasm and check is invalid",
			rule: Rule{
				ID:       "W3",
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Wasm: &WasmConfig{
					Path: "policies/layer-balance.wasm",
				},
				Check: CheckExpr{Raw: "count(deps()) > 0"},
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "rule with wasm and empty path is invalid",
			rule: Rule{
				ID:       "W4",
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Wasm: &WasmConfig{
					Path: "",
				},
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "traditional rule without wasm is unaffected",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: false,
		},
		{
			name: "rule with wasm and from/to is valid (hybrid)",
			rule: Rule{
				ID:       "W5",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Wasm: &WasmConfig{
					Path: "policies/layer-balance.wasm",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Rule.Validate() expected error containing %q, got nil", tt.errMsg)
				} else if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Rule.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
