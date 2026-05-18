package application

import (
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
	"gopkg.in/yaml.v3"
)

// PresetLoader loads raw YAML preset data by name.
type PresetLoader interface {
	LoadPreset(name string) ([]byte, error)
	ListPresets() []string
}

// PresetServiceImpl loads preset templates from a PresetLoader,
// parses them into domain.Config, and validates the resulting configuration.
type PresetServiceImpl struct {
	loader PresetLoader
}

// NewPresetService creates a new PresetServiceImpl with the given loader.
func NewPresetService(loader PresetLoader) *PresetServiceImpl {
	return &PresetServiceImpl{loader: loader}
}

// LoadPreset loads a preset by name, parses the YAML into a domain.Config,
// validates it, and returns the config. Returns a descriptive error if:
//   - the preset name is invalid or doesn't exist
//   - the YAML is malformed
//   - the resulting config fails validation (missing layers, rules referencing
//     non-existent layers, etc.)
func (s *PresetServiceImpl) LoadPreset(name string) (*domain.Config, error) {
	if name == "" {
		return nil, fmt.Errorf("preset name is required")
	}

	// Load raw YAML via the loader
	rawYAML, err := s.loader.LoadPreset(name)
	if err != nil {
		return nil, fmt.Errorf("loading preset %q: %w", name, err)
	}

	// Parse YAML into domain.Config
	var cfg domain.Config
	if err := yaml.Unmarshal(rawYAML, &cfg); err != nil {
		return nil, fmt.Errorf("parsing preset %q YAML: %w", name, err)
	}

	// Validate the resulting config (layers defined, rules reference existing layers, etc.)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("preset %q produced invalid config: %w", name, err)
	}

	return &cfg, nil
}

// ListPresets returns the names of all available presets.
func (s *PresetServiceImpl) ListPresets() []string {
	return s.loader.ListPresets()
}
