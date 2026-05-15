package domain

import (
	"testing"
	"time"
)

func TestAuditReport_StructFields(t *testing.T) {
	now := time.Now()
	report := AuditReport{
		Timestamp:   now,
		ProjectRoot: "/home/user/project",
		ConfigHash:  "abc123",
		Violations: []Violation{
			{
				ID:          "V-001",
				RuleID:      "R1",
				File:        "internal/domain/user.go",
				Line:        10,
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/arx/internal/infrastructure/db",
				Message:     "domain cannot depend on infrastructure",
				Severity:    SeverityError,
			},
		},
		CouplingMatrix: NewCouplingMatrix(),
		DebtScore:      NewDebtScore(),
		TrendReport: TrendReport{
			ViolationDelta: -2,
			DebtDelta:      -5,
			Status:         TrendImproved,
			Summary:        "Architecture improved",
		},
	}

	// Verify all fields are accessible
	if report.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if report.ProjectRoot == "" {
		t.Error("ProjectRoot should not be empty")
	}
	if report.ConfigHash == "" {
		t.Error("ConfigHash should not be empty")
	}
	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(report.Violations))
	}
	if report.Violations[0].ID != "V-001" {
		t.Errorf("Expected violation ID 'V-001', got '%s'", report.Violations[0].ID)
	}
	if report.DebtScore.BySeverity == nil {
		t.Error("DebtScore.BySeverity should be initialized")
	}
	if report.CouplingMatrix.FromTo == nil {
		t.Error("CouplingMatrix.FromTo should be initialized")
	}
}

func TestCouplingMatrix_Add(t *testing.T) {
	matrix := NewCouplingMatrix()

	// Add dependencies
	matrix.Add("domain", "infrastructure")
	matrix.Add("domain", "infrastructure")
	matrix.Add("application", "domain")
	matrix.Add("domain", "application")

	// Verify counts
	if matrix.Get("domain", "infrastructure") != 2 {
		t.Errorf("Expected domain→infrastructure count = 2, got %d", matrix.Get("domain", "infrastructure"))
	}
	if matrix.Get("application", "domain") != 1 {
		t.Errorf("Expected application→domain count = 1, got %d", matrix.Get("application", "domain"))
	}
	if matrix.Get("domain", "application") != 1 {
		t.Errorf("Expected domain→application count = 1, got %d", matrix.Get("domain", "application"))
	}
	if matrix.Get("infrastructure", "domain") != 0 {
		t.Errorf("Expected infrastructure→domain count = 0, got %d", matrix.Get("infrastructure", "domain"))
	}

	// Verify total count
	if matrix.Count() != 4 {
		t.Errorf("Expected total count = 4, got %d", matrix.Count())
	}
}

func TestCouplingMatrix_Add_NilInitialization(t *testing.T) {
	matrix := CouplingMatrix{}

	// Should handle nil FromTo gracefully
	matrix.Add("domain", "infrastructure")

	if matrix.Get("domain", "infrastructure") != 1 {
		t.Errorf("Expected count = 1 after add, got %d", matrix.Get("domain", "infrastructure"))
	}
}

func TestDebtScore_Calculate(t *testing.T) {
	tests := []struct {
		name     string
		bySeverity map[string]int
		wantTotal int
	}{
		{
			name: "only errors",
			bySeverity: map[string]int{
				"error": 5,
			},
			wantTotal: 15, // 5 * 3
		},
		{
			name: "only warnings",
			bySeverity: map[string]int{
				"warning": 10,
			},
			wantTotal: 10, // 10 * 1
		},
		{
			name: "mixed severities",
			bySeverity: map[string]int{
				"error":   3,
				"warning": 5,
				"info":    10,
			},
			wantTotal: 14, // (3 * 3) + (5 * 1) + (10 * 0)
		},
		{
			name:     "empty",
			bySeverity: map[string]int{},
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debt := DebtScore{
				BySeverity: tt.bySeverity,
			}
			debt.Calculate()
			if debt.Total != tt.wantTotal {
				t.Errorf("Calculate() total = %d, want %d", debt.Total, tt.wantTotal)
			}
		})
	}
}

func TestDebtScore_AddViolation(t *testing.T) {
	debt := NewDebtScore()

	debt.AddViolation("error")
	debt.AddViolation("error")
	debt.AddViolation("warning")
	debt.AddViolation("info")

	if debt.BySeverity["error"] != 2 {
		t.Errorf("Expected 2 errors, got %d", debt.BySeverity["error"])
	}
	if debt.BySeverity["warning"] != 1 {
		t.Errorf("Expected 1 warning, got %d", debt.BySeverity["warning"])
	}
	if debt.BySeverity["info"] != 1 {
		t.Errorf("Expected 1 info, got %d", debt.BySeverity["info"])
	}

	// Total should be: (2 * 3) + (1 * 1) + (1 * 0) = 7
	if debt.Total != 7 {
		t.Errorf("Expected total = 7, got %d", debt.Total)
	}
}

func TestDebtScore_SetTrend(t *testing.T) {
	tests := []struct {
		name      string
		delta     int
		wantTrend string
	}{
		{name: "debt increased", delta: 5, wantTrend: "up"},
		{name: "debt decreased", delta: -3, wantTrend: "down"},
		{name: "debt unchanged", delta: 0, wantTrend: "stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debt := NewDebtScore()
			debt.SetTrend(tt.delta)

			if debt.Trend != tt.wantTrend {
				t.Errorf("SetTrend(%d) trend = %q, want %q", tt.delta, debt.Trend, tt.wantTrend)
			}
			if debt.TrendDelta != tt.delta {
				t.Errorf("SetTrend(%d) delta = %d, want %d", tt.delta, debt.TrendDelta, tt.delta)
			}
		})
	}
}

func TestTrendReport_Improved(t *testing.T) {
	previous := &AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
			{ID: "V-002", Severity: SeverityError},
			{ID: "V-003", Severity: SeverityWarning},
		},
		DebtScore: DebtScore{
			Total: 10,
			BySeverity: map[string]int{
				"error":   2,
				"warning": 4,
			},
		},
	}

	current := &AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
		},
		DebtScore: DebtScore{
			Total: 3,
			BySeverity: map[string]int{
				"error": 1,
			},
		},
	}

	trend := NewTrendReport(current, previous)

	if !trend.IsImproved() {
		t.Error("Expected trend to be improved")
	}
	if trend.IsDegraded() {
		t.Error("Expected trend to not be degraded")
	}
	if trend.ViolationDelta != -2 {
		t.Errorf("Expected violation delta = -2, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != -7 {
		t.Errorf("Expected debt delta = -7, got %d", trend.DebtDelta)
	}
	if trend.Status != TrendImproved {
		t.Errorf("Expected status = %q, got %q", TrendImproved, trend.Status)
	}
}

func TestTrendReport_Degraded(t *testing.T) {
	previous := &AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
		},
		DebtScore: DebtScore{
			Total: 3,
			BySeverity: map[string]int{
				"error": 1,
			},
		},
	}

	current := &AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
			{ID: "V-002", Severity: SeverityError},
			{ID: "V-003", Severity: SeverityError},
			{ID: "V-004", Severity: SeverityWarning},
		},
		DebtScore: DebtScore{
			Total: 10,
			BySeverity: map[string]int{
				"error":   3,
				"warning": 1,
			},
		},
	}

	trend := NewTrendReport(current, previous)

	if trend.IsImproved() {
		t.Error("Expected trend to not be improved")
	}
	if !trend.IsDegraded() {
		t.Error("Expected trend to be degraded")
	}
	if trend.ViolationDelta != 3 {
		t.Errorf("Expected violation delta = 3, got %d", trend.ViolationDelta)
	}
	if trend.DebtDelta != 7 {
		t.Errorf("Expected debt delta = 7, got %d", trend.DebtDelta)
	}
	if trend.Status != TrendDegraded {
		t.Errorf("Expected status = %q, got %q", TrendDegraded, trend.Status)
	}
}

func TestTrendReport_Unchanged(t *testing.T) {
	previous := &AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
			{ID: "V-002", Severity: SeverityWarning},
		},
		DebtScore: DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	current := &AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations: []Violation{
			{ID: "V-001", Severity: SeverityError},
			{ID: "V-002", Severity: SeverityWarning},
		},
		DebtScore: DebtScore{
			Total: 4,
			BySeverity: map[string]int{
				"error":   1,
				"warning": 1,
			},
		},
	}

	trend := NewTrendReport(current, previous)

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
	if trend.Status != TrendUnchanged {
		t.Errorf("Expected status = %q, got %q", TrendUnchanged, trend.Status)
	}
}

func TestTrendReport_NoPrevious(t *testing.T) {
	current := &AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: "/home/user/project",
		Violations:  []Violation{},
		DebtScore:   DebtScore{Total: 0},
	}

	trend := NewTrendReport(current, nil)

	if trend.Status != TrendUnchanged {
		t.Errorf("Expected status = %q when no previous audit, got %q", TrendUnchanged, trend.Status)
	}
	if trend.Summary != "No previous audit for comparison" {
		t.Errorf("Expected summary for no previous audit, got %q", trend.Summary)
	}
}

func TestAuditMetrics_CalculateViolationDensity(t *testing.T) {
	tests := []struct {
		name         string
		violations   int
		linesOfCode  int
		wantDensity  float64
	}{
		{name: "10 violations in 1000 LOC", violations: 10, linesOfCode: 1000, wantDensity: 10.0},
		{name: "5 violations in 5000 LOC", violations: 5, linesOfCode: 5000, wantDensity: 1.0},
		{name: "100 violations in 10000 LOC", violations: 100, linesOfCode: 10000, wantDensity: 10.0},
		{name: "zero LOC", violations: 10, linesOfCode: 0, wantDensity: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := AuditMetrics{}
			metrics.CalculateViolationDensity(tt.violations, tt.linesOfCode)

			if metrics.ViolationDensity != tt.wantDensity {
				t.Errorf("CalculateViolationDensity(%d, %d) = %f, want %f",
					tt.violations, tt.linesOfCode, metrics.ViolationDensity, tt.wantDensity)
			}
		})
	}
}
