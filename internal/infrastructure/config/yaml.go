package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

// YAMLReader implements the ports.ConfigReader interface for YAML files
type YAMLReader struct{}

// NewYAMLReader creates a new YAML config reader
func NewYAMLReader() *YAMLReader {
	return &YAMLReader{}
}

// Read reads and parses a YAML configuration file
func (r *YAMLReader) Read(configPath string) (*domain.Config, error) {
	// Resolve to absolute path if relative
	if !filepath.IsAbs(configPath) {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("resolving config path: %w", err)
		}
		configPath = absPath
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Pipeline: interpolate env vars (first pass — for include paths)
	data, err = InterpolateEnvVars(data)
	if err != nil {
		return nil, fmt.Errorf("interpolating env vars: %w", err)
	}

	// Pipeline: resolve !include tags relative to config directory
	configDir := filepath.Dir(configPath)
	data, err = ResolveIncludes(configDir, data)
	if err != nil {
		return nil, fmt.Errorf("resolving includes: %w", err)
	}

	// Pipeline: interpolate env vars (second pass — for config values)
	data, err = InterpolateEnvVars(data)
	if err != nil {
		return nil, fmt.Errorf("interpolating env vars: %w", err)
	}

	// Parse YAML
	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration structure
func (r *YAMLReader) Validate(config *domain.Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	return config.Validate()
}

// Ensure YAMLReader implements ports.ConfigReader interface
var _ ports.ConfigReader = (*YAMLReader)(nil)
