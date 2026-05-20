package detector

import (
	"log"

	crosslanguage "github.com/pauvalls/arx/internal/infrastructure/detector/crosslanguage"
	csharpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/csharp"
	godetector "github.com/pauvalls/arx/internal/infrastructure/detector/go"
	javadetector "github.com/pauvalls/arx/internal/infrastructure/detector/java"
	kotlindetector "github.com/pauvalls/arx/internal/infrastructure/detector/kotlin"
	phpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/php"
	"github.com/pauvalls/arx/internal/infrastructure/detector/plugin"
	pydetector "github.com/pauvalls/arx/internal/infrastructure/detector/python"
	rubydetector "github.com/pauvalls/arx/internal/infrastructure/detector/ruby"
	rustdetector "github.com/pauvalls/arx/internal/infrastructure/detector/rust"
	swiftdetector "github.com/pauvalls/arx/internal/infrastructure/detector/swift"
	tsdetector "github.com/pauvalls/arx/internal/infrastructure/detector/typescript"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// GetDetectors returns all available language detectors.
func GetDetectors() []ports.Detector {
	return []ports.Detector{
		godetector.New(),
		tsdetector.New(),
		pydetector.New(),
		javadetector.New(),
		kotlindetector.New(),
		rustdetector.New(),
		csharpdetector.New(),
		rubydetector.New(),
		swiftdetector.New(),
		phpdetector.New(),
	}
}

// GetPlugins creates detector wrappers for each plugin defined in the config.
// It queries each plugin's capabilities, logs warnings for mismatches, and
// skips plugins with duplicate names.
func GetPlugins(cfg *domain.Config) []ports.Detector {
	if cfg == nil || len(cfg.Plugins) == 0 {
		return nil
	}

	var result []ports.Detector
	seen := make(map[string]bool)

	for _, pc := range cfg.Plugins {
		if seen[pc.Name] {
			log.Printf("Warning: skipping plugin %q (duplicate name)", pc.Name)
			continue
		}
		seen[pc.Name] = true

		// Query capabilities for validation
		caps, err := plugin.GetCapabilities(pc)
		if err != nil {
			log.Printf("Warning: plugin %q capabilities query failed: %v (will still register)", pc.Name, err)
		} else if caps.Name != pc.Name {
			log.Printf("Warning: plugin %q reports name %q in capabilities", pc.Name, caps.Name)
		}

		result = append(result, plugin.NewPluginDetector(pc))
	}

	return result
}

// GetDetectorsForConfig returns all available detectors plus the cross-language
// detector and any configured plugin detectors.
func GetDetectorsForConfig(cfg *domain.Config) []ports.Detector {
	detectors := GetDetectors()
	if cfg != nil {
		if cfg.CrossLanguage != nil {
			detectors = append(detectors, crosslanguage.New(cfg.CrossLanguage))
		}
		// Append plugin detectors
		pluginDetectors := GetPlugins(cfg)
		detectors = append(detectors, pluginDetectors...)
	}
	return detectors
}
