package preset

import (
	"strings"
	"testing"
)

func TestListPresets_ReturnsValidNames(t *testing.T) {
	presets := ListPresets()

	if len(presets) == 0 {
		t.Fatal("expected at least one preset, got none")
	}

	// Check that all returned names are valid
	for _, name := range presets {
		if !validPresetName(name) {
			t.Errorf("preset name %q is invalid", name)
		}
		if strings.Contains(name, ".") {
			t.Errorf("preset name %q should not contain extension", name)
		}
	}
}

func TestLoadPreset_ValidName(t *testing.T) {
	presets := ListPresets()
	if len(presets) == 0 {
		t.Fatal("no presets available for testing")
	}

	// Test loading the first available preset
	content, err := LoadPreset(presets[0])
	if err != nil {
		t.Fatalf("failed to load preset %q: %v", presets[0], err)
	}

	if len(content) == 0 {
		t.Error("loaded preset content is empty")
	}

	// Content should start with YAML header comment
	if !strings.Contains(string(content), "#") {
		t.Error("preset should contain YAML comments")
	}
}

func TestLoadPreset_InvalidName(t *testing.T) {
	invalidNames := []string{
		"",
		"../etc/passwd",
		"preset with spaces",
		"preset_with_underscores",
		"preset.with.dots",
		"preset$special",
	}

	for _, name := range invalidNames {
		_, err := LoadPreset(name)
		if err == nil {
			t.Errorf("expected error for invalid name %q, got nil", name)
		}
	}
}

func TestLoadPreset_NonExistent(t *testing.T) {
	_, err := LoadPreset("nonexistent-preset")
	if err == nil {
		t.Fatal("expected error for non-existent preset, got nil")
	}

	// Error message should mention available presets
	errMsg := err.Error()
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("error message should mention 'not found': %s", errMsg)
	}
	if !strings.Contains(errMsg, "Available presets:") {
		t.Errorf("error message should list available presets: %s", errMsg)
	}
}
