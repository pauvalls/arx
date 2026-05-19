package presets

import (
	"embed"
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var presetFS embed.FS

// PresetTemplate represents a configuration template loaded from embedded FS
type PresetTemplate struct {
	Name        string
	Description string
	Config      *domain.Config
}

// AvailablePresets returns a list of available preset names
func AvailablePresets() []string {
	return []string{"clean", "hexagonal", "ddd", "layered", "onion"}
}

// LoadPreset loads a preset template by name (clean, hexagonal, ddd)
// Returns PresetTemplate with parsed Config or error if preset not found/invalid
func LoadPreset(name string) (*PresetTemplate, error) {
	// Validate preset name
	validPresets := AvailablePresets()
	isValid := false
	for _, p := range validPresets {
		if p == name {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("unknown preset %q, available presets: %s", name, strings.Join(validPresets, ", "))
	}

	// Read embedded file
	filename := fmt.Sprintf("%s.yaml", name)
	content, err := presetFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read preset %q: %w", name, err)
	}

	// Parse YAML
	var config domain.Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse preset %q YAML: %w", name, err)
	}

	// Validate the parsed config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("preset %q has invalid config: %w", name, err)
	}

	return &PresetTemplate{
		Name:        name,
		Description: getPresetDescription(name),
		Config:      &config,
	}, nil
}

// ApplyPreset creates a config from preset with project-specific customization
// Currently applies project root path adjustments if needed
func ApplyPreset(template *PresetTemplate, projectRoot string) (*domain.Config, error) {
	if template == nil {
		return nil, fmt.Errorf("template is nil")
	}
	if template.Config == nil {
		return nil, fmt.Errorf("template config is nil")
	}

	// Deep copy the config to avoid modifying the template
	configCopy, err := copyConfig(template.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to copy template config: %w", err)
	}

	// Apply project-specific customizations here if needed
	// For now, presets are used as-is since paths are relative patterns
	_ = projectRoot // Mark as used for future customizations

	return configCopy, nil
}

// copyConfig creates a deep copy of a Config struct
func copyConfig(src *domain.Config) (*domain.Config, error) {
	if src == nil {
		return nil, nil
	}

	// Marshal to YAML and unmarshal to create deep copy
	data, err := yaml.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var copy domain.Config
	if err := yaml.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config copy: %w", err)
	}

	return &copy, nil
}

// getPresetDescription returns a human-readable description for each preset
func getPresetDescription(name string) string {
	switch name {
	case "clean":
		return "Clean Architecture: domain, application, infrastructure, presentation layers"
	case "hexagonal":
		return "Hexagonal/Ports-Adapters: domain, ports, adapters, infrastructure layers"
	case "ddd":
		return "Domain-Driven Design: domain, application, infrastructure, interfaces layers"
	case "layered":
		return "Layered Architecture (N-tier): presentation, business, persistence, infrastructure layers"
	case "onion":
		return "Onion Architecture (ports and adapters): domain, application, ports, infrastructure layers"
	default:
		return "Unknown preset"
	}
}
