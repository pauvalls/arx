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
