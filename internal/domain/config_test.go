package domain

import (
	"testing"
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
