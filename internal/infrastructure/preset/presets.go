package preset

import (
	"embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed *.yaml
var presetFS embed.FS

// validName matches only alphanumeric characters and hyphens.
var validName = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// ListPresets returns the names of all available preset templates
// (filenames without the .yaml extension).
func ListPresets() []string {
	entries, err := presetFS.ReadDir(".")
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".yaml"))
	}
	return names
}

// LoadPreset reads a preset template by name and returns its raw YAML content.
// The name must contain only alphanumeric characters and hyphens.
func LoadPreset(name string) ([]byte, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("invalid preset name %q: only alphanumeric characters and hyphens are allowed", name)
	}

	filename := name + ".yaml"
	data, err := presetFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("loading preset %q: %w", name, err)
	}

	return data, nil
}

// PresetDir returns the base name of the directory used for presets,
// useful for display purposes.
func PresetDir() string {
	return filepath.Base("configs/presets")
}
