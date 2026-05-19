package domain

import (
	"strings"
	"testing"
)

// helper: create layers for tests
func testLayers() []Layer {
	return []Layer{
		{Name: "domain", Paths: []string{"internal/domain/"}},
		{Name: "application", Paths: []string{"internal/application/"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
	}
}

// helper: create deps for tests
func testDeps(deps []struct{ src, importPath, resolvedLayer string }) []Dependency {
	result := make([]Dependency, 0, len(deps))
	for _, d := range deps {
		result = append(result, Dependency{
			SourceFile:    d.src,
			SourceLine:    1,
			ImportPath:    d.importPath,
			ResolvedLayer: d.resolvedLayer,
		})
	}
	return result
}

// ─── ValidateTemplateParams ──────────────────────────────────────────────────

func TestValidateTemplateParams(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		params     map[string]interface{}
		wantErr    bool
		errContain string
	}{
		// max-deps
		{
			name:     "max-deps: valid params",
			template: "max-deps",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  3,
			},
			wantErr: false,
		},
		{
			name:     "max-deps: missing from",
			template: "max-deps",
			params: map[string]interface{}{
				"to":  []interface{}{"infrastructure"},
				"max": 3,
			},
			wantErr:    true,
			errContain: `missing required param "from"`,
		},
		{
			name:     "max-deps: wrong type for max (string)",
			template: "max-deps",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  "not-a-number",
			},
			wantErr:    true,
			errContain: "expected int",
		},
		{
			name:     "max-deps: wrong type for from (int)",
			template: "max-deps",
			params: map[string]interface{}{
				"from": 123,
				"to":   []interface{}{"infrastructure"},
				"max":  3,
			},
			wantErr:    true,
			errContain: "expected string",
		},
		{
			name:     "max-deps: to as []string",
			template: "max-deps",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []string{"infrastructure"},
				"max":  3,
			},
			wantErr: false,
		},
		{
			name:     "max-deps: max as float64 (YAML default)",
			template: "max-deps",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  float64(3),
			},
			wantErr: false,
		},
		// no-leak
		{
			name:     "no-leak: valid params",
			template: "no-leak",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure", "application"},
			},
			wantErr: false,
		},
		{
			name:     "no-leak: missing forbidden",
			template: "no-leak",
			params: map[string]interface{}{
				"layer": "domain",
			},
			wantErr:    true,
			errContain: `missing required param "forbidden"`,
		},
		{
			name:     "no-leak: forbidden element not string",
			template: "no-leak",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{123},
			},
			wantErr:    true,
			errContain: "expected []string",
		},
		// layer-balance
		{
			name:     "layer-balance: valid params",
			template: "layer-balance",
			params: map[string]interface{}{
				"min": 1,
				"max": 10,
			},
			wantErr: false,
		},
		{
			name:     "layer-balance: missing min",
			template: "layer-balance",
			params: map[string]interface{}{
				"max": 10,
			},
			wantErr:    true,
			errContain: `missing required param "min"`,
		},
		{
			name:     "layer-balance: max as float64",
			template: "layer-balance",
			params: map[string]interface{}{
				"min": float64(1),
				"max": float64(10),
			},
			wantErr: false,
		},
		// unknown template
		{
			name:       "unknown template",
			template:   "nonexistent",
			params:     map[string]interface{}{},
			wantErr:    true,
			errContain: `unknown template "nonexistent"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateParams(tt.template, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplateParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContain)
				} else if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
			}
		})
	}
}

// ─── TemplateMaxDeps ─────────────────────────────────────────────────────────

func TestTemplateMaxDeps(t *testing.T) {
	layers := testLayers()

	tests := []struct {
		name         string
		params       map[string]interface{}
		deps         []Dependency
		wantCount    int // expected number of violations
		wantMsgContains string
	}{
		{
			name: "under threshold — no violation",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  3,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/user.go", "internal/infrastructure/cache", "infrastructure"},
			}),
			wantCount: 0,
		},
		{
			name: "over threshold — one violation",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  1,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/order.go", "internal/infrastructure/cache", "infrastructure"},
				{"internal/domain/product.go", "internal/infrastructure/queue", "infrastructure"},
			}),
			wantCount:       1,
			wantMsgContains: "has 3 dependencies to",
		},
		{
			name: "max=0 — any dep is violation",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  0,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount:       1,
			wantMsgContains: "has 1 dependencies to",
		},
		{
			name: "multiple targets — count all together",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure", "application"},
				"max":  2,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/order.go", "internal/application/service", "application"},
				{"internal/domain/product.go", "internal/infrastructure/cache", "infrastructure"},
			}),
			wantCount:       1,
			wantMsgContains: "has 3 dependencies to",
		},
		{
			name: "deps from other source layer — not counted",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  0,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/application/service.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount: 0,
		},
		{
			name: "deps to other target layer — not counted",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  0,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/application/service", "application"},
			}),
			wantCount: 0,
		},
		{
			name: "exact threshold — no violation",
			params: map[string]interface{}{
				"from": "domain",
				"to":   []interface{}{"infrastructure"},
				"max":  2,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/order.go", "internal/infrastructure/cache", "infrastructure"},
			}),
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := TemplateMaxDeps(tt.params, tt.deps, layers)
			if len(violations) != tt.wantCount {
				t.Errorf("TemplateMaxDeps() returned %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("  violation[%d]: %s", i, v.Message)
				}
				return
			}
			if tt.wantMsgContains != "" && len(violations) > 0 {
				if !strings.Contains(violations[0].Message, tt.wantMsgContains) {
					t.Errorf("message %q should contain %q", violations[0].Message, tt.wantMsgContains)
				}
			}
		})
	}
}

// ─── TemplateNoLeak ──────────────────────────────────────────────────────────

func TestTemplateNoLeak(t *testing.T) {
	layers := testLayers()

	tests := []struct {
		name      string
		params    map[string]interface{}
		deps      []Dependency
		wantCount int
	}{
		{
			name: "no forbidden imports — no violation",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure"},
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/application/service.go", "internal/domain/user", "domain"},
				{"internal/application/service.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount: 0,
		},
		{
			name: "single forbidden import — one violation",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure"},
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount: 1,
		},
		{
			name: "multiple forbidden layers — each detected",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure", "application"},
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/order.go", "internal/application/service", "application"},
			}),
			wantCount: 2,
		},
		{
			name: "multiple forbidden imports from same file",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure"},
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/user.go", "internal/infrastructure/cache", "infrastructure"},
			}),
			wantCount: 2,
		},
		{
			name: "non-matching source layer — no violation",
			params: map[string]interface{}{
				"layer":     "domain",
				"forbidden": []interface{}{"infrastructure"},
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/application/service.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := TemplateNoLeak(tt.params, tt.deps, layers)
			if len(violations) != tt.wantCount {
				t.Errorf("TemplateNoLeak() returned %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("  violation[%d]: %s", i, v.Message)
				}
			}
			// Verify violation message format
			for _, v := range violations {
				if !strings.Contains(v.Message, "imports") || !strings.Contains(v.Message, "from forbidden layer") {
					t.Errorf("unexpected message format: %q", v.Message)
				}
			}
		})
	}
}

// ─── TemplateLayerBalance ────────────────────────────────────────────────────

func TestTemplateLayerBalance(t *testing.T) {
	layers := testLayers()

	tests := []struct {
		name      string
		params    map[string]interface{}
		deps      []Dependency
		wantCount int
	}{
		{
			name: "within range — no violation",
			params: map[string]interface{}{
				"min": 0,
				"max": 10,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/application/service", "application"},
				{"internal/application/service.go", "internal/infrastructure/db", "infrastructure"},
			}),
			wantCount: 0,
		},
		{
			name: "below min — violation",
			params: map[string]interface{}{
				"min": 5,
				"max": 10,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/application/service", "application"},
			}),
			wantCount: 3, // domain has 1 (min 5), application has 0 (min 5), infrastructure has 0 (min 5)
		},
		{
			name: "above max — violation",
			params: map[string]interface{}{
				"min": 0,
				"max": 1,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/application/service", "application"},
				{"internal/domain/order.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/domain/product.go", "internal/infrastructure/cache", "infrastructure"},
			}),
			wantCount: 1, // domain has 3 (max 1)
		},
		{
			name: "empty deps — all layers below min",
			params: map[string]interface{}{
				"min": 1,
				"max": 10,
			},
			deps:      []Dependency{},
			wantCount: 3, // all 3 layers have 0 deps (min 1)
		},
		{
			name: "both min and max violations",
			params: map[string]interface{}{
				"min": 2,
				"max": 2,
			},
			deps: testDeps([]struct{ src, importPath, resolvedLayer string }{
				{"internal/domain/user.go", "internal/application/service", "application"},
				{"internal/application/service.go", "internal/infrastructure/db", "infrastructure"},
				{"internal/application/handler.go", "internal/domain/user", "domain"},
			}),
			wantCount: 2, // domain: 1 (min 2), application: 2 (ok), infrastructure: 0 (min 2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := TemplateLayerBalance(tt.params, tt.deps, layers)
			if len(violations) != tt.wantCount {
				t.Errorf("TemplateLayerBalance() returned %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("  violation[%d]: %s -> %s", i, v.SourceLayer, v.Message)
				}
			}
			// Verify message format
			for _, v := range violations {
				if !strings.Contains(v.Message, "dependencies") {
					t.Errorf("unexpected message format: %q", v.Message)
				}
			}
		})
	}
}

// ─── Backward Compatibility ──────────────────────────────────────────────────

func TestTemplateBackwardCompat(t *testing.T) {
	// A rule without a template field should behave identically.
	// TemplateRegistry lookup for empty string should not be called.
	rule := Rule{
		ID:       "R1",
		From:     "domain",
		To:       []string{"infrastructure"},
		Type:     RuleTypeCannot,
		Severity: SeverityError,
	}

	// Validate should succeed without template field
	if err := rule.Validate(); err != nil {
		t.Errorf("Rule.Validate() with no template: %v", err)
	}

	// Template field is empty — EvaluateRules should skip template path
	// (verified by the fact that standard rules still work)
	deps := []Dependency{
		{
			SourceFile:    "internal/domain/user.go",
			SourceLine:    10,
			ImportPath:    "internal/infrastructure/db",
			ResolvedLayer: "infrastructure",
		},
	}
	layers := testLayers()
	violations := EvaluateRules(deps, []Rule{rule}, layers)

	if len(violations) != 1 {
		t.Errorf("expected 1 violation for traditional rule, got %d", len(violations))
	}
	if violations[0].RuleID != "R1" {
		t.Errorf("expected rule ID R1, got %s", violations[0].RuleID)
	}
}

// ─── Missing params error ────────────────────────────────────────────────────

// ─── checkParamType Edge Cases ─────────────────────────────────────────────

func TestCheckParamType_DefaultBranches(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		val          interface{}
		expectedType string
		wantErr      bool
		errContain   string
	}{
		{
			name:         "[]string default — int value",
			key:          "to",
			val:          42,
			expectedType: "[]string",
			wantErr:      true,
			errContain:   "expected []string, got int",
		},
		{
			name:         "[]string default — bool value",
			key:          "to",
			val:          true,
			expectedType: "[]string",
			wantErr:      true,
			errContain:   "expected []string, got bool",
		},
		{
			name:         "int default — bool value",
			key:          "max",
			val:          true,
			expectedType: "int",
			wantErr:      true,
			errContain:   "expected int, got bool",
		},
		{
			name:         "int default — map value",
			key:          "max",
			val:          map[string]int{},
			expectedType: "int",
			wantErr:      true,
			errContain:   "expected int, got map",
		},
		{
			name:         "unknown expected type",
			key:          "test",
			val:          "hello",
			expectedType: "unknown-type",
			wantErr:      true,
			errContain:   `unknown expected type "unknown-type"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkParamType(tt.key, tt.val, tt.expectedType)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkParamType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContain)
				} else if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
			}
		})
	}
}

// ─── toInt Edge Cases ───────────────────────────────────────────────────────

func TestToInt(t *testing.T) {
	tests := []struct {
		name  string
		val   interface{}
		want  int
	}{
		{
			name: "int value",
			val:  42,
			want: 42,
		},
		{
			name: "float64 value (YAML default)",
			val:  float64(3),
			want: 3,
		},
		{
			name: "string numeric value",
			val:  "7",
			want: 7,
		},
		{
			name: "string non-numeric defaults to 0",
			val:  "hello",
			want: 0,
		},
		{
			name: "bool value defaults to 0",
			val:  true,
			want: 0,
		},
		{
			name: "nil value defaults to 0",
			val:  nil,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInt(tt.val)
			if got != tt.want {
				t.Errorf("toInt(%v) = %d, want %d", tt.val, got, tt.want)
			}
		})
	}
}

// ─── resolveSourceLayer Edge Cases ─────────────────────────────────────────

func TestResolveSourceLayer(t *testing.T) {
	layers := []Layer{
		{Name: "domain", Paths: []string{"internal/domain/"}},
		{Name: "application", Paths: []string{"internal/application/"}},
	}

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{
			name:     "matching path returns layer name",
			filePath: "internal/domain/user.go",
			want:     "domain",
		},
		{
			name:     "non-matching path returns empty",
			filePath: "external/pkg/main.go",
			want:     "",
		},
		{
			name:     "empty path returns empty",
			filePath: "",
			want:     "",
		},
		{
			name:     "path with no segments returns empty",
			filePath: "random",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSourceLayer(tt.filePath, layers)
			if got != tt.want {
				t.Errorf("resolveSourceLayer(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

// ─── toStrSlice Edge Cases ─────────────────────────────────────────────────

func TestToStrSlice(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want []string
	}{
		{
			name: "[]string value",
			val:  []string{"a", "b"},
			want: []string{"a", "b"},
		},
		{
			name: "[]interface{} with strings",
			val:  []interface{}{"x", "y"},
			want: []string{"x", "y"},
		},
		{
			name: "[]interface{} with non-string elements skips them",
			val:  []interface{}{"a", 42, "b"},
			want: []string{"a", "b"},
		},
		{
			name: "default — non-slice value returns nil",
			val:  "hello",
			want: nil,
		},
		{
			name: "default — int value returns nil",
			val:  42,
			want: nil,
		},
		{
			name: "default — nil returns nil",
			val:  nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStrSlice(tt.val)
			if len(got) != len(tt.want) {
				t.Errorf("toStrSlice(%v) = %v (len %d), want %v (len %d)", tt.val, got, len(got), tt.want, len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("toStrSlice(%v)[%d] = %q, want %q", tt.val, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTemplateMissingParamsError(t *testing.T) {
	layers := testLayers()

	// Calling a template with missing params should not panic
	// and should return reasonable results (the template may produce
	// empty results or work with zero values)
	t.Run("max-deps with empty params", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TemplateMaxDeps panicked with: %v", r)
			}
		}()
		violations := TemplateMaxDeps(map[string]interface{}{}, []Dependency{}, layers)
		// With empty params, from="" matches nothing, so 0 violations
		if len(violations) != 0 {
			t.Errorf("expected 0 violations with empty params, got %d", len(violations))
		}
	})

	t.Run("no-leak with empty params", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TemplateNoLeak panicked with: %v", r)
			}
		}()
		violations := TemplateNoLeak(map[string]interface{}{}, []Dependency{}, layers)
		if len(violations) != 0 {
			t.Errorf("expected 0 violations with empty params, got %d", len(violations))
		}
	})

	t.Run("layer-balance with empty params", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TemplateLayerBalance panicked with: %v", r)
			}
		}()
		violations := TemplateLayerBalance(map[string]interface{}{}, []Dependency{}, layers)
		// min=0, max=0 → all layers with 0 deps are at threshold (not below min)
		// but if deps exist, they'd exceed max=0
		_ = violations
	})
}
