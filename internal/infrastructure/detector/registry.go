package detector

import (
	crosslanguage "github.com/pauvalls/arx/internal/infrastructure/detector/crosslanguage"
	csharpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/csharp"
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

// GetDetectors returns all available language detectors.
// In the future, this could support dynamic plugin loading.
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

// GetDetectorsForConfig returns all available detectors plus the cross-language
// detector if the config has cross-language mappings defined.
func GetDetectorsForConfig(cfg *domain.Config) []ports.Detector {
	detectors := GetDetectors()
	if cfg != nil && cfg.CrossLanguage != nil {
		detectors = append(detectors, crosslanguage.New(cfg.CrossLanguage))
	}
	return detectors
}
