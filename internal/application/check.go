package application

import (
	"context"
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// LoadConfig reads and validates a configuration file using the provided ConfigReader.
func LoadConfig(configPath string, reader ports.ConfigReader) (*domain.Config, error) {
	if reader == nil {
		return nil, fmt.Errorf("config reader is nil")
	}

	config, err := reader.Read(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from %s: %w", configPath, err)
	}

	if err := reader.Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// RunDetectors executes all applicable detectors and aggregates their dependencies.
// A detector is considered applicable if its Detect() method returns true for the project.
func RunDetectors(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector) ([]domain.Dependency, error) {
	if len(detectors) == 0 {
		return nil, fmt.Errorf("no detectors provided")
	}

	var allDependencies []domain.Dependency

	for _, detector := range detectors {
		if detector == nil {
			continue
		}

		// Check if this detector is applicable
		applicable, err := detector.Detect(ctx, projectRoot)
		if err != nil {
			return nil, fmt.Errorf("detector %q detection failed: %w", detector.Name(), err)
		}

		if !applicable {
			continue
		}

		// Extract dependencies
		deps, err := detector.ExtractImports(ctx, projectRoot, layers)
		if err != nil {
			return nil, fmt.Errorf("detector %q extraction failed: %w", detector.Name(), err)
		}

		allDependencies = append(allDependencies, deps...)
	}

	return allDependencies, nil
}

// EvaluateArchitecture checks dependencies against architectural rules and returns violations.
// It enriches violations with explanations from the built-in explanations library.
func EvaluateArchitecture(dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer) []domain.Violation {
	violations := domain.EvaluateRules(dependencies, rules, layers)

	// Enrich violations with explanations from the built-in library
	for i := range violations {
		violations[i].Message = enrichViolationMessage(violations[i], rules)
	}

	return violations
}

// enrichViolationMessage looks up the rule's explanation and enhances the violation message.
func enrichViolationMessage(violation domain.Violation, rules []domain.Rule) string {
	// Find the matching rule
	for _, rule := range rules {
		if rule.ID == violation.RuleID {
			if rule.Explanation != "" {
				return rule.Explanation
			}
			// Fall back to built-in explanations
			return GetExplanation(rule.ID)
		}
	}

	return violation.Message
}

// GenerateReport outputs the violations using the provided Reporter.
func GenerateReport(violations []domain.Violation, format ports.OutputFormat, reporter ports.Reporter) error {
	if reporter == nil {
		return fmt.Errorf("reporter is nil")
	}

	if err := reporter.Report(violations, format); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}
