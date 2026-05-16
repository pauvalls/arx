package domain

import (
	"regexp"
	"strings"
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

func TestRule_CompilePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "valid regex",
			pattern: `com/example/.*`,
			wantErr: false,
		},
		{
			name:    "invalid regex",
			pattern: `[invalid`,
			wantErr: true,
		},
		{
			name:    "empty pattern is no-op",
			pattern: "",
			wantErr: false,
		},
		{
			name:    "valid regex with special chars",
			pattern: `com\.legacy\..+`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rule{Pattern: tt.pattern}
			err := r.CompilePattern()
			if (err != nil) != tt.wantErr {
				t.Errorf("CompilePattern() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.pattern != "" {
				if r.compiledPattern == nil {
					t.Error("CompilePattern() compiledPattern is nil for valid pattern")
				}
			}
			if tt.pattern == "" {
				if r.compiledPattern != nil {
					t.Error("CompilePattern() compiledPattern should be nil for empty pattern")
				}
			}
		})
	}
}

func TestRule_Violates_Pattern(t *testing.T) {
	// Pre-compile patterns for test rules
	validPattern, err := regexp.Compile(`com/legacy/.*`)
	if err != nil {
		t.Fatalf("failed to compile test pattern: %v", err)
	}
	prefixPattern, err := regexp.Compile(`com\.example\..*Controller`)
	if err != nil {
		t.Fatalf("failed to compile test pattern: %v", err)
	}

	tests := []struct {
		name        string
		rule        Rule
		importPath  string
		sourceLayer string
		targetLayer string
		want        bool
	}{
		{
			name: "pattern-only rule - matches",
			rule: Rule{
				ID:              "R-P1",
				Pattern:         `com/legacy/.*`,
				Type:            RuleTypeCannot,
				Severity:        SeverityError,
				compiledPattern: validPattern,
			},
			importPath:  "com/legacy/util/OldClass",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        true,
		},
		{
			name: "pattern-only rule - does not match",
			rule: Rule{
				ID:              "R-P2",
				Pattern:         `com/legacy/.*`,
				Type:            RuleTypeCannot,
				Severity:        SeverityError,
				compiledPattern: validPattern,
			},
			importPath:  "com/example/NewClass",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        false,
		},
		{
			name: "combined rule - both match",
			rule: Rule{
				ID:              "R-P3",
				From:            "application",
				To:              []string{"domain"},
				Pattern:         `com\.example\..*Controller`,
				Type:            RuleTypeCannot,
				Severity:        SeverityError,
				compiledPattern: prefixPattern,
			},
			importPath:  "com.example.UserController",
			sourceLayer: "application",
			targetLayer: "domain",
			want:        true,
		},
		{
			name: "combined rule - pattern does not match",
			rule: Rule{
				ID:              "R-P4",
				From:            "application",
				To:              []string{"domain"},
				Pattern:         `com\.example\..*Controller`,
				Type:            RuleTypeCannot,
				Severity:        SeverityError,
				compiledPattern: prefixPattern,
			},
			importPath:  "com.example.UserService",
			sourceLayer: "application",
			targetLayer: "domain",
			want:        false,
		},
		{
			name: "combined rule - from/to does not match",
			rule: Rule{
				ID:              "R-P5",
				From:            "application",
				To:              []string{"domain"},
				Pattern:         `com\.example\..*Controller`,
				Type:            RuleTypeCannot,
				Severity:        SeverityError,
				compiledPattern: prefixPattern,
			},
			importPath:  "com.example.UserController",
			sourceLayer: "infrastructure",
			targetLayer: "domain",
			want:        false,
		},
		{
			name: "pattern-only rule with Can type - not violated",
			rule: Rule{
				ID:              "R-P6",
				Pattern:         `com/legacy/.*`,
				Type:            RuleTypeCan,
				Severity:        SeverityInfo,
				compiledPattern: validPattern,
			},
			importPath:  "com/legacy/util/OldClass",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        false,
		},
		{
			name: "pattern-only rule with Must type - not violated",
			rule: Rule{
				ID:              "R-P7",
				Pattern:         `com/legacy/.*`,
				Type:            RuleTypeMust,
				Severity:        SeverityWarning,
				compiledPattern: validPattern,
			},
			importPath:  "com/legacy/util/OldClass",
			sourceLayer: "domain",
			targetLayer: "infrastructure",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Violates(tt.importPath, tt.sourceLayer, tt.targetLayer)
			if got != tt.want {
				t.Errorf("Rule.Violates() = %v, want %v (pattern=%q, import=%q, from=%s, to=%v)",
					got, tt.want, tt.rule.Pattern, tt.importPath, tt.rule.From, tt.rule.To)
			}
		})
	}
}

func TestRule_Validate_Pattern(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "pattern-only rule with valid regex",
			rule: Rule{
				ID:       "R-P1",
				Pattern:  `com/legacy/.*`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: false,
		},
		{
			name: "pattern-only rule with invalid regex",
			rule: Rule{
				ID:       "R-P2",
				Pattern:  `[invalid`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "invalid pattern",
		},
		{
			name: "combined rule (from/to + pattern) with valid regex",
			rule: Rule{
				ID:       "R-P3",
				From:     "application",
				To:       []string{"domain"},
				Pattern:  `com/legacy/.*`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: false,
		},
		{
			name: "combined rule (from/to + pattern) with invalid regex",
			rule: Rule{
				ID:       "R-P4",
				From:     "application",
				To:       []string{"domain"},
				Pattern:  `[invalid`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "invalid pattern",
		},
		{
			name: "pattern-only rule with empty to is allowed",
			rule: Rule{
				ID:       "R-P5",
				Pattern:  `com/legacy/.*`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: false,
		},
		{
			name: "pattern-only rule missing ID",
			rule: Rule{
				Pattern:  `com/legacy/.*`,
				Type:     RuleTypeCannot,
				Severity: SeverityError,
			},
			wantErr: true,
			errMsg:  "rule ID is required",
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

	// Verify the invalid regex cases actually cleared compiledPattern on error
	t.Run("invalid regex does not leave compiledPattern set", func(t *testing.T) {
		r := Rule{
			ID:      "R-ERR",
			Pattern: `[invalid`,
			Type:    RuleTypeCannot,
		}
		_ = r.Validate()
		if r.compiledPattern != nil {
			t.Error("compiledPattern should be nil after failed validation")
		}
	})
}

func TestRule_Validate_Overrides(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid severity override",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: SeverityWarning},
				},
			},
			wantErr: false,
		},
		{
			name: "valid disabled override",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Enabled: boolPtr(false)},
				},
			},
			wantErr: false,
		},
		{
			name: "valid override with both severity and enabled",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: SeverityWarning, Enabled: boolPtr(true)},
				},
			},
			wantErr: false,
		},
		{
			name: "valid empty path override (match all)",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "", Severity: SeverityInfo},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid override severity",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: "critical"},
				},
			},
			wantErr: true,
			errMsg:  "invalid severity",
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

func TestRule_IsExcludedFor(t *testing.T) {
	tests := []struct {
		name     string
		excludes []string
		filePath string
		want     bool
	}{
		{
			name:     "empty excludes - not excluded",
			excludes: []string{},
			filePath: "internal/domain/user.go",
			want:     false,
		},
		{
			name:     "exact match",
			excludes: []string{"internal/legacy/old.go"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "exact match - no match",
			excludes: []string{"internal/legacy/old.go"},
			filePath: "internal/legacy/new.go",
			want:     false,
		},
		{
			name:     "prefix match with trailing slash",
			excludes: []string{"internal/legacy/"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "prefix match - nested file",
			excludes: []string{"internal/legacy/"},
			filePath: "internal/legacy/deep/nested.go",
			want:     true,
		},
		{
			name:     "prefix match - different directory",
			excludes: []string{"internal/legacy/"},
			filePath: "internal/domain/user.go",
			want:     false,
		},
		{
			name:     "glob single star - matches",
			excludes: []string{"internal/legacy/*.go"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "glob single star - no match different extension",
			excludes: []string{"internal/legacy/*.go"},
			filePath: "internal/legacy/old.txt",
			want:     false,
		},
		{
			name:     "glob single star - no match nested",
			excludes: []string{"internal/legacy/*.go"},
			filePath: "internal/legacy/deep/old.go",
			want:     false,
		},
		{
			name:     "glob double star - matches nested",
			excludes: []string{"internal/legacy/**"},
			filePath: "internal/legacy/deep/nested.go",
			want:     true,
		},
		{
			name:     "glob double star - matches direct",
			excludes: []string{"internal/legacy/**"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "glob double star - no match different dir",
			excludes: []string{"internal/legacy/**"},
			filePath: "internal/domain/user.go",
			want:     false,
		},
		{
			name:     "multiple patterns - first matches",
			excludes: []string{"internal/legacy/**", "vendor/**"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "multiple patterns - second matches",
			excludes: []string{"internal/legacy/**", "vendor/**"},
			filePath: "vendor/some/pkg/file.go",
			want:     true,
		},
		{
			name:     "multiple patterns - none match",
			excludes: []string{"internal/legacy/**", "vendor/**"},
			filePath: "internal/domain/user.go",
			want:     false,
		},
		{
			name:     "glob star in middle of path",
			excludes: []string{"internal/*/old.go"},
			filePath: "internal/legacy/old.go",
			want:     true,
		},
		{
			name:     "glob star in middle - no match",
			excludes: []string{"internal/*/old.go"},
			filePath: "internal/domain/old.go",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Exclude:  tt.excludes,
			}
			// Compile exclude patterns
			if err := r.CompileExcludePatterns(); err != nil {
				t.Fatalf("CompileExcludePatterns() error = %v", err)
			}

			got := r.IsExcludedFor(tt.filePath)
			if got != tt.want {
				t.Errorf("IsExcludedFor(%q) = %v, want %v (excludes=%v)", tt.filePath, got, tt.want, tt.excludes)
			}
		})
	}
}

func TestRule_CompileExcludePatterns(t *testing.T) {
	tests := []struct {
		name     string
		excludes []string
		wantErr  bool
	}{
		{
			name:     "empty excludes",
			excludes: []string{},
			wantErr:  false,
		},
		{
			name:     "valid glob patterns",
			excludes: []string{"internal/legacy/**", "vendor/**"},
			wantErr:  false,
		},
		{
			name:     "valid single star pattern",
			excludes: []string{"*.go", "internal/*/*.go"},
			wantErr:  false,
		},
		{
			name:     "valid prefix pattern with trailing slash",
			excludes: []string{"internal/legacy/"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Exclude:  tt.excludes,
			}
			err := r.CompileExcludePatterns()
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileExcludePatterns() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(tt.excludes) > 0 {
				if len(r.compiledExclude) != len(tt.excludes) {
					t.Errorf("compiledExclude length = %d, want %d", len(r.compiledExclude), len(tt.excludes))
				}
			}
			if tt.wantErr && r.compiledExclude != nil {
				t.Error("compiledExclude should be nil after failed compilation")
			}
		})
	}
}

func TestRule_Validate_Exclude(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid exclude patterns",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Exclude:  []string{"internal/legacy/**", "vendor/**"},
			},
			wantErr: false,
		},
		{
			name: "empty exclude is valid",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Exclude:  []string{},
			},
			wantErr: false,
		},
		{
			name: "exclude with trailing slash is valid",
			rule: Rule{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityError,
				Exclude:  []string{"internal/legacy/"},
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
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Rule.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
