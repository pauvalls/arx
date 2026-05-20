package domain

import (
	"fmt"
	"regexp"
	"time"
)

// PluginConfig defines the configuration for an external plugin detector.
type PluginConfig struct {
	Name       string   `yaml:"name" json:"name"`
	Command    string   `yaml:"command" json:"command"`
	Args       []string `yaml:"args,omitempty" json:"args,omitempty"`
	Languages  []string `yaml:"languages" json:"languages"`
	Timeout    string   `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Extensions []string `yaml:"extensions,omitempty" json:"extensions,omitempty"`
}

var pluginNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// builtinDetectorNames is the set of built-in detector names that plugins cannot conflict with.
var builtinDetectorNames = map[string]bool{
	"go": true, "typescript": true, "python": true, "java": true,
	"kotlin": true, "rust": true, "csharp": true, "ruby": true,
	"swift": true, "php": true,
}

// Validate checks the plugin configuration for required fields and valid values.
func (p PluginConfig) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if !pluginNameRe.MatchString(p.Name) {
		return fmt.Errorf("plugin name %q must match %s", p.Name, pluginNameRe.String())
	}
	if builtinDetectorNames[p.Name] {
		return fmt.Errorf("plugin name %q conflicts with built-in detector name", p.Name)
	}
	if p.Command == "" {
		return fmt.Errorf("plugin %q: command is required", p.Name)
	}
	if len(p.Languages) == 0 {
		return fmt.Errorf("plugin %q: at least one language must be specified", p.Name)
	}
	if p.Timeout != "" {
		if _, err := time.ParseDuration(p.Timeout); err != nil {
			return fmt.Errorf("plugin %q: invalid timeout %q: %w", p.Name, p.Timeout, err)
		}
	}
	return nil
}
