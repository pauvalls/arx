package detector

import (
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// registeredDetectors holds all registered language detectors.
// Registration is done through a separate init package to avoid
// infrastructure→infrastructure circular dependencies (C-01).
var registeredDetectors []ports.Detector

// Register adds detectors to the global registry.
func Register(d ...ports.Detector) {
	registeredDetectors = append(registeredDetectors, d...)
}

// GetDetectors returns all registered language detectors.
func GetDetectors() []ports.Detector {
	result := make([]ports.Detector, len(registeredDetectors))
	copy(result, registeredDetectors)
	return result
}

// ResetDetectors clears the registry (for testing only).
func ResetDetectors() {
	registeredDetectors = nil
}

// GetPlugins creates detector wrappers for each plugin defined in the config.
// Accepts a factory function to avoid importing the plugin sub-package directly.
func GetPlugins(cfg *domain.Config, factory ports.PluginDetectorFactory) []ports.Detector {
	if cfg == nil || len(cfg.Plugins) == 0 || factory == nil {
		return nil
	}

	var result []ports.Detector
	seen := make(map[string]bool)

	for _, pc := range cfg.Plugins {
		if seen[pc.Name] {
			continue
		}
		seen[pc.Name] = true
		result = append(result, factory(pc))
	}

	return result
}

// GetDetectorsForConfig returns registered detectors plus plugin detectors.
// Additional detectors (like cross-language) can be appended by the caller.
func GetDetectorsForConfig(cfg *domain.Config, pluginFactory ...ports.PluginDetectorFactory) []ports.Detector {
	detectors := GetDetectors()
	if cfg == nil {
		return detectors
	}

	// Append plugin detectors using the provided factory
	if len(pluginFactory) > 0 && pluginFactory[0] != nil {
		pluginDetectors := GetPlugins(cfg, pluginFactory[0])
		detectors = append(detectors, pluginDetectors...)
	}
	return detectors
}
