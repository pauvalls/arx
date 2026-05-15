package application

import (
	"fmt"
	"testing"
)

func TestAvailablePresets(t *testing.T) {
	t.Run("returns three presets", func(t *testing.T) {
		presets := AvailablePresets()

		if len(presets) != 3 {
			t.Fatalf("expected 3 presets, got %d", len(presets))
		}

		expected := map[string]bool{
			"clean":       true,
			"hexagonal":   true,
			"ddd":         true,
		}

		for _, p := range presets {
			if !expected[p] {
				t.Errorf("unexpected preset: %q", p)
			}
		}
	})
}

func TestLoadPreset(t *testing.T) {
	tests := []struct {
		name      string
		preset    string
		wantError bool
		check     func(*PresetTemplate) error
	}{
		{
			name:   "clean preset loads successfully",
			preset: "clean",
			check: func(pt *PresetTemplate) error {
				if pt.Name != "clean" {
					return fmt.Errorf("expected name 'clean', got %q", pt.Name)
				}
				if len(pt.Config.Layers) == 0 {
					return fmt.Errorf("expected layers in clean preset")
				}
				if len(pt.Config.Rules) == 0 {
					return fmt.Errorf("expected rules in clean preset")
				}
				return nil
			},
		},
		{
			name:   "hexagonal preset loads successfully",
			preset: "hexagonal",
			check: func(pt *PresetTemplate) error {
				if pt.Name != "hexagonal" {
					return fmt.Errorf("expected name 'hexagonal', got %q", pt.Name)
				}
				// Check for ports layer (specific to hexagonal)
				hasPorts := false
				for _, layer := range pt.Config.Layers {
					if layer.Name == "ports" {
						hasPorts = true
						break
					}
				}
				if !hasPorts {
					return fmt.Errorf("hexagonal preset should have 'ports' layer")
				}
				return nil
			},
		},
		{
			name:   "ddd preset loads successfully",
			preset: "ddd",
			check: func(pt *PresetTemplate) error {
				if pt.Name != "ddd" {
					return fmt.Errorf("expected name 'ddd', got %q", pt.Name)
				}
				// Check for interfaces layer (specific to ddd)
				hasInterfaces := false
				for _, layer := range pt.Config.Layers {
					if layer.Name == "interfaces" {
						hasInterfaces = true
						break
					}
				}
				if !hasInterfaces {
					return fmt.Errorf("ddd preset should have 'interfaces' layer")
				}
				return nil
			},
		},
		{
			name:      "invalid preset returns error",
			preset:    "invalid",
			wantError: true,
		},
		{
			name:      "empty preset name returns error",
			preset:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := LoadPreset(tt.preset)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if template == nil {
				t.Fatal("expected template, got nil")
			}

			if tt.check != nil {
				if err := tt.check(template); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

func TestLoadPreset_ValidatesConfig(t *testing.T) {
	// All presets should produce valid configs that pass domain.Config.Validate()
	presets := AvailablePresets()

	for _, presetName := range presets {
		t.Run(presetName, func(t *testing.T) {
			template, err := LoadPreset(presetName)
			if err != nil {
				t.Fatalf("failed to load preset: %v", err)
			}

			// The LoadPreset already validates, but let's double-check
			if err := template.Config.Validate(); err != nil {
				t.Errorf("preset config validation failed: %v", err)
			}
		})
	}
}

func TestApplyPreset(t *testing.T) {
	t.Run("applies clean preset", func(t *testing.T) {
		template, err := LoadPreset("clean")
		if err != nil {
			t.Fatalf("failed to load preset: %v", err)
		}

		config, err := ApplyPreset(template, "/test/project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config == nil {
			t.Fatal("expected config, got nil")
		}

		// Verify config is a copy (not the same instance)
		if config == template.Config {
			t.Fatal("expected config copy, got same instance")
		}

		// Verify config structure
		if len(config.Layers) == 0 {
			t.Error("expected layers in applied config")
		}
		if len(config.Rules) == 0 {
			t.Error("expected rules in applied config")
		}
	})

	t.Run("nil template returns error", func(t *testing.T) {
		_, err := ApplyPreset(nil, "/test/project")
		if err == nil {
			t.Fatal("expected error for nil template, got nil")
		}
	})

	t.Run("nil config in template returns error", func(t *testing.T) {
		template := &PresetTemplate{
			Name:   "test",
			Config: nil,
		}
		_, err := ApplyPreset(template, "/test/project")
		if err == nil {
			t.Fatal("expected error for nil config, got nil")
		}
	})
}

func TestApplyPreset_PreservesStructure(t *testing.T) {
	template, err := LoadPreset("hexagonal")
	if err != nil {
		t.Fatalf("failed to load preset: %v", err)
	}

	config, err := ApplyPreset(template, "/test/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify structure is preserved
	if len(config.Layers) != len(template.Config.Layers) {
		t.Errorf("expected %d layers, got %d", len(template.Config.Layers), len(config.Layers))
	}

	if len(config.Rules) != len(template.Config.Rules) {
		t.Errorf("expected %d rules, got %d", len(template.Config.Rules), len(config.Rules))
	}

	// Verify exclude patterns
	if len(config.Exclude) == 0 {
		t.Error("expected exclude patterns in applied config")
	}

	// Verify language overrides
	if len(config.LanguageOverrides) == 0 {
		t.Error("expected language overrides in applied config")
	}
}

func TestPresetFS_EmbedWorks(t *testing.T) {
	// Verify embed.FS is properly set up
	t.Run("preset files exist in embed FS", func(t *testing.T) {
		presets := AvailablePresets()
		for _, p := range presets {
			filename := fmt.Sprintf("%s.yaml", p)
			_, err := presetFS.ReadFile(filename)
			if err != nil {
				t.Errorf("failed to read %s from embed FS: %v", filename, err)
			}
		}
	})
}
