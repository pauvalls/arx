package domain

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestAudit_EvaluateRules(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []Dependency
		rules        []Rule
		layers       []Layer
		wantCount    int
		wantRuleIDs  []string
	}{
		{
			name: "single violation - domain cannot import infrastructure",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/domain/user.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount:   1,
			wantRuleIDs: []string{"R1"},
		},
		{
			name: "no violations - allowed dependency",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/application/service.go",
					SourceLine:    15,
					ImportPath:    "github.com/example/arx/internal/domain/user",
					ResolvedLayer: "domain",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "application", Paths: []string{"internal/application"}},
				{Name: "domain", Paths: []string{"internal/domain"}},
			},
			wantCount: 0,
		},
		{
			name: "multiple violations",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/domain/user.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
				{
					SourceFile:    "internal/domain/order.go",
					SourceLine:    20,
					ImportPath:    "github.com/example/arx/internal/infrastructure/cache",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount:   2,
			wantRuleIDs: []string{"R1", "R1"},
		},
		{
			name: "unresolved layers are skipped",
			dependencies: []Dependency{
				{
					SourceFile:    "unknown/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "must rule - no violation when dependency exists",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/application/service.go",
					SourceLine:    15,
					ImportPath:    "github.com/example/arx/internal/domain/user",
					ResolvedLayer: "domain",
				},
			},
			rules: []Rule{
				{
					ID:       "R2",
					From:     "application",
					To:       []string{"domain"},
					Type:     RuleTypeMust,
					Severity: SeverityWarning,
				},
			},
			layers: []Layer{
				{Name: "application", Paths: []string{"internal/application"}},
				{Name: "domain", Paths: []string{"internal/domain"}},
			},
			wantCount: 0,
		},
		{
			name: "empty dependencies - no violations",
			dependencies: []Dependency{},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "empty rules - no violations",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/domain/user.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules:    []Rule{},
			layers:   []Layer{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := EvaluateRules(tt.dependencies, tt.rules, tt.layers)

			if len(violations) != tt.wantCount {
				t.Errorf("EvaluateRules() returned %d violations, want %d", len(violations), tt.wantCount)
			}

			if tt.wantRuleIDs != nil {
				for i, v := range violations {
					if i < len(tt.wantRuleIDs) && v.RuleID != tt.wantRuleIDs[i] {
						t.Errorf("Violation[%d].RuleID = %q, want %q", i, v.RuleID, tt.wantRuleIDs[i])
					}
				}
			}

			// Check violation ID format
			for i, v := range violations {
				expectedID := GenerateViolationID(tt.rules[0], i+1)
				if v.ID != expectedID {
					t.Errorf("Violation[%d].ID = %q, want %q", i, v.ID, expectedID)
				}
			}
		})
	}
}

func TestAudit_EvaluateRules_Overrides(t *testing.T) {
	tests := []struct {
		name           string
		dependencies   []Dependency
		rules          []Rule
		layers         []Layer
		wantCount      int
		wantSeverity   Severity
		wantOverridden bool
	}{
		{
			name: "rule disabled for path skips violation",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "legacy",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Overrides: []RuleOverride{
						{Path: "internal/legacy/", Enabled: boolPtr(false)},
					},
				},
			},
			layers: []Layer{
				{Name: "legacy", Paths: []string{"internal/legacy"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "severity override changes violation severity",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "legacy",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Overrides: []RuleOverride{
						{Path: "internal/legacy/", Severity: SeverityWarning},
					},
				},
			},
			layers: []Layer{
				{Name: "legacy", Paths: []string{"internal/legacy"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount:      1,
			wantSeverity:   SeverityWarning,
			wantOverridden: true,
		},
		{
			name: "no override leaves violation unaffected",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/domain/user.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Overrides: []RuleOverride{
						{Path: "internal/legacy/", Severity: SeverityWarning},
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount:      1,
			wantSeverity:   SeverityError,
			wantOverridden: false,
		},
		{
			name: "override with both severity and enabled=false disables",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "legacy",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Overrides: []RuleOverride{
						{Path: "internal/legacy/", Severity: SeverityWarning, Enabled: boolPtr(false)},
					},
				},
			},
			layers: []Layer{
				{Name: "legacy", Paths: []string{"internal/legacy"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			// Disabled takes precedence — no violation
			wantCount: 0,
		},
		{
			name: "multiple overrides - disabled wins over severity override",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/deep/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "legacy",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Overrides: []RuleOverride{
						{Path: "internal/legacy/", Severity: SeverityWarning},
						{Path: "internal/legacy/deep/", Enabled: boolPtr(false)},
					},
				},
			},
			layers: []Layer{
				{Name: "legacy", Paths: []string{"internal/legacy"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			// Disabled by more specific override
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := EvaluateRules(tt.dependencies, tt.rules, tt.layers)

			if len(violations) != tt.wantCount {
				t.Errorf("EvaluateRules() returned %d violations, want %d", len(violations), tt.wantCount)
			}

			if tt.wantCount > 0 && len(violations) > 0 {
				v := violations[0]
				if v.Severity != tt.wantSeverity {
					t.Errorf("Violation.Severity = %q, want %q", v.Severity, tt.wantSeverity)
				}
				if v.Overridden != tt.wantOverridden {
					t.Errorf("Violation.Overridden = %v, want %v", v.Overridden, tt.wantOverridden)
				}
			}
		})
	}
}

func TestGenerateViolationID(t *testing.T) {
	tests := []struct {
		name  string
		rule  Rule
		index int
		want  string
	}{
		{
			name: "first violation",
			rule: Rule{ID: "R1"},
			index: 1,
			want:  "D-01",
		},
		{
			name: "tenth violation",
			rule: Rule{ID: "R1"},
			index: 10,
			want:  "D-10",
		},
		{
			name: "hundredth violation",
			rule: Rule{ID: "R1"},
			index: 100,
			want:  "D-100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateViolationID(tt.rule, tt.index)
			if got != tt.want {
				t.Errorf("GenerateViolationID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAudit_EvaluateRules_Excludes(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []Dependency
		rules        []Rule
		layers       []Layer
		wantCount    int
	}{
		{
			name: "excluded path - violation skipped",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/**"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "non-excluded path - violation reported",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/domain/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/**"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 1,
		},
		{
			name: "multiple rules with different excludes",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
				{
					SourceFile:    "internal/domain/file.go",
					SourceLine:    20,
					ImportPath:    "github.com/example/arx/internal/infrastructure/cache",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "legacy",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/**"},
				},
				{
					ID:       "R2",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityWarning,
					Exclude:  []string{"internal/domain/**"},
				},
			},
			layers: []Layer{
				{Name: "legacy", Paths: []string{"internal/legacy"}},
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			// First dep excluded by R1, second dep excluded by R2
			wantCount: 0,
		},
		{
			name: "mixed - some excluded, some not",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/file.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
				{
					SourceFile:    "internal/domain/file.go",
					SourceLine:    20,
					ImportPath:    "github.com/example/arx/internal/infrastructure/cache",
					ResolvedLayer: "infrastructure",
				},
				{
					SourceFile:    "internal/domain/another.go",
					SourceLine:    30,
					ImportPath:    "github.com/example/arx/internal/infrastructure/queue",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/**"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				{Name: "legacy", Paths: []string{"internal/legacy"}},
			},
			// First dep excluded (legacy), second and third reported (domain)
			wantCount: 2,
		},
		{
			name: "glob pattern exact match",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/old.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/old.go"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "glob pattern wildcard match",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/old.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/*.go"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
		{
			name: "glob pattern double star nested",
			dependencies: []Dependency{
				{
					SourceFile:    "internal/legacy/deep/nested.go",
					SourceLine:    10,
					ImportPath:    "github.com/example/arx/internal/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Exclude:  []string{"internal/legacy/**"},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := EvaluateRules(tt.dependencies, tt.rules, tt.layers)

			if len(violations) != tt.wantCount {
				t.Errorf("EvaluateRules() returned %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("Violation %d: RuleID=%q, File=%q", i, v.RuleID, v.File)
				}
			}
		})
	}
}

// ─── Template Rule Evaluation ────────────────────────────────────────────────

func TestAudit_EvaluateRules_TemplateRules(t *testing.T) {
	tests := []struct {
		name         string
		dependencies []Dependency
		rules        []Rule
		layers       []Layer
		wantCount    int
		wantIDs      []string // expected violation ID prefixes
	}{
		{
			name: "only template rules — max-deps violation",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/order.go", SourceLine: 2, ImportPath: "internal/infrastructure/cache", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/product.go", SourceLine: 3, ImportPath: "internal/infrastructure/queue", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "T1",
					Template: "max-deps",
					Severity: SeverityError,
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"infrastructure"},
						"max":  1,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			wantCount: 1,
			wantIDs:   []string{"T-"},
		},
		{
			name: "only template rules — no-leak violations",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 5, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/order.go", SourceLine: 10, ImportPath: "internal/application/service", ResolvedLayer: "application"},
			},
			rules: []Rule{
				{
					ID:       "T2",
					Template: "no-leak",
					Severity: SeverityError,
					Params: map[string]interface{}{
						"layer":     "domain",
						"forbidden": []interface{}{"infrastructure", "application"},
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "application", Paths: []string{"internal/application/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			wantCount: 2,
			wantIDs:   []string{"T-", "T-"},
		},
		{
			name: "only standard rules — backward compat",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			wantCount: 1,
			wantIDs:   []string{"D-"},
		},
		{
			name: "mixed standard + template rules — both produce violations",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/order.go", SourceLine: 2, ImportPath: "internal/infrastructure/cache", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/product.go", SourceLine: 3, ImportPath: "internal/infrastructure/queue", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
				{
					ID:       "T3",
					Template: "max-deps",
					Severity: SeverityWarning,
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"infrastructure"},
						"max":  1,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			// 3 standard violations (one per dep) + 1 template violation (max-deps exceeded)
			wantCount: 4,
		},
		{
			name: "template rule catches what standard rule misses",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/application/service", ResolvedLayer: "application"},
				{SourceFile: "internal/domain/user.go", SourceLine: 2, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				// Standard rule only blocks domain→infrastructure, allows domain→application
				{
					ID:       "R2",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
				// Template rule blocks domain from having >0 deps to application
				{
					ID:       "T4",
					Template: "max-deps",
					Severity: SeverityWarning,
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"application"},
						"max":  0,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "application", Paths: []string{"internal/application/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			// 1 standard violation (domain→infrastructure) + 1 template violation (domain→application exceeds max=0)
			wantCount: 2,
		},
		{
			name: "template rule with no violations",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "T5",
					Template: "max-deps",
					Severity: SeverityError,
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"infrastructure"},
						"max":  5,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			wantCount: 0,
		},
		{
			name: "hybrid rule — standard from/to AND template both evaluated",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
				{SourceFile: "internal/domain/order.go", SourceLine: 2, ImportPath: "internal/infrastructure/cache", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "H1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
					Template: "max-deps",
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"infrastructure"},
						"max":  0,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			// 2 standard violations (Cannot rule) + 1 template violation (max-deps exceeded)
			wantCount: 3,
		},
		{
			name: "sequential IDs with no gaps across standard and template",
			dependencies: []Dependency{
				{SourceFile: "internal/domain/user.go", SourceLine: 1, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
			},
			rules: []Rule{
				{
					ID:       "R1",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     RuleTypeCannot,
					Severity: SeverityError,
				},
				{
					ID:       "T6",
					Template: "max-deps",
					Severity: SeverityError,
					Params: map[string]interface{}{
						"from": "domain",
						"to":   []interface{}{"infrastructure"},
						"max":  0,
					},
				},
			},
			layers: []Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := EvaluateRules(tt.dependencies, tt.rules, tt.layers)

			if len(violations) != tt.wantCount {
				t.Errorf("EvaluateRules() returned %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("  violation[%d]: ID=%q RuleID=%q Message=%q", i, v.ID, v.RuleID, v.Message)
				}
				return
			}

			// Check ID prefixes if specified
			if tt.wantIDs != nil {
				for i, v := range violations {
					if i < len(tt.wantIDs) && !strings.HasPrefix(v.ID, tt.wantIDs[i]) {
						t.Errorf("Violation[%d].ID = %q, want prefix %q", i, v.ID, tt.wantIDs[i])
					}
				}
			}

			// Verify sequential IDs with no gaps
			for i, v := range violations {
				expectedIdx := i + 1
				// Extract number from ID (D-01, T-02, etc.)
				var num int
				if len(v.ID) >= 3 {
					fmt.Sscanf(v.ID[2:], "%d", &num)
				}
				if num != expectedIdx {
					t.Errorf("Violation[%d].ID = %q, expected index %d", i, v.ID, expectedIdx)
				}
			}
		})
	}
}

// ─── WASM Rule Evaluation ────────────────────────────────────────────────────

func TestEvaluateWasmRules_NoWasmRules(t *testing.T) {
	rules := []Rule{
		{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: RuleTypeCannot, Severity: SeverityError},
	}
	deps := []Dependency{
		{SourceFile: "internal/domain/user.go", ImportPath: "pkg/infra", ResolvedLayer: "infrastructure"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		return nil // should not be called
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, deps, layers, nil, evaluatorFn)
	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluateWasmRules_WithWasmRule(t *testing.T) {
	rules := []Rule{
		{
			ID:       "W1",
			Severity: SeverityError,
			Wasm:     &WasmConfig{Path: "policies/test.wasm"},
		},
	}
	deps := []Dependency{
		{SourceFile: "internal/domain/user.go", ImportPath: "pkg/infra", ResolvedLayer: "infrastructure"},
	}
	layers := []Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		if wasmPath == "policies/test.wasm" {
			return &mockEvaluator{
				violations: []Violation{
					{Message: "balance violation", File: "internal/domain/user.go"},
				},
			}
		}
		return nil
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, deps, layers, nil, evaluatorFn)
	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].RuleID != "W1" {
		t.Errorf("expected RuleID 'W1', got %q", violations[0].RuleID)
	}
	if violations[0].Message != "balance violation" {
		t.Errorf("expected Message 'balance violation', got %q", violations[0].Message)
	}
}

func TestEvaluateWasmRules_MultipleWasmRules(t *testing.T) {
	rules := []Rule{
		{
			ID:       "W1",
			Severity: SeverityError,
			Wasm:     &WasmConfig{Path: "policies/balance.wasm"},
		},
		{
			ID:       "W2",
			Severity: SeverityWarning,
			Wasm:     &WasmConfig{Path: "policies/symmetry.wasm"},
		},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		switch wasmPath {
		case "policies/balance.wasm":
			return &mockEvaluator{
				violations: []Violation{
					{Message: "balance violation", File: "file1.go"},
				},
			}
		case "policies/symmetry.wasm":
			return &mockEvaluator{
				violations: []Violation{
					{Message: "symmetry violation", File: "file2.go"},
				},
			}
		default:
			return nil
		}
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, nil, evaluatorFn)
	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(violations))
	}
}

func TestEvaluateWasmRules_EvaluatorError(t *testing.T) {
	rules := []Rule{
		{
			ID:   "W1",
			Wasm: &WasmConfig{Path: "policies/broken.wasm"},
		},
	}

	calls := 0
	evaluatorFn := func(wasmPath string) WasmEvaluator {
		calls++
		return nil // evaluator not found (error case)
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, nil, evaluatorFn)
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
	if calls != 1 {
		t.Errorf("expected 1 evaluatorFn call, got %d", calls)
	}
}

func TestEvaluateWasmRules_EvaluatorReturnsError(t *testing.T) {
	rules := []Rule{
		{
			ID:   "W1",
			Wasm: &WasmConfig{Path: "policies/test.wasm"},
		},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		return &mockEvaluator{err: fmt.Errorf("evaluation failed")}
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, nil, evaluatorFn)
	if errCount != 1 {
		t.Errorf("expected 1 error, got %d", errCount)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluateWasmRules_InheritsSeverity(t *testing.T) {
	rules := []Rule{
		{
			ID:       "W1",
			Severity: SeverityWarning,
			Wasm:     &WasmConfig{Path: "policies/test.wasm"},
		},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		return &mockEvaluator{
			violations: []Violation{
				{RuleID: "W1", Message: "test", File: "test.go"}, // no Severity set
			},
		}
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, nil, evaluatorFn)
	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Severity != SeverityWarning {
		t.Errorf("expected Severity %q, got %q", SeverityWarning, violations[0].Severity)
	}
}

func TestEvaluateWasmRules_MixedWithStandardRules(t *testing.T) {
	rules := []Rule{
		{
			ID:       "R1",
			From:     "domain",
			To:       []string{"infrastructure"},
			Type:     RuleTypeCannot,
			Severity: SeverityError,
		},
		{
			ID:       "W1",
			Severity: SeverityWarning,
			Wasm:     &WasmConfig{Path: "policies/test.wasm"},
		},
	}

	evaluatorFn := func(wasmPath string) WasmEvaluator {
		return &mockEvaluator{
			violations: []Violation{
				{Message: "wasm violation"},
			},
		}
	}

	// Standard rules are evaluated via EvaluateRules, WASM rules via EvaluateWasmRules
	standardViolations := EvaluateRules(nil, rules, nil)
	wasmViolations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, standardViolations, evaluatorFn)

	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	// Standard rules produce no violations (no deps), WASM produces 1
	if len(wasmViolations) != 1 {
		t.Fatalf("expected 1 wasm violation, got %d", len(wasmViolations))
	}
}

func TestEvaluateWasmRules_DisabledRule(t *testing.T) {
	rules := []Rule{
		{
			ID:   "W1",
			Wasm: &WasmConfig{Path: "policies/test.wasm"},
			Overrides: []RuleOverride{
				{Path: "", Enabled: boolPtr(false)},
			},
		},
	}

	called := false
	evaluatorFn := func(wasmPath string) WasmEvaluator {
		called = true
		return nil
	}

	violations, errCount := EvaluateWasmRules(context.Background(), rules, nil, nil, nil, evaluatorFn)
	if errCount != 0 {
		t.Errorf("expected 0 errors, got %d", errCount)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
	if called {
		t.Error("evaluatorFn should not be called for disabled rule")
	}
}
