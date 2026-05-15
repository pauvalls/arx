package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// DiagramService generates dependency diagrams from project analysis
type DiagramService struct {
	detectors []ports.Detector
}

// NewDiagramService creates a new DiagramService
func NewDiagramService(detectors []ports.Detector) *DiagramService {
	return &DiagramService{detectors: detectors}
}

// DiagramResult holds the output of diagram generation
type DiagramResult struct {
	Dependencies []domain.Dependency
	Layers       []domain.Layer
	Violations   []domain.Violation
}

// Generate analyzes a project and returns diagram data
func (s *DiagramService) Generate(projectRoot string, layers []domain.Layer, config *domain.Config) (*DiagramResult, error) {
	// Run detectors to extract dependencies
	deps, err := s.extractDependencies(projectRoot, layers)
	if err != nil {
		return nil, fmt.Errorf("failed to extract dependencies: %w", err)
	}

	// Evaluate rules to find violations
	violations := s.evaluateRules(deps, config.Rules, layers)

	return &DiagramResult{
		Dependencies: deps,
		Layers:       layers,
		Violations:   violations,
	}, nil
}

// extractDependencies runs all detectors and aggregates results
func (s *DiagramService) extractDependencies(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	ctx := context.Background()
	var allDeps []domain.Dependency

	for _, detector := range s.detectors {
		if detector == nil {
			continue
		}

		applicable, err := detector.Detect(ctx, projectRoot)
		if err != nil {
			continue // Skip detector on error
		}
		if !applicable {
			continue
		}

		deps, err := detector.ExtractImports(ctx, projectRoot, layers)
		if err != nil {
			return nil, fmt.Errorf("detector %s failed: %w", detector.Name(), err)
		}

		allDeps = append(allDeps, deps...)
	}

	// Sort dependencies for consistent output
	sortDependencies(allDeps)

	return allDeps, nil
}

// evaluateRules checks dependencies against rules and returns violations
func (s *DiagramService) evaluateRules(deps []domain.Dependency, rules []domain.Rule, layers []domain.Layer) []domain.Violation {
	return EvaluateArchitecture(deps, rules, layers)
}

// sortDependencies sorts dependencies for consistent diagram output
func sortDependencies(deps []domain.Dependency) {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].SourceFile != deps[j].SourceFile {
			return deps[i].SourceFile < deps[j].SourceFile
		}
		return deps[i].ImportPath < deps[j].ImportPath
	})
}
