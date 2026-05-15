package preset

import (
	"strings"
	"testing"
)

func TestListPresets_ReturnsValidNames(t *testing.T) {
	names := ListPresets()

	if len(names) == 0 {
		t.Fatal("expected at least one preset, got none")
	}

	// All returned names should be non-empty and not contain .yaml extension
	for _, name := range names {
		if name == "" {
			t.Error("preset name should not be empty")
		}
		if strings.Contains(name, ".yaml") {
			t.Errorf("preset name %q should not contain .yaml extension", name)
		}
		if !validName.MatchString(name) {
			t.Errorf("preset name %q contains invalid characters", name)
		}
	}
}

func TestLoadPreset_ValidName(t *testing.T) {
	names := ListPresets()
	if len(names) == 0 {
		t.Skip("no presets available to test")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			data, err := LoadPreset(name)
			if err != nil {
				t.Fatalf("unexpected error loading preset %q: %v", name, err)
			}

			if len(data) == 0 {
				t.Fatal("expected non-empty preset content")
			}

			// Verify content looks like YAML (contains at least one colon for key-value)
			if !strings.Contains(string(data), ":") {
				t.Error("preset content does not appear to be valid YAML")
			}
		})
	}
}

func TestLoadPreset_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrSub  string
	}{
		{
			name:       "path traversal with slashes",
			input:      "../../etc/passwd",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "spaces in name",
			input:      "my preset",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "special characters",
			input:      "preset; rm -rf /",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "null byte",
			input:      "preset\x00name",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "empty name",
			input:      "",
			wantErrSub: "invalid preset name",
		},
		{
			name:       "dots in name",
			input:      "preset.yaml",
			wantErrSub: "invalid preset name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadPreset(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

func TestLoadPreset_NonExistent(t *testing.T) {
	// Valid name format but file doesn't exist
	_, err := LoadPreset("nonexistent-preset")
	if err == nil {
		t.Fatal("expected error for non-existent preset, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-preset") {
		t.Errorf("error should mention preset name, got: %v", err)
	}
}
