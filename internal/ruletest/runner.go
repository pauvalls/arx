package ruletest

import (
	"fmt"
	"path/filepath"

	"github.com/pauvalls/arx/internal/domain"
)

// ConfigReader reads arx configuration files
type ConfigReader interface {
	Read(configPath string) (*domain.Config, error)
}

// DependenciesFunc detects dependencies for a given project root and layers
type DependenciesFunc func(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error)

// RuleTestRunner evaluates test suites against real fixtures using
// the EvaluateRules pipeline and EvalSuite
type RuleTestRunner struct {
	configReader ConfigReader
	detectDeps   DependenciesFunc
}

// NewRuleTestRunner creates a new RuleTestRunner with the given dependencies
func NewRuleTestRunner(reader ConfigReader, detect DependenciesFunc) *RuleTestRunner {
	if detect == nil {
		detect = func(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
			return nil, nil
		}
	}
	return &RuleTestRunner{
		configReader: reader,
		detectDeps:   detect,
	}
}

// Run evaluates a TestSuite against a fixture directory.
// It reads the arx.yaml from the fixture, detects dependencies,
// runs EvaluateRules, and compares results against expectations.
func (r *RuleTestRunner) Run(suite TestSuite, fixturePath string) (result EvalResult, err error) {
	// Panic recovery — don't crash on malformed fixtures
	defer func() {
		if rec := recover(); rec != nil {
			result = EvalResult{
				Passed: false,
				Cases: []CaseResult{
					{
						Name:    "(runner)",
						Passed:  false,
						Details: fmt.Sprintf("panic recovered: %v", rec),
					},
				},
			}
			err = nil // panic is caught, return result as error state
		}
	}()

	if r.configReader == nil {
		return EvalResult{}, fmt.Errorf("config reader is nil")
	}

	// Read the fixture's arx.yaml
	configPath := filepath.Join(fixturePath, "arx.yaml")
	cfg, err := r.configReader.Read(configPath)
	if err != nil {
		return EvalResult{}, fmt.Errorf("reading config from %s: %w", configPath, err)
	}

	// Detect dependencies for the fixture
	dependencies, err := r.detectDeps(fixturePath, cfg.Layers)
	if err != nil {
		return EvalResult{}, fmt.Errorf("detecting dependencies: %w", err)
	}

	// Evaluate rules against dependencies
	violations := domain.EvaluateRules(dependencies, cfg.Rules, cfg.Layers)

	// Check expectations for each test case
	result = EvalSuite(suite, violations)

	return result, nil
}
