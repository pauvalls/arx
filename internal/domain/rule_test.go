package domain

import (
	"testing"
)

func TestRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid cannot rule",
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
			name: "valid must rule",
			rule: Rule{
				ID:       "R2",
				From:     "application",
				To:       []string{"domain"},
				Type:     RuleTypeMust,
				Severity: SeverityWarning,
			},
			wantErr: false,
		},
		{
			name: "valid can rule",
			rule: Rule{
				ID:       "R3",
				From:     "application",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCan,
				Severity: SeverityInfo,
			},
			wantErr: false,
		},
		{
			name: "valid must not circular rule",
			rule: Rule{
				ID:       "R4",
				From:     "domain",
				To:       []string{"application"},
				Type:     RuleTypeMustNotCircular,
				Severity: SeverityError,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			rule: Rule{
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "rule ID is required",
		},
		{
			name: "missing from",
			rule: Rule{
				ID:       "R1",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "'from' field is required",
		},
		{
			name: "missing to",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "'to' field must have at least one target",
		},
		{
			name: "invalid rule type",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     "InvalidType",
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "invalid rule type",
		},
		{
			name: "invalid severity",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: "InvalidSeverity",
			},
			wantErr: true,
			errMsg:  "invalid severity",
		},
		{
			name: "empty severity defaults to valid",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Rule.Validate() expected error containing %q, got nil", tt.errMsg)
				}
			}
		})
	}
}

func TestRule_Violates(t *testing.T) {
	tests := []struct {
		name        string
		rule        Rule
		importPath  string
		sourceLayer string
		targetLayer string
		want        bool
	}{
		{
			name: "cannot rule - violated",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/infrastructure/db",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        true,
		},
		{
			name: "cannot rule - different source",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/infrastructure/db",
			sourceLayer: "application",
			targetLayer: "infrastructure",
			want:        false,
		},
		{
			name: "cannot rule - different target",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/application",
			sourceLayer: "domain",
			targetLayer: "application",
			want:        false,
		},
		{
			name: "must rule - not violated (dependency exists as required)",
			rule: Rule{
				ID:       "R2",
				From:     "application",
				To:       []string{"domain"},
				Type:     RuleTypeMust,
				Severity: SeverityWarning,
			},
			importPath:  "github.com/example/arx/internal/domain/user",
			sourceLayer: "application",
			targetLayer: "domain",
			want:        false,
		},
		{
			name: "can rule - never violated",
			rule: Rule{
				ID:       "R3",
				From:     "application",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCan,
				Severity: SeverityInfo,
			},
			importPath:  "github.com/example/arx/internal/infrastructure/db",
			sourceLayer: "application",
			targetLayer: "infrastructure",
			want:        false,
		},
		{
			name: "must not circular - violated",
			rule: Rule{
				ID:       "R4",
				From:     "domain",
				To:       []string{"application"},
				Type:     RuleTypeMustNotCircular,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/application/service",
			sourceLayer: "domain",
			targetLayer: "application",
			want:        true,
		},
		{
			name: "multiple targets - first target violated",
			rule: Rule{
				ID:       "R5",
				From:     "domain",
				To:       []string{"infrastructure", "application"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/infrastructure/db",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        true,
		},
		{
			name: "multiple targets - second target violated",
			rule: Rule{
				ID:       "R5",
				From:     "domain",
				To:       []string{"infrastructure", "application"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			importPath:  "github.com/example/arx/internal/application/service",
			sourceLayer: "domain",
			targetLayer: "application",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Violates(tt.importPath, tt.sourceLayer, tt.targetLayer)
			if got != tt.want {
				t.Errorf("Rule.Violates() = %v, want %v", got, tt.want)
			}
		})
	}
}
