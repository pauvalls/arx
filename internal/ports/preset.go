package ports

import "github.com/pauvalls/arx/internal/domain"

// PresetService defines the interface for loading and applying configuration presets.
type PresetService interface {
	// LoadPreset loads a preset by name, parses it into a domain.Config,
	// validates the config, and returns it. Returns a descriptive error
	// if the preset doesn't exist, has invalid YAML, or produces an invalid config.
	LoadPreset(name string) (*domain.Config, error)

	// ListPresets returns the names of all available presets.
	ListPresets() []string
}
