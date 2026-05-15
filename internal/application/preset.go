package application

import (
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/preset"
	"gopkg.in/yaml.v3"
)

// PresetServiceImpl loads preset templates from the embedded infrastructure layer,
// parses them into domain.Config, and validates the resulting configuration.
type PresetServiceImpl struct{}

// NewPresetService creates a new PresetServiceImpl.
func NewPresetService() *PresetServiceImpl {
	return &PresetServiceImpl{}
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

	// Load raw YAML from infrastructure layer
	rawYAML, err := preset.LoadPreset(name)
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
	return preset.ListPresets()
}
