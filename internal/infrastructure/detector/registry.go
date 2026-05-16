package detector

import (
	csharpdetector "github.com/pauvalls/arx/internal/infrastructure/detector/csharp"
	godetector "github.com/pauvalls/arx/internal/infrastructure/detector/go"
	javadetector "github.com/pauvalls/arx/internal/infrastructure/detector/java"
	kotlindetector "github.com/pauvalls/arx/internal/infrastructure/detector/kotlin"
	pydetector "github.com/pauvalls/arx/internal/infrastructure/detector/python"
	rubydetector "github.com/pauvalls/arx/internal/infrastructure/detector/ruby"
	rustdetector "github.com/pauvalls/arx/internal/infrastructure/detector/rust"
	tsdetector "github.com/pauvalls/arx/internal/infrastructure/detector/typescript"
	"github.com/pauvalls/arx/internal/ports"
)

// GetDetectors returns all available detectors
// In the future, this could support dynamic plugin loading
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
	}
}
