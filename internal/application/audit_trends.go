package application

import (
	"github.com/pauvalls/arx/internal/domain"
)

// CalculateTrends compares current audit with previous audit and returns a trend report.
// Compares: ViolationDelta (count difference), DebtDelta (score difference).
// Determines status: Improved (negative delta), Degraded (positive delta), Unchanged (zero delta).
// Handles no-history case: returns Unchanged status with appropriate summary.
func CalculateTrends(current *domain.AuditReport, previous *domain.AuditReport) domain.TrendReport {
	return domain.NewTrendReport(current, previous)
}
