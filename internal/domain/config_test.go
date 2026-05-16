package domain

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"infrastructure"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: Config{
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{},
			},
			wantErr: true,
			errMsg:  "config version is required",
		},
		{
			name: "missing layers",
			config: Config{
				Version: "1.0.0",
				Layers:  []Layer{},
				Rules:   []Rule{},
			},
			wantErr: true,
			errMsg:  "at least one layer must be defined",
		},
		{
			name: "duplicate layer names",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "domain", Paths: []string{"pkg/domain"}},
				},
				Rules: []Rule{},
			},
			wantErr: true,
			errMsg:  "duplicate layer name",
		},
		{
			name: "rule references unknown layer in from",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "unknown",
						To:       []string{"domain"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: true,
			errMsg:  "references unknown layer",
		},
		{
			name: "rule references unknown layer in to",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"unknown"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: true,
			errMsg:  "references unknown layer",
		},
		{
			name: "valid config with all optional fields",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{
						Name:        "domain",
						Paths:       []string{"internal/domain"},
						Description: "Domain layer",
						Tags:        []string{"core"},
					},
					{
						Name:  "infrastructure",
						Paths: []string{"internal/infrastructure"},
					},
				},
				Rules: []Rule{
					{
						ID:          "R1",
						From:        "domain",
						To:          []string{"infrastructure"},
						Type:        RuleTypeCannot,
						Severity:    SeverityError,
						Explanation: "Domain should be pure",
					},
				},
				LanguageOverrides: map[string]LanguageOverride{
					"go": {Extensions: []string{".go"}},
				},
				Exclude: []string{"vendor", "test"},
				SeverityConfig: map[Severity]SeverityConfig{
					SeverityError: {FailBuild: true, ShowInUI: true},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Config.Validate() expected error containing %q, got nil", tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_Validate_Pattern(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with pattern-only rule",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R-P1",
						Pattern:  `com/legacy/.*`,
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with combined pattern+from/to rule",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "application", Paths: []string{"internal/application"}},
				},
				Rules: []Rule{
					{
						ID:       "R-P2",
						From:     "domain",
						To:       []string{"application"},
						Pattern:  `com/legacy/.*`,
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid regex in rule pattern fails with rule ID in error",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R-BAD",
						Pattern:  `[invalid`,
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Config.Validate() expected error containing %q, got nil", tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_Hash(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "basic config produces consistent hash",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"infrastructure"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
				},
			},
		},
		{
			name: "config with optional fields",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules:   []Rule{},
				Exclude: []string{"vendor"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1, err := tt.config.Hash()
			if err != nil {
				t.Fatalf("Config.Hash() error = %v", err)
			}

			// Hash should be a 64-char hex string (SHA-256)
			if len(hash1) != 64 {
				t.Errorf("Config.Hash() length = %d, want 64", len(hash1))
			}

			// Same config should produce same hash (deterministic)
			hash2, err := tt.config.Hash()
			if err != nil {
				t.Fatalf("Config.Hash() second call error = %v", err)
			}
			if hash1 != hash2 {
				t.Errorf("Config.Hash() not deterministic: %q != %q", hash1, hash2)
			}

			// Two identical configs (different instances) should produce same hash
			config2 := tt.config
			hash3, err := config2.Hash()
			if err != nil {
				t.Fatalf("Config.Hash() on copy error = %v", err)
			}
			if hash1 != hash3 {
				t.Errorf("identical configs should produce same hash: %q != %q", hash1, hash3)
			}
		})
	}
}

func TestConfig_Hash_DifferentConfigs(t *testing.T) {
	config1 := Config{
		Version: "1.0.0",
		Layers:  []Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
		Rules:   []Rule{},
	}

	config2 := Config{
		Version: "2.0.0",
		Layers:  []Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
		Rules:   []Rule{},
	}

	hash1, err := config1.Hash()
	if err != nil {
		t.Fatalf("Config.Hash() error = %v", err)
	}

	hash2, err := config2.Hash()
	if err != nil {
		t.Fatalf("Config.Hash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("different configs should produce different hashes")
	}
}

func TestConfig_Hash_FieldChange(t *testing.T) {
	base := Config{
		Version: "1.0.0",
		Layers:  []Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
		Rules:   []Rule{},
	}

	baseHash, _ := base.Hash()

	// Changing each field should produce a different hash
	t.Run("version change", func(t *testing.T) {
		c := base
		c.Version = "2.0.0"
		h, _ := c.Hash()
		if h == baseHash {
			t.Error("version change should produce different hash")
		}
	})

	t.Run("layer change", func(t *testing.T) {
		c := base
		c.Layers = []Layer{{Name: "application", Paths: []string{"internal/application"}}}
		h, _ := c.Hash()
		if h == baseHash {
			t.Error("layer change should produce different hash")
		}
	})

	t.Run("rule change", func(t *testing.T) {
		c := base
		c.Rules = []Rule{{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: RuleTypeCannot, Severity: SeverityError}}
		h, _ := c.Hash()
		if h == baseHash {
			t.Error("rule change should produce different hash")
		}
	})

	t.Run("exclude change", func(t *testing.T) {
		c := base
		c.Exclude = []string{"vendor"}
		h, _ := c.Hash()
		if h == baseHash {
			t.Error("exclude change should produce different hash")
		}
	})
}

func TestConfig_Validate_RuleExcludes(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rule with exclude patterns",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"infrastructure"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
						Exclude:  []string{"internal/legacy/**", "vendor/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "rule with trailing slash exclude is valid",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"infrastructure"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
						Exclude:  []string{"internal/legacy/"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple rules with excludes",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
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
						Exclude:  []string{"vendor/**"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Config.Validate() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_MaxViolations_Validation(t *testing.T) {
	tests := []struct {
		name          string
		maxViolations int
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "zero threshold (backward compatible)",
			maxViolations: 0,
			wantErr:       false,
		},
		{
			name:          "positive threshold",
			maxViolations: 5,
			wantErr:       false,
		},
		{
			name:          "large positive threshold",
			maxViolations: 100,
			wantErr:       false,
		},
		{
			name:          "negative threshold rejected",
			maxViolations: -1,
			wantErr:       true,
			errMsg:        "max_violations cannot be negative",
		},
		{
			name:          "negative threshold rejected with value",
			maxViolations: -10,
			wantErr:       true,
			errMsg:        "max_violations cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules:         []Rule{},
				MaxViolations: tt.maxViolations,
			}

			err := config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil {
					t.Errorf("Config.Validate() expected error containing %q, got nil", tt.errMsg)
				} else if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestConfig_ViolationThreshold(t *testing.T) {
	tests := []struct {
		name          string
		maxViolations int
		wantThreshold int
	}{
		{
			name:          "zero returns zero",
			maxViolations: 0,
			wantThreshold: 0,
		},
		{
			name:          "positive value returned",
			maxViolations: 5,
			wantThreshold: 5,
		},
		{
			name:          "large value returned",
			maxViolations: 100,
			wantThreshold: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Version:       "1.0.0",
				Layers:        []Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
				Rules:         []Rule{},
				MaxViolations: tt.maxViolations,
			}

			got := config.ViolationThreshold()
			if got != tt.wantThreshold {
				t.Errorf("Config.ViolationThreshold() = %d, want %d", got, tt.wantThreshold)
			}
		})
	}
}

// ─── Template Rule Config Validation ─────────────────────────────────────────

func TestConfig_Validate_TemplateRules(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantErr    bool
		errContain string
	}{
		{
			name: "valid template rule passes",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "T1",
						Template: "max-deps",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"from": "domain",
							"to":   []interface{}{"infrastructure"},
							"max":  3,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unknown template fails at config level",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{
					{
						ID:       "T2",
						Template: "nonexistent",
						Severity: SeverityError,
						Params:   map[string]interface{}{},
					},
				},
			},
			wantErr:    true,
			errContain: `unknown template "nonexistent"`,
		},
		{
			name: "missing params fails at config level",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{
					{
						ID:       "T3",
						Template: "max-deps",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"to":  []interface{}{"infrastructure"},
							"max": 3,
						},
					},
				},
			},
			wantErr:    true,
			errContain: `missing required param "from"`,
		},
		{
			name: "template referencing unknown layer fails",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "T4",
						Template: "max-deps",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"from": "nonexistent-layer",
							"to":   []interface{}{"infrastructure"},
							"max":  3,
						},
					},
				},
			},
			wantErr:    true,
			errContain: `template param "from" references unknown layer`,
		},
		{
			name: "template with unknown target layer fails",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "T5",
						Template: "max-deps",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"from": "domain",
							"to":   []interface{}{"infrastructure", "unknown-layer"},
							"max":  3,
						},
					},
				},
			},
			wantErr:    true,
			errContain: `template param "to"`,
		},
		{
			name: "mixed standard + template rules — all valid",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "application", Paths: []string{"internal/application"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "R1",
						From:     "domain",
						To:       []string{"infrastructure"},
						Type:     RuleTypeCannot,
						Severity: SeverityError,
					},
					{
						ID:       "T6",
						Template: "no-leak",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"layer":     "domain",
							"forbidden": []interface{}{"infrastructure"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no-leak template with unknown forbidden layer",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
					{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
				},
				Rules: []Rule{
					{
						ID:       "T7",
						Template: "no-leak",
						Severity: SeverityError,
						Params: map[string]interface{}{
							"layer":     "domain",
							"forbidden": []interface{}{"infrastructure", "unknown"},
						},
					},
				},
			},
			wantErr:    true,
			errContain: `template param "forbidden"`,
		},
		{
			name: "layer-balance template — no layer refs to validate",
			config: Config{
				Version: "1.0.0",
				Layers: []Layer{
					{Name: "domain", Paths: []string{"internal/domain"}},
				},
				Rules: []Rule{
					{
						ID:       "T8",
						Template: "layer-balance",
						Severity: SeverityWarning,
						Params: map[string]interface{}{
							"min": 1,
							"max": 10,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil {
					t.Errorf("Config.Validate() expected error containing %q, got nil", tt.errContain)
				} else if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), tt.errContain)
				}
			}
		})
	}
}

// ─── Severity Mapping Tests ──────────────────────────────────────────────────

func TestConfig_SeverityMapping_Valid(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: "critical",
			},
		},
		SeverityMapping: map[string]string{
			"critical": "error",
			"minor":    "warning",
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error = %v", err)
	}

	if config.Rules[0].Severity != SeverityError {
		t.Errorf("expected severity %q, got %q", SeverityError, config.Rules[0].Severity)
	}
}

func TestConfig_SeverityMapping_InvalidTargetSeverity(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Rules: []Rule{},
		SeverityMapping: map[string]string{
			"critical": "invalid",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Config.Validate() expected error, got nil")
	}

	wantContain := "maps to invalid severity"
	if !strings.Contains(err.Error(), wantContain) {
		t.Errorf("Config.Validate() error = %q, want to contain %q", err.Error(), wantContain)
	}
}

func TestConfig_SeverityMapping_Empty(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityWarning,
			},
		},
		SeverityMapping: map[string]string{},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error = %v", err)
	}

	if config.Rules[0].Severity != SeverityWarning {
		t.Errorf("expected severity %q, got %q", SeverityWarning, config.Rules[0].Severity)
	}
}

func TestConfig_SeverityMapping_RuleSeverityRemapped(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: "critical",
			},
			{
				ID:       "R2",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: SeverityInfo,
			},
		},
		SeverityMapping: map[string]string{
			"critical": "error",
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error = %v", err)
	}

	if config.Rules[0].Severity != SeverityError {
		t.Errorf("rule 0: expected severity %q, got %q", SeverityError, config.Rules[0].Severity)
	}
	if config.Rules[1].Severity != SeverityInfo {
		t.Errorf("rule 1: expected severity %q (unchanged), got %q", SeverityInfo, config.Rules[1].Severity)
	}
}

func TestConfig_SeverityMapping_OverrideRemapped(t *testing.T) {
	config := Config{
		Version: "1.0.0",
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []Rule{
			{
				ID:       "R1",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     RuleTypeCannot,
				Severity: "critical",
				Overrides: []RuleOverride{
					{Path: "internal/legacy", Severity: "minor"},
				},
			},
		},
		SeverityMapping: map[string]string{
			"critical": "error",
			"minor":    "warning",
		},
	}

	err := config.Validate()
	if err != nil {
		t.Fatalf("Config.Validate() error = %v", err)
	}

	if config.Rules[0].Severity != SeverityError {
		t.Errorf("rule severity: expected %q, got %q", SeverityError, config.Rules[0].Severity)
	}
	if config.Rules[0].Overrides[0].Severity != SeverityWarning {
		t.Errorf("override severity: expected %q, got %q", SeverityWarning, config.Rules[0].Overrides[0].Severity)
	}
}

// ─── $schema Field Tests ─────────────────────────────────────────────────────

func TestConfig_SchemaField_MarshalsYAML(t *testing.T) {
	config := Config{
		Schema:  "./arx-schema.json",
		Version: "1.0",
		Layers:  []Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []Rule{},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "$schema:") {
		t.Errorf("YAML output missing $schema field:\n%s", content)
	}
	if !strings.Contains(content, "./arx-schema.json") {
		t.Errorf("YAML output missing schema URL:\n%s", content)
	}
}

func TestConfig_SchemaField_UnmarshalsYAML(t *testing.T) {
	yamlContent := `
$schema: "./arx-schema.json"
version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain/**"]
rules: []
`
	var config Config
	if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
		t.Fatalf("yaml.Unmarshal error = %v", err)
	}

	if config.Schema != "./arx-schema.json" {
		t.Errorf("Schema = %q, want %q", config.Schema, "./arx-schema.json")
	}
}

func TestConfig_SchemaField_OmittedWhenEmpty(t *testing.T) {
	config := Config{
		Version: "1.0",
		Layers:  []Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []Rule{},
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		t.Fatalf("yaml.Marshal error = %v", err)
	}

	if strings.Contains(string(data), "$schema") {
		t.Errorf("YAML output should not contain $schema when empty:\n%s", string(data))
	}
}
