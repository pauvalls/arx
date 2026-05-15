package domain

import (
	"time"
)

// AuditReport represents the complete result of an architecture audit
type AuditReport struct {
	Timestamp      time.Time      `json:"timestamp"`
	ProjectRoot    string         `json:"project_root"`
	ConfigHash     string         `json:"config_hash"`
	Violations     []Violation    `json:"violations"`
	CouplingMatrix CouplingMatrix `json:"coupling_matrix"`
	DebtScore      DebtScore      `json:"debt_score"`
	TrendReport    TrendReport    `json:"trend_report,omitempty"`
}

// CouplingMatrix represents the dependency counts between layers
// Key structure: map[from_layer]map[to_layer]count
type CouplingMatrix struct {
	FromTo map[string]map[string]int `json:"from_to"`
}

// NewCouplingMatrix creates a new empty coupling matrix
func NewCouplingMatrix() CouplingMatrix {
	return CouplingMatrix{
		FromTo: make(map[string]map[string]int),
	}
}

// Add increments the dependency count from one layer to another
func (m *CouplingMatrix) Add(fromLayer, toLayer string) {
	if m.FromTo == nil {
		m.FromTo = make(map[string]map[string]int)
	}
	if m.FromTo[fromLayer] == nil {
		m.FromTo[fromLayer] = make(map[string]int)
	}
	m.FromTo[fromLayer][toLayer]++
}

// Get returns the dependency count between two layers
func (m *CouplingMatrix) Get(fromLayer, toLayer string) int {
	if m.FromTo == nil {
		return 0
	}
	if m.FromTo[fromLayer] == nil {
		return 0
	}
	return m.FromTo[fromLayer][toLayer]
}

// Count returns the total number of dependencies in the matrix
func (m *CouplingMatrix) Count() int {
	total := 0
	for _, targets := range m.FromTo {
		for _, count := range targets {
			total += count
		}
	}
	return total
}

// DebtScore represents the technical debt calculation
type DebtScore struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"` // error/warning/info → count
	Trend      string         `json:"trend"`       // "up", "down", "stable"
	TrendDelta int            `json:"trend_delta"` // numeric delta vs previous audit
}

// NewDebtScore creates a new debt score with initialized severity map
func NewDebtScore() DebtScore {
	return DebtScore{
		BySeverity: make(map[string]int),
		Trend:      "stable",
	}
}

// Calculate computes the total debt score from severity counts
// Formula: (error_count × 3) + (warning_count × 1) + (info_count × 0)
func (d *DebtScore) Calculate() {
	if d.BySeverity == nil {
		d.BySeverity = make(map[string]int)
	}

	errors := d.BySeverity["error"]
	warnings := d.BySeverity["warning"]
	infos := d.BySeverity["info"]

	d.Total = (errors * 3) + (warnings * 1) + (infos * 0)
}

// AddViolation adds a violation to the debt score by severity
func (d *DebtScore) AddViolation(severity string) {
	if d.BySeverity == nil {
		d.BySeverity = make(map[string]int)
	}
	d.BySeverity[severity]++
	d.Calculate()
}

// SetTrend sets the trend direction and delta
func (d *DebtScore) SetTrend(delta int) {
	d.TrendDelta = delta
	if delta > 0 {
		d.Trend = "up"
	} else if delta < 0 {
		d.Trend = "down"
	} else {
		d.Trend = "stable"
	}
}

// TrendStatus represents the trend direction
type TrendStatus string

const (
	TrendImproved   TrendStatus = "improved"
	TrendDegraded   TrendStatus = "degraded"
	TrendUnchanged  TrendStatus = "unchanged"
)

// TrendReport represents the comparison between current and previous audits
type TrendReport struct {
	ViolationDelta int         `json:"violation_delta"` // negative = improved
	DebtDelta      int         `json:"debt_delta"`      // negative = improved
	Status         TrendStatus `json:"status"`          // improved/degraded/unchanged
	Summary        string      `json:"summary"`         // human-readable summary
}

// NewTrendReport creates a new trend report comparing current and previous audits
func NewTrendReport(current, previous *AuditReport) TrendReport {
	if previous == nil {
		return TrendReport{
			Status:  TrendUnchanged,
			Summary: "No previous audit for comparison",
		}
	}

	currentViolations := len(current.Violations)
	previousViolations := len(previous.Violations)
	violationDelta := currentViolations - previousViolations

	debtDelta := current.DebtScore.Total - previous.DebtScore.Total

	// Determine overall status
	var status TrendStatus
	if violationDelta < 0 || debtDelta < 0 {
		status = TrendImproved
	} else if violationDelta > 0 || debtDelta > 0 {
		status = TrendDegraded
	} else {
		status = TrendUnchanged
	}

	// Generate summary
	summary := generateTrendSummary(violationDelta, debtDelta, status)

	return TrendReport{
		ViolationDelta: violationDelta,
		DebtDelta:      debtDelta,
		Status:         status,
		Summary:        summary,
	}
}

// generateTrendSummary creates a human-readable trend summary
func generateTrendSummary(violationDelta, debtDelta int, status TrendStatus) string {
	switch status {
	case TrendImproved:
		if violationDelta < 0 && debtDelta < 0 {
			return "Architecture improved: reduced violations and debt"
		} else if violationDelta < 0 {
			return "Architecture improved: reduced violations"
		} else {
			return "Architecture improved: reduced technical debt"
		}
	case TrendDegraded:
		if violationDelta > 0 && debtDelta > 0 {
			return "Architecture degraded: increased violations and debt"
		} else if violationDelta > 0 {
			return "Architecture degraded: increased violations"
		} else {
			return "Architecture degraded: increased technical debt"
		}
	default:
		return "Architecture unchanged from previous audit"
	}
}

// IsImproved returns true if the trend shows improvement
func (t *TrendReport) IsImproved() bool {
	return t.Status == TrendImproved
}

// IsDegraded returns true if the trend shows degradation
func (t *TrendReport) IsDegraded() bool {
	return t.Status == TrendDegraded
}

// AuditMetrics represents additional audit statistics
type AuditMetrics struct {
	TotalFiles       int     `json:"total_files"`
	TotalImports     int     `json:"total_imports"`
	ViolationDensity float64 `json:"violation_density"` // violations per KLOC
}

// CalculateViolationDensity computes violations per thousand lines of code
func (m *AuditMetrics) CalculateViolationDensity(violations int, linesOfCode int) {
	if linesOfCode == 0 {
		m.ViolationDensity = 0
		return
	}
	m.ViolationDensity = float64(violations) / (float64(linesOfCode) / 1000.0)
}
