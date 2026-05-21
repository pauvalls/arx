// Package bootstrap initializes shared infrastructure components.
// It registers detectors into the detector package without creating
// infrastructure→infrastructure circular dependencies (C-01).
//
// The init() function in this package runs automatically when imported,
// which happens from the composition root (cmd/arx/).
package bootstrap

import (
	crosslanguage "github.com/pauvalls/arx/internal/infrastructure/detector/crosslanguage"
	csharpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/csharp"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/detector/plugin"
	godetector "github.com/pauvalls/arx/internal/infrastructure/detector/go"
	javadetector "github.com/pauvalls/arx/internal/infrastructure/detector/java"
	kotlindetector "github.com/pauvalls/arx/internal/infrastructure/detector/kotlin"
	phpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/php"
	pydetector "github.com/pauvalls/arx/internal/infrastructure/detector/python"
	rubydetector "github.com/pauvalls/arx/internal/infrastructure/detector/ruby"
	rustdetector "github.com/pauvalls/arx/internal/infrastructure/detector/rust"
	swiftdetector "github.com/pauvalls/arx/internal/infrastructure/detector/swift"
	tsdetector "github.com/pauvalls/arx/internal/infrastructure/detector/typescript"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func init() {
	detector.Register(
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
	)
}

// BuildDetectors builds a complete detector list including registered language
// detectors, an optional cross-language detector, and optional plugin detectors.
// This replaces the old detector.GetDetectorsForConfig() pattern that created
// infrastructure→infrastructure C-01 violations.
func BuildDetectors(cfg *domain.Config, pluginFactory ports.PluginDetectorFactory) []ports.Detector {
	detectors := detector.GetDetectors()
	if cfg == nil {
		return detectors
	}

	// Add cross-language detector if configured
	if cfg.CrossLanguage != nil {
		detectors = append(detectors, crosslanguage.New(cfg.CrossLanguage))
	}

	// Add plugin detectors via factory
	if pluginFactory != nil && len(cfg.Plugins) > 0 {
		pluginDetectors := detector.GetPlugins(cfg, pluginFactory)
		detectors = append(detectors, pluginDetectors...)
	}

	return detectors
}

// BuildDetectorsWithPlugins is a convenience wrapper that passes plugin.NewPluginDetector
// as the factory. Call when no custom factory is needed.
func BuildDetectorsWithPlugins(cfg *domain.Config) []ports.Detector {
	return BuildDetectors(cfg, plugin.NewPluginDetector)
}
