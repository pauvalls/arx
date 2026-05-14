package application

import (
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/preset"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

// PresetServiceImpl implements the PresetService interface
type PresetServiceImpl struct{}

// NewPresetService creates a new PresetService instance
func NewPresetService() ports.PresetService {
	return &PresetServiceImpl{}
}

// ListPresets returns all available preset names
func (s *PresetServiceImpl) ListPresets() []string {
	return preset.ListPresets()
}

// LoadPreset loads a preset by name and returns a validated Config
func (s *PresetServiceImpl) LoadPreset(name string) (*domain.Config, error) {
	// Load raw YAML content
	rawYAML, err := preset.LoadPreset(name)
	if err != nil {
		return nil, err
	}

	// Parse YAML to domain.Config
	var config domain.Config
	if err := yaml.Unmarshal(rawYAML, &config); err != nil {
		return nil, fmt.Errorf("failed to parse preset %q: %w", name, err)
	}

	// Validate the loaded config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("preset %q is invalid: %w", name, err)
	}

	return &config, nil
}

// Ensure PresetServiceImpl implements the interface
var _ ports.PresetService = (*PresetServiceImpl)(nil)
