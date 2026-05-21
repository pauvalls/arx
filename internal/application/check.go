package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// WasmEvaluatorFunc is a function that returns a WasmEvaluator for a given WASM path.
type WasmEvaluatorFunc func(wasmPath string) domain.WasmEvaluator

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

// DetectorStatus holds the result of a single detector's execution.
type DetectorStatus struct {
	Name       string
	Applicable bool
	DepCount   int
	Error      string // empty if no error
}

// DetectorResult aggregates dependencies and per-detector statuses.
type DetectorResult struct {
	Dependencies []domain.Dependency
	Statuses     []DetectorStatus
}

// RunDetectorsWithStatus executes all detectors concurrently and returns per-detector results.
// Unlike RunDetectors, this returns status for every detector regardless of applicability.
// Each detector runs in its own goroutine with a derived context. A single detector failure
// does NOT cancel other detectors — all errors are collected and returned together.
// If maxWorkers > 0, at most maxWorkers detectors run concurrently (worker pool via semaphore).
// If maxWorkers <= 0, all detectors start concurrently (unlimited, existing behavior).
func RunDetectorsWithStatus(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector, maxWorkers ...int) (*DetectorResult, error) {
	maxWorkersVal := 0
	if len(maxWorkers) > 0 {
		maxWorkersVal = maxWorkers[0]
	}
	if len(detectors) == 0 {
		return nil, fmt.Errorf("no detectors provided")
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allDependencies []domain.Dependency
	statuses := make([]DetectorStatus, len(detectors))
	errs := make([]error, len(detectors))

	// Semaphore for worker pool when maxWorkers > 0
	var sem chan struct{}
	if maxWorkersVal > 0 {
		sem = make(chan struct{}, maxWorkersVal)
	}

	for i, detector := range detectors {
		if detector == nil {
			continue
		}

		idx := i
		d := detector
		wg.Add(1)

		if sem != nil {
			sem <- struct{}{}
		}

		go func() {
			defer wg.Done()
			if sem != nil {
				defer func() { <-sem }()
			}
			// Each detector gets its own cancellable context so one failure
			// doesn't cancel others.
			dCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			status := DetectorStatus{Name: d.Name()}

			// Check if this detector is applicable
			applicable, detectErr := d.Detect(dCtx, projectRoot)
			if detectErr != nil {
				status.Error = detectErr.Error()
				mu.Lock()
				statuses[idx] = status
				errs[idx] = fmt.Errorf("detector %q detection failed: %w", d.Name(), detectErr)
				mu.Unlock()
				return
			}

			status.Applicable = applicable

			if !applicable {
				mu.Lock()
				statuses[idx] = status
				mu.Unlock()
				return
			}

			// Extract dependencies
			deps, extractErr := d.ExtractImports(dCtx, projectRoot, layers)
			if extractErr != nil {
				status.Error = extractErr.Error()
				mu.Lock()
				statuses[idx] = status
				errs[idx] = fmt.Errorf("detector %q extraction failed: %w", d.Name(), extractErr)
				mu.Unlock()
				return
			}

			status.DepCount = len(deps)

			mu.Lock()
			allDependencies = append(allDependencies, deps...)
			statuses[idx] = status
			mu.Unlock()
		}()
	}

	wg.Wait()

	combinedErr := errors.Join(errs...)
	if combinedErr != nil {
		return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, combinedErr
	}

	return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, nil
}

// RunDetectors executes all applicable detectors concurrently and aggregates their dependencies.
// A detector is considered applicable if its Detect() method returns true for the project.
// Detectors run in parallel; an error in one detector cancels the context for others.
// maxWorkers controls concurrent goroutines (0 = unlimited, existing behavior).
func RunDetectors(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector, maxWorkers ...int) ([]domain.Dependency, error) {
	maxWorkersVal := 0
	if len(maxWorkers) > 0 {
		maxWorkersVal = maxWorkers[0]
	}
	result, err := RunDetectorsWithStatus(ctx, projectRoot, layers, detectors, maxWorkersVal)
	if err != nil {
		if result != nil {
			return result.Dependencies, err
		}
		return nil, err
	}
	return result.Dependencies, nil
}

// RunDetectorsWithProfile executes all detectors concurrently with profiling.
// It returns the aggregated dependencies, a performance report with per-detector timing,
// and any errors collected from detectors. A single detector failure does NOT cancel others.
// If maxWorkers > 0, at most maxWorkers detectors run concurrently (worker pool via semaphore).
func RunDetectorsWithProfile(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector, maxWorkers ...int) ([]domain.Dependency, *domain.PerformanceReport, error) {
	maxWorkersVal := 0
	if len(maxWorkers) > 0 {
		maxWorkersVal = maxWorkers[0]
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allDeps []domain.Dependency
	var allErrs error
	start := time.Now()
	collector := domain.NewPerfCollector()

	// Semaphore for worker pool when maxWorkers > 0
	var sem chan struct{}
	if maxWorkersVal > 0 {
		sem = make(chan struct{}, maxWorkersVal)
	}

	for _, detector := range detectors {
		if detector == nil {
			continue
		}

		d := detector
		wg.Add(1)

		if sem != nil {
			sem <- struct{}{}
		}

		go func() {
			defer wg.Done()
			if sem != nil {
				defer func() { <-sem }()
			}

			dCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			// Time the entire detector operation (Detect + ExtractImports)
			timer := domain.NewPerfTimer()

			// Check if this detector is applicable
			applicable, detectErr := d.Detect(dCtx, projectRoot)
			if detectErr != nil {
				elapsed := timer.Elapsed()
				collector.AddPhase(d.Name(), elapsed)
				mu.Lock()
				allErrs = errors.Join(allErrs, fmt.Errorf("detector %q detection failed: %w", d.Name(), detectErr))
				mu.Unlock()
				return
			}

			if !applicable {
				elapsed := timer.Elapsed()
				collector.AddPhase(d.Name(), elapsed)
				return
			}

			// Extract dependencies
			deps, extractErr := d.ExtractImports(dCtx, projectRoot, layers)
			if extractErr != nil {
				elapsed := timer.Elapsed()
				collector.AddPhase(d.Name(), elapsed)
				mu.Lock()
				allErrs = errors.Join(allErrs, fmt.Errorf("detector %q extraction failed: %w", d.Name(), extractErr))
				mu.Unlock()
				return
			}

			elapsed := timer.Elapsed()
			collector.AddPhase(d.Name(), elapsed)

			mu.Lock()
			allDeps = append(allDeps, deps...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	report := collector.Report()
	report.Total = time.Since(start)

	return allDeps, &report, allErrs
}

// EvaluateArchitecture checks dependencies against architectural rules and returns violations.
// It enriches violations with explanations from the built-in explanations library.
// userFuncs is an optional compiled user-function map (may be nil).
func EvaluateArchitecture(dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer, userFuncs ...map[string]domain.Expr) []domain.Violation {
	violations := domain.EvaluateRules(dependencies, rules, layers, userFuncs...)

	// Enrich violations with explanations from the built-in library
	for i := range violations {
		violations[i].Message = enrichViolationMessage(violations[i], rules)
	}

	return violations
}

// EvaluateArchitectureWithWasm is like EvaluateArchitecture but also evaluates WASM policies.
// The wasmEval function returns an evaluator for the given WASM file path (may be nil).
func EvaluateArchitectureWithWasm(ctx context.Context, dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer, wasmEval WasmEvaluatorFunc, userFuncs ...map[string]domain.Expr) []domain.Violation {
	violations := domain.EvaluateRules(dependencies, rules, layers, userFuncs...)

	// Evaluate WASM policies
	wasmViolations, _ := domain.EvaluateWasmRules(ctx, rules, dependencies, layers, violations, wasmEval)
	violations = append(violations, wasmViolations...)

	// Enrich all violations with explanations
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
