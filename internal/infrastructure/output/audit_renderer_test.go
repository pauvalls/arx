package output

import (
	"strings"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestAuditReportRenderer_EmptyReport(t *testing.T) {
	t.Parallel()

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	if output == "" {
		t.Error("Expected non-empty output")
	}

	// Should show project root
	if !strings.Contains(output, "/test/project") {
		t.Error("Expected output to contain project root")
	}

	// Should show no violations message
	if !strings.Contains(output, "0 total") && !strings.Contains(output, "0 violations") {
		t.Error("Expected output to show 0 violations")
	}
}

func TestAuditReportRenderer_ViolationSummary(t *testing.T) {
	t.Parallel()

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
		Violations: []domain.Violation{
			{ID: "v1", Severity: "error", RuleID: "rule1"},
			{ID: "v2", Severity: "error", RuleID: "rule2"},
			{ID: "v3", Severity: "warning", RuleID: "rule3"},
			{ID: "v4", Severity: "info", RuleID: "rule4"},
		},
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Should show violation counts by severity
	if !strings.Contains(output, "2 errors") {
		t.Error("Expected output to show '2 errors'")
	}
	if !strings.Contains(output, "1 warning") {
		t.Error("Expected output to show '1 warning'")
	}
	if !strings.Contains(output, "1 info") {
		t.Error("Expected output to show '1 info'")
	}
}

func TestAuditReportRenderer_DebtScore(t *testing.T) {
	t.Parallel()

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
		DebtScore: domain.DebtScore{
			Total: 42,
			BySeverity: map[string]int{
				"error":   10,
				"warning": 12,
			},
			Trend:      "up",
			TrendDelta: 5,
		},
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Should show debt score
	if !strings.Contains(output, "42") {
		t.Error("Expected output to contain debt score '42'")
	}

	// Should show trend indicator for "up"
	if !strings.Contains(output, "↑") {
		t.Error("Expected output to contain upward trend indicator '↑'")
	}
}

func TestAuditReportRenderer_DebtScoreDownTrend(t *testing.T) {
	t.Parallel()

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
		DebtScore: domain.DebtScore{
			Total:      20,
			Trend:      "down",
			TrendDelta: -10,
		},
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Should show trend indicator for "down"
	if !strings.Contains(output, "↓") {
		t.Error("Expected output to contain downward trend indicator '↓'")
	}
}

func TestAuditReportRenderer_CouplingMatrix(t *testing.T) {
	t.Parallel()

	matrix := domain.NewCouplingMatrix()
	// Add multiple times to simulate counts
	for i := 0; i < 5; i++ {
		matrix.Add("application", "domain")
	}
	for i := 0; i < 3; i++ {
		matrix.Add("domain", "infrastructure")
	}
	for i := 0; i < 2; i++ {
		matrix.Add("infrastructure", "domain")
	}

	report := domain.AuditReport{
		Timestamp:      time.Now(),
		ProjectRoot:    "/test/project",
		CouplingMatrix: matrix,
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Should show coupling matrix header
	if !strings.Contains(output, "Coupling Matrix") && !strings.Contains(output, "coupling") {
		t.Error("Expected output to contain coupling matrix section")
	}

	// Should show layer names
	if !strings.Contains(output, "application") {
		t.Error("Expected output to contain 'application' layer")
	}
	if !strings.Contains(output, "domain") {
		t.Error("Expected output to contain 'domain' layer")
	}
}

func TestAuditReportRenderer_TrendReport(t *testing.T) {
	t.Parallel()

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
		TrendReport: domain.TrendReport{
			ViolationDelta: -3,
			DebtDelta:      -10,
			Status:         domain.TrendImproved,
			Summary:        "Architecture improved: reduced violations and debt",
		},
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Should show trend section
	if !strings.Contains(output, "Trend") && !strings.Contains(output, "trend") {
		t.Error("Expected output to contain trend section")
	}

	// Should show improvement indicator
	if !strings.Contains(output, "improved") && !strings.Contains(output, "Improved") {
		t.Error("Expected output to show improvement")
	}
}

func TestAuditReportRenderer_FullReport(t *testing.T) {
	t.Parallel()

	// Build a complete report with all sections
	matrix := domain.NewCouplingMatrix()
	for i := 0; i < 10; i++ {
		matrix.Add("application", "domain")
	}
	for i := 0; i < 2; i++ {
		matrix.Add("application", "infrastructure")
	}
	matrix.Add("domain", "infrastructure")

	report := domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/test/project",
		Violations: []domain.Violation{
			{ID: "v1", Severity: "error", RuleID: "domain-no-import-infra", SourceLayer: "domain", TargetLayer: "infrastructure"},
			{ID: "v2", Severity: "warning", RuleID: "app-no-import-infra", SourceLayer: "application", TargetLayer: "infrastructure"},
		},
		CouplingMatrix: matrix,
		DebtScore: domain.DebtScore{
			Total: 15,
			BySeverity: map[string]int{
				"error":   3,
				"warning": 6,
			},
			Trend:      "stable",
			TrendDelta: 0,
		},
		TrendReport: domain.TrendReport{
			ViolationDelta: -2,
			DebtDelta:      -5,
			Status:         domain.TrendImproved,
			Summary:        "Architecture improved",
		},
	}

	renderer := NewAuditReportRenderer()
	output := renderer.Render(report)

	// Verify all sections are present
	sections := []string{
		"/test/project",
		"error",
		"warning",
		"Coupling",
		"15", // debt score
		"Improved",
	}

	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected output to contain %q", section)
		}
	}
}
