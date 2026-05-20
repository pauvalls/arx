package github

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig holds the GitHub App configuration.
type AppConfig struct {
	AppID          int64  `json:"app_id" yaml:"app_id"`
	AppSlug        string `json:"app_slug,omitempty" yaml:"app_slug,omitempty"`
	InstallationID int64  `json:"installation_id" yaml:"installation_id"`
	PrivateKeyPath string `json:"private_key_path" yaml:"private_key_path"`
	PrivateKey     []byte `json:"-" yaml:"-"` // loaded from file, never serialized
	WebhookSecret  string `json:"-" yaml:"-"`
}

// PRCheckConfig holds PR check specific configuration.
type PRCheckConfig struct {
	AutoApprove    bool   `json:"auto_approve" yaml:"auto_approve"`
	AutoApproveOn  string `json:"auto_approve_on" yaml:"auto_approve_on"`
	SummaryFormat  string `json:"summary_format" yaml:"summary_format"`
}

// Config is the top-level configuration for GitHub integration.
type Config struct {
	Version   string      `yaml:"version"`
	GitHubApp AppConfig   `yaml:"github_app"`
	Webhook   struct {
		Secret string `yaml:"secret"`
	} `yaml:"webhook"`
	PRCheck PRCheckConfig `yaml:"pr_check"`
}

// LoadConfig reads and parses the arx-config.yaml file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config file %s is empty", path)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Copy webhook secret from the nested struct
	cfg.GitHubApp.WebhookSecret = cfg.Webhook.Secret

	return &cfg, nil
}

// LoadPrivateKey reads the private key file for the GitHub App.
func LoadPrivateKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading private key %s: %w", path, err)
	}
	return data, nil
}

// LoadFullConfig loads the config and also reads the private key file.
func LoadFullConfig(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	if cfg.GitHubApp.PrivateKeyPath != "" {
		keyData, err := LoadPrivateKey(cfg.GitHubApp.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("loading private key: %w", err)
		}
		cfg.GitHubApp.PrivateKey = keyData
	}

	return cfg, nil
}

// AutoApproveEnabled returns true if the config enables auto-approval for PR checks.
func AutoApproveEnabled(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.PRCheck.AutoApprove
}
