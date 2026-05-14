package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// AuditService generates comprehensive architecture health reports
type AuditService struct {
	configReader   ports.ConfigReader
	detectors      []ports.Detector
	historyStorage ports.HistoryStorage
}

// NewAuditService creates a new AuditService
func NewAuditService(
	configReader ports.ConfigReader,
	detectors []ports.Detector,
	historyStorage ports.HistoryStorage,
) *AuditService {
	return &AuditService{
		configReader:   configReader,
		detectors:      detectors,
		historyStorage: historyStorage,
	}
}

// Audit runs a comprehensive architecture audit
func (s *AuditService) Audit(ctx context.Context, projectRoot, configPath string) (*domain.AuditReport, error) {
	// Load config
	cfg, err := s.configReader.Read(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Run detectors to extract dependencies
	deps, err := s.extractDependencies(ctx, projectRoot, cfg.Layers)
	if err != nil {
		return nil, fmt.Errorf("failed to extract dependencies: %w", err)
	}

	// Evaluate rules to find violations
	violations := EvaluateArchitecture(deps, cfg.Rules, cfg.Layers)

	// Calculate coupling matrix
	coupling := s.calculateCoupling(deps, cfg.Layers)

	// Calculate debt score
	debt := s.calculateDebt(violations, coupling)

	// Load previous audit for trends
	previous, _ := s.historyStorage.LoadLatest()

	// Calculate trends
	trend := domain.NewTrendReport(
		&domain.AuditReport{
			Violations: violations,
			DebtScore:  debt,
		},
		previous,
	)

	// Create report
	report := &domain.AuditReport{
		Timestamp:      time.Now(),
		ProjectRoot:    projectRoot,
		ConfigHash:     s.calculateConfigHash(cfg),
		Violations:     violations,
		CouplingMatrix: coupling,
		DebtScore:      debt,
		TrendReport:    trend,
	}

	// Save to history
	if err := s.historyStorage.Save(report); err != nil {
		return nil, fmt.Errorf("failed to save audit history: %w", err)
	}

	// Apply retention policy
	if err := s.historyStorage.DeleteOld(s.historyStorage.GetRetentionLimit()); err != nil {
		return nil, fmt.Errorf("failed to clean old audits: %w", err)
	}

	return report, nil
}

// extractDependencies runs all detectors and aggregates results
func (s *AuditService) extractDependencies(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
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

	return allDeps, nil
}

// calculateCoupling builds a coupling matrix from dependencies
func (s *AuditService) calculateCoupling(deps []domain.Dependency, layers []domain.Layer) domain.CouplingMatrix {
	matrix := domain.CouplingMatrix{}

	for _, dep := range deps {
		if dep.ResolvedLayer == "" {
			continue
		}

		sourceLayer := s.resolveLayer(dep.SourceFile, layers)
		if sourceLayer == "" {
			continue
		}

		matrix.Add(sourceLayer, dep.ResolvedLayer)
	}

	return matrix
}

// calculateDebt computes technical debt score
func (s *AuditService) calculateDebt(violations []domain.Violation, coupling domain.CouplingMatrix) domain.DebtScore {
	debt := domain.DebtScore{
		BySeverity: make(map[string]int),
	}

	// Count by severity
	for _, v := range violations {
		debt.BySeverity[string(v.Severity)]++
	}

	// Calculate total: (errors×3) + (warnings×1) + (infos×0)
	debt.Total = debt.BySeverity["error"]*3 + debt.BySeverity["warning"]*1

	// Add circular dependency penalty (×5 per circular pair)
	circularPairs := s.countCircularPairs(coupling)
	debt.Total += circularPairs * 5

	return debt
}

// countCircularPairs finds bidirectional dependencies
func (s *AuditService) countCircularPairs(coupling domain.CouplingMatrix) int {
	count := 0
	seen := make(map[string]bool)

	for fromLayer, toMap := range coupling.FromTo {
		for toLayer := range toMap {
			// Check if reverse dependency exists
			if coupling.Get(toLayer, fromLayer) > 0 {
				// Create canonical key to avoid double counting
				pair := fmt.Sprintf("%s-%s", min(fromLayer, toLayer), max(fromLayer, toLayer))
				if !seen[pair] {
					seen[pair] = true
					count++
				}
			}
		}
	}

	return count
}

// resolveLayer finds the layer for a given file path
func (s *AuditService) resolveLayer(filePath string, layers []domain.Layer) string {
	for _, layer := range layers {
		for _, pattern := range layer.Paths {
			if s.matchPattern(pattern, filePath) {
				return layer.Name
			}
		}
	}
	return ""
}

// matchPattern checks if a file path matches a glob pattern
func (s *AuditService) matchPattern(pattern, filePath string) bool {
	// Simple pattern matching for ** patterns
	pattern = trimSuffix(pattern, "/**")
	return hasPrefix(filePath, pattern) || contains(filePath, "/"+pattern+"/")
}

// calculateConfigHash generates a hash of the config for change detection
func (s *AuditService) calculateConfigHash(cfg interface{}) string {
	// Simple hash - in production would hash the actual config content
	hash := sha256.Sum256([]byte(fmt.Sprintf("%v", cfg)))
	return hex.EncodeToString(hash[:])
}

// Helper functions
func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func min(a, b string) string {
	if a < b {
		return a
	}
	return b
}

func max(a, b string) string {
	if a > b {
		return a
	}
	return b
}
