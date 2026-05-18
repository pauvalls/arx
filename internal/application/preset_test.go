package application

import (
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/infrastructure/preset"
)

// presetLoaderAdapter wraps the real preset package for dependency injection.
type presetLoaderAdapter struct{}

func (presetLoaderAdapter) LoadPreset(name string) ([]byte, error) { return preset.LoadPreset(name) }
func (presetLoaderAdapter) ListPresets() []string                  { return preset.ListPresets() }

func TestPresetService_LoadValidPreset(t *testing.T) {
	svc := NewPresetService(presetLoaderAdapter{})

	validPresets := svc.ListPresets()
	if len(validPresets) != 3 {
		t.Fatalf("expected 3 presets, got %d", len(validPresets))
	}

	for _, name := range validPresets {
		t.Run(name, func(t *testing.T) {
			cfg, err := svc.LoadPreset(name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected config, got nil")
			}
			if cfg.Version == "" {
				t.Error("expected version to be set")
			}
			if len(cfg.Layers) == 0 {
				t.Error("expected at least one layer")
			}
			if len(cfg.Rules) == 0 {
				t.Error("expected at least one rule")
			}

			// Verify all layers have paths defined
			for _, layer := range cfg.Layers {
				if layer.Name == "" {
					t.Error("layer name is required")
				}
				if len(layer.Paths) == 0 {
					t.Errorf("layer %q must have at least one path", layer.Name)
				}
			}

			// Verify all rules reference existing layers
			layerNames := make(map[string]bool)
			for _, l := range cfg.Layers {
				layerNames[l.Name] = true
			}
			for _, rule := range cfg.Rules {
				if !layerNames[rule.From] {
					t.Errorf("rule %q references unknown layer %q", rule.ID, rule.From)
				}
				for _, to := range rule.To {
					if !layerNames[to] {
						t.Errorf("rule %q references unknown layer %q in 'to'", rule.ID, to)
					}
				}
			}
		})
	}
}

func TestPresetService_InvalidPresetName(t *testing.T) {
	svc := NewPresetService(presetLoaderAdapter{})

	tests := []struct {
		name       string
		input      string
		wantErrSub string
	}{
		{
			name:       "empty name",
			input:      "",
			wantErrSub: "preset name is required",
		},
		{
			name:       "non-existent preset",
			input:      "nonexistent",
			wantErrSub: "nonexistent",
		},
		{
			name:       "path traversal",
			input:      "../../etc/passwd",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "spaces in name",
			input:      "my preset",
			wantErrSub: "invalid preset name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.LoadPreset(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

func TestPresetService_ListPresets(t *testing.T) {
	svc := NewPresetService(presetLoaderAdapter{})

	presets := svc.ListPresets()
	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}

	expected := map[string]bool{
		"clean":     true,
		"hexagonal": true,
		"ddd":       true,
	}

	for _, p := range presets {
		if !expected[p] {
			t.Errorf("unexpected preset: %q", p)
		}
	}
}

func TestPresetService_InterfaceCompliance(t *testing.T) {
	// Verify PresetServiceImpl can be created with the new constructor
	svc := NewPresetService(presetLoaderAdapter{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}
