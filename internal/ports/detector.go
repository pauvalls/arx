package ports

import (
	"context"

	"github.com/pauvalls/arx/internal/domain"
)

// PluginDetectorFactory is a function type that creates a new plugin-based Detector.
// Breaking the direct import dependency from detector/registry to detector/plugin
// resolves an architecture circular dependency (C-01).
type PluginDetectorFactory func(cfg domain.PluginConfig) Detector

// Detector defines the interface for language-specific dependency detectors
type Detector interface {
	// Name returns the detector name (e.g., "go", "typescript", "python")
	Name() string

	// Detect checks if this detector can handle the given project
	// Returns true if the project uses this language
	Detect(ctx context.Context, projectRoot string) (bool, error)

	// ExtractImports extracts all dependencies from the project
	// Returns a list of dependencies with resolved layers
	ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error)
}
