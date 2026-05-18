package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// AuditService orchestrates the complete architecture audit workflow.
// It loads configuration, runs detectors, evaluates rules, calculates metrics,
// and persists audit history for trend analysis.
type AuditService struct {
	configReader   ports.ConfigReader
	detectors      []ports.Detector
	historyStorage ports.HistoryStorage
	historyPath    string
}

// NewAuditService creates a new AuditService with the given dependencies.
// If historyStorage is nil, history persistence is skipped.
// If historyPath is empty, defaults to ".arx-history".
func NewAuditService(
	configReader ports.ConfigReader,
	detectors []ports.Detector,
	historyStorage ports.HistoryStorage,
	historyPath string,
) *AuditService {
	if historyPath == "" {
		historyPath = ".arx-history"
	}
	return &AuditService{
		configReader:   configReader,
		detectors:      detectors,
		historyStorage: historyStorage,
		historyPath:    historyPath,
	}
}

// Audit executes a complete architecture audit on the specified project.
// Flow: Load config → Run detectors → Evaluate rules → Calculate coupling → Calculate debt → Save history
// Returns a comprehensive AuditReport with violations, metrics, and trends.
func (s *AuditService) Audit(ctx context.Context, projectRoot, configPath string) (*domain.AuditReport, error) {
	// Step 1: Load configuration
	config, err := s.loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Step 2: Run detectors to extract dependencies
	dependencies, err := s.runDetectors(ctx, projectRoot, config.Layers)
	if err != nil {
		return nil, err
	}

	// Step 3: Evaluate rules and get violations
	violations := s.evaluateRules(dependencies, config.Rules, config.Layers, config.UserFunctions())

	// Step 4: Calculate coupling matrix
	couplingMatrix := s.calculateCoupling(dependencies, config.Layers)

	// Step 5: Calculate debt score
	debtScore := s.calculateDebt(violations, couplingMatrix)

	// Step 6: Calculate config hash for tracking changes
	configHash, err := s.hashConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to hash config: %w", err)
	}

	// Step 7: Build the audit report
	report := &domain.AuditReport{
		Timestamp:      time.Now(),
		ProjectRoot:    projectRoot,
		ConfigHash:     configHash,
		Violations:     violations,
		CouplingMatrix: couplingMatrix,
		DebtScore:      debtScore,
	}

	// Step 8: Calculate trends (handles nil historyStorage internally)
	report.TrendReport = s.calculateTrends(report)

	// Step 9: Save to history if storage is configured
	if s.historyStorage != nil {
		if err := s.saveHistory(ctx, report); err != nil {
			// Log warning but don't fail the audit
			fmt.Fprintf(os.Stderr, "Warning: failed to save audit history: %v\n", err)
		}
	}

	return report, nil
}

// loadConfig reads and validates the configuration file.
func (s *AuditService) loadConfig(configPath string) (*domain.Config, error) {
	if s.configReader == nil {
		return nil, fmt.Errorf("config reader is nil")
	}

	config, err := s.configReader.Read(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from %s: %w", configPath, err)
	}

	if err := s.configReader.Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// runDetectors executes all applicable detectors and aggregates their dependencies.
func (s *AuditService) runDetectors(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	if len(s.detectors) == 0 {
		return nil, fmt.Errorf("no detectors provided")
	}

	return RunDetectors(ctx, projectRoot, layers, s.detectors)
}

// evaluateRules checks dependencies against architectural rules.
func (s *AuditService) evaluateRules(dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer, userFuncs ...map[string]domain.Expr) []domain.Violation {
	return EvaluateArchitecture(dependencies, rules, layers, userFuncs...)
}

// calculateCoupling builds a matrix of dependencies between layers.
func (s *AuditService) calculateCoupling(dependencies []domain.Dependency, layers []domain.Layer) domain.CouplingMatrix {
	matrix := domain.NewCouplingMatrix()

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Count dependencies between layers
	for _, dep := range dependencies {
		sourceLayer := resolveLayerForCoupling(dep.SourceFile, layerMap)
		targetLayer := dep.ResolvedLayer

		// Skip if we couldn't resolve layers
		if sourceLayer == "" || targetLayer == "" {
			continue
		}

		matrix.Add(sourceLayer, targetLayer)
	}

	return matrix
}

// resolveLayerForCoupling finds the layer that matches a given file path.
func resolveLayerForCoupling(filePath string, layerMap map[string]*domain.Layer) string {
	for name, layer := range layerMap {
		if layer.MatchesPath(filePath) {
			return name
		}
	}
	return ""
}

// calculateDebt computes the technical debt score from violations.
// Formula: (error_count × 3) + (warning_count × 1) + (info_count × 0)
// Plus circular dependency penalty: +5 per circular dependency.
func (s *AuditService) calculateDebt(violations []domain.Violation, matrix domain.CouplingMatrix) domain.DebtScore {
	debt := domain.NewDebtScore()

	// Count violations by severity
	for _, v := range violations {
		debt.AddViolation(string(v.Severity))
	}

	// Add circular dependency penalty
	circularCount := countCircularDependencies(matrix)
	for i := 0; i < circularCount; i++ {
		debt.BySeverity["circular"] = debt.BySeverity["circular"] + 5
	}
	debt.Calculate()

	return debt
}

// countCircularDependencies detects bidirectional dependencies in the coupling matrix.
func countCircularDependencies(matrix domain.CouplingMatrix) int {
	circularCount := 0
	checked := make(map[string]bool)

	for fromLayer, targets := range matrix.FromTo {
		for toLayer := range targets {
			// Check if reverse dependency exists
			pair := fromLayer + "->" + toLayer
			reversePair := toLayer + "->" + fromLayer

			if checked[reversePair] {
				continue // Already counted this pair
			}

			// Check if reverse exists
			if reverseTargets, ok := matrix.FromTo[toLayer]; ok {
				if _, exists := reverseTargets[fromLayer]; exists {
					circularCount++
					checked[pair] = true
				}
			}
		}
	}

	return circularCount
}

// calculateTrends compares current audit with previous history.
func (s *AuditService) calculateTrends(current *domain.AuditReport) domain.TrendReport {
	if s.historyStorage == nil {
		return domain.TrendReport{
			Status:  domain.TrendUnchanged,
			Summary: "History storage not configured",
		}
	}

	// Load previous audit
	ctx := context.Background()
	previous, err := s.historyStorage.LoadLatest(ctx)
	if err != nil || previous == nil {
		return domain.TrendReport{
			Status:  domain.TrendUnchanged,
			Summary: "No previous audit for comparison",
		}
	}

	// Calculate trend report
	return domain.NewTrendReport(current, previous)
}

// saveHistory persists the audit report to the history storage.
func (s *AuditService) saveHistory(ctx context.Context, report *domain.AuditReport) error {
	if s.historyStorage == nil {
		return fmt.Errorf("history storage is nil")
	}

	// Save the report
	if _, err := s.historyStorage.Save(ctx, report); err != nil {
		return fmt.Errorf("failed to save audit: %w", err)
	}

	// Enforce retention policy
	if _, err := s.historyStorage.DeleteOld(ctx, 10); err != nil {
		return fmt.Errorf("failed to enforce retention policy: %w", err)
	}

	return nil
}

// hashConfig generates a SHA256 hash of the config file for change tracking.
func (s *AuditService) hashConfig(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
