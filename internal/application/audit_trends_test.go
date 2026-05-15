package application

import (
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestCalculateTrends_Improved(t *testing.T) {
	previous := &domain.AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityError},
			{ID: "V-003", Severity: domain.SeverityWarning},
			{ID: "V-004", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 10,
			BySeverity: map[string]int{
				"error":   2,
				"warning": 4,
			},
		},
	}

	current := &domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	trend := CalculateTrends(current, previous)

	if !trend.IsImproved() {
		t.Error("Expected trend to be improved")
	}
	if trend.IsDegraded() {
		t.Error("Expected trend to not be degraded")
	}
	if trend.ViolationDelta != -2 {
		t.Errorf("Expected violation delta = -2, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != -6 {
		t.Errorf("Expected debt delta = -6, got %d", trend.DebtDelta)
	}
	if trend.Status != domain.TrendImproved {
		t.Errorf("Expected status = %q, got %q", domain.TrendImproved, trend.Status)
	}
}

func TestCalculateTrends_Degraded(t *testing.T) {
	previous := &domain.AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
		},
		DebtScore: domain.DebtScore{
			Total: 3,
			BySeverity: map[string]int{
				"error": 1,
			},
		},
	}

	current := &domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityError},
			{ID: "V-003", Severity: domain.SeverityError},
			{ID: "V-004", Severity: domain.SeverityWarning},
			{ID: "V-005", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 11,
			BySeverity: map[string]int{
				"error":   3,
				"warning": 2,
			},
		},
	}

	trend := CalculateTrends(current, previous)

	if trend.IsImproved() {
		t.Error("Expected trend to not be improved")
	}
	if !trend.IsDegraded() {
		t.Error("Expected trend to be degraded")
	}
	if trend.ViolationDelta != 4 {
		t.Errorf("Expected violation delta = 4, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != 8 {
		t.Errorf("Expected debt delta = 8, got %d", trend.DebtDelta)
	}
	if trend.Status != domain.TrendDegraded {
		t.Errorf("Expected status = %q, got %q", domain.TrendDegraded, trend.Status)
	}
}

func TestCalculateTrends_NoChange(t *testing.T) {
	previous := &domain.AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	current := &domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	trend := CalculateTrends(current, previous)

	if trend.IsImproved() {
		t.Error("Expected trend to not be improved")
	}
	if trend.IsDegraded() {
		t.Error("Expected trend to not be degraded")
	}
	if trend.ViolationDelta != 0 {
		t.Errorf("Expected violation delta = 0, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != 0 {
		t.Errorf("Expected debt delta = 0, got %d", trend.DebtDelta)
	}
	if trend.Status != domain.TrendUnchanged {
		t.Errorf("Expected status = %q, got %q", domain.TrendUnchanged, trend.Status)
	}
}

func TestCalculateTrends_NoHistory(t *testing.T) {
	current := &domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []domain.Violation{
			{ID: "V-001", Severity: domain.SeverityError},
			{ID: "V-002", Severity: domain.SeverityWarning},
		},
		DebtScore: domain.DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	trend := CalculateTrends(current, nil)

	if trend.Status != domain.TrendUnchanged {
		t.Errorf("Expected status = %q when no previous audit, got %q", domain.TrendUnchanged, trend.Status)
	}
	if trend.Summary != "No previous audit for comparison" {
		t.Errorf("Expected summary for no previous audit, got %q", trend.Summary)
	}
	if trend.ViolationDelta != 0 {
		t.Errorf("Expected violation delta = 0 when no history, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != 0 {
		t.Errorf("Expected debt delta = 0 when no history, got %d", trend.DebtDelta)
	}
}
