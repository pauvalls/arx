package ports

import "github.com/pauvalls/arx/internal/domain"

// PresetService handles loading and validation of architecture presets
type PresetService interface {
	// ListPresets returns all available preset names
	ListPresets() []string
	
	// LoadPreset loads a preset by name and returns a validated Config
	LoadPreset(name string) (*domain.Config, error)
}
