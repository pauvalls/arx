package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// LanguageOverride allows language-specific configuration overrides
type LanguageOverride struct {
	Extensions []string `json:"extensions" yaml:"extensions"`
	Comment    string   `json:"comment,omitempty" yaml:"comment,omitempty"`
	Import     string   `json:"import,omitempty" yaml:"import,omitempty"`
}

// SeverityConfig allows customization of severity behavior
type SeverityConfig struct {
	FailBuild bool `json:"fail_build" yaml:"fail_build"`
	ShowInUI  bool `json:"show_in_ui" yaml:"show_in_ui"`
}

// Config represents the complete Arx configuration
type Config struct {
	Version        string                      `json:"version" yaml:"version"`
	Layers         []Layer                     `json:"layers" yaml:"layers"`
	Rules          []Rule                      `json:"rules" yaml:"rules"`
	LanguageOverrides map[string]LanguageOverride `json:"language_overrides,omitempty" yaml:"language_overrides,omitempty"`
	Exclude        []string                    `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	SeverityConfig map[Severity]SeverityConfig `json:"severity_config,omitempty" yaml:"severity_config,omitempty"`
	MaxViolations  int                         `json:"max_violations,omitempty" yaml:"max_violations,omitempty"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("config version is required")
	}

	if len(c.Layers) == 0 {
		return fmt.Errorf("at least one layer must be defined")
	}

	// Validate max_violations threshold
	if c.MaxViolations < 0 {
		return fmt.Errorf("max_violations cannot be negative (got %d)", c.MaxViolations)
	}

	// Validate all layers
	layerNames := make(map[string]bool)
	for i := range c.Layers {
		if err := c.Layers[i].Validate(); err != nil {
			return fmt.Errorf("layer[%d]: %w", i, err)
		}
		if layerNames[c.Layers[i].Name] {
			return fmt.Errorf("duplicate layer name: %q", c.Layers[i].Name)
		}
		layerNames[c.Layers[i].Name] = true
	}

	// Validate all rules
	for i := range c.Rules {
		if err := c.Rules[i].Validate(); err != nil {
			return fmt.Errorf("rule[%d]: %w", i, err)
		}
		// Check that rule references valid layers (skip pattern-only rules)
		if c.Rules[i].From != "" {
			if !layerNames[c.Rules[i].From] {
				return fmt.Errorf("rule[%d]: 'from' references unknown layer %q", i, c.Rules[i].From)
			}
		}
		for _, to := range c.Rules[i].To {
			if !layerNames[to] {
				return fmt.Errorf("rule[%d]: 'to' references unknown layer %q", i, to)
			}
		}
	}

	return nil
}

// Hash returns a SHA-256 hex digest of the marshaled JSON config.
// Used for cache invalidation and baseline staleness checks.
func (c *Config) Hash() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// ViolationThreshold returns the maximum number of violations allowed before failing.
// Returns 0 if no threshold is set (unlimited, backward-compatible).
func (c *Config) ViolationThreshold() int {
	return c.MaxViolations
}
