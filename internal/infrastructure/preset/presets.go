package preset

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed *.yaml
var presetFS embed.FS

// validPresetName checks if the preset name contains only alphanumeric characters and hyphens
func validPresetName(name string) bool {
	if name == "" {
		return false
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", name)
	return matched
}

// ListPresets returns all available preset names (without .yaml extension)
func ListPresets() []string {
	entries, err := presetFS.ReadDir(".")
	if err != nil {
		return []string{}
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			names = append(names, name)
		}
	}
	return names
}

// LoadPreset loads a preset template by name and returns its raw YAML content
func LoadPreset(name string) ([]byte, error) {
	if !validPresetName(name) {
		return nil, fmt.Errorf("invalid preset name %q: must contain only alphanumeric characters and hyphens", name)
	}

	filename := name + ".yaml"
	content, err := presetFS.ReadFile(filename)
	if err != nil {
		// Check if it's a file not found error
		available := ListPresets()
		if len(available) == 0 {
			return nil, fmt.Errorf("preset %q not found: no presets available", name)
		}
		return nil, fmt.Errorf("preset %q not found. Available presets: %s", name, strings.Join(available, ", "))
	}

	return content, nil
}
