package application

import (
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
)

// CalculateDebt computes the technical debt score from violations and circular dependencies.
// Formula: (errors×3) + (warnings×1) + (infos×0) + (circular×5)
// Returns a DebtScore with total and breakdown by severity.
func CalculateDebt(violations []domain.Violation, circularPairs int) domain.DebtScore {
	debt := domain.NewDebtScore()

	// Count violations by severity
	for _, v := range violations {
		debt.AddViolation(string(v.Severity))
	}

	// Add circular dependency penalty: 5 points per circular pair
	circularPenalty := 0
	if circularPairs > 0 {
		circularPenalty = circularPairs * 5
		debt.BySeverity["circular"] = circularPenalty
	}

	// Calculate base score (errors, warnings, infos)
	debt.Calculate()

	// Add circular penalty to total
	debt.Total += circularPenalty

	return debt
}

// String returns a human-readable representation of the debt score.
func DebtScoreString(score domain.DebtScore) string {
	return fmt.Sprintf("DebtScore: Total=%d (error=%d, warning=%d, info=%d, circular=%d)",
		score.Total,
		score.BySeverity["error"],
		score.BySeverity["warning"],
		score.BySeverity["info"],
		score.BySeverity["circular"],
	)
}
