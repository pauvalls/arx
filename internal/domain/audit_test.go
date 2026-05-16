package domain

import (
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
