package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/ports"
)

func TestPresetService_LoadValidPreset(t *testing.T) {
	service := NewPresetService()

	// Get available presets
	presets := service.ListPresets()
	if len(presets) == 0 {
		t.Fatal("no presets available for testing")
	}

	// Test loading each preset
	for _, presetName := range presets {
		t.Run(presetName, func(t *testing.T) {
			config, err := service.LoadPreset(presetName)
			if err != nil {
				t.Fatalf("failed to load preset %q: %v", presetName, err)
			}

			if config == nil {
				t.Fatal("loaded config is nil")
			}

			if config.Version == "" {
				t.Error("config version is empty")
			}

			if len(config.Layers) == 0 {
				t.Error("config has no layers defined")
			}

			if len(config.Rules) == 0 {
				t.Error("config has no rules defined")
			}
		})
	}
}

func TestPresetService_InvalidPresetName(t *testing.T) {
	service := NewPresetService()

	invalidNames := []string{
		"",
		"../etc/passwd",
		"preset with spaces",
		"nonexistent-preset",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			_, err := service.LoadPreset(name)
			if err == nil {
				t.Errorf("expected error for invalid name %q, got nil", name)
			}
		})
	}
}

func TestPresetService_ListPresets(t *testing.T) {
	service := NewPresetService()

	presets := service.ListPresets()
	if len(presets) == 0 {
		t.Fatal("expected at least one preset, got none")
	}

	// Should include the three standard presets
	expectedPresets := map[string]bool{
		"clean":       false,
		"hexagonal":   false,
		"ddd":         false,
	}

	for _, preset := range presets {
		if _, ok := expectedPresets[preset]; ok {
			expectedPresets[preset] = true
		}
	}

	for preset, found := range expectedPresets {
		if !found {
			t.Errorf("expected preset %q not found", preset)
		}
	}
}

func TestPresetService_InterfaceCompliance(t *testing.T) {
	var _ ports.PresetService = (*PresetServiceImpl)(nil)
}
