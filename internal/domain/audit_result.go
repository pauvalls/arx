package domain

import (
	"fmt"
	"time"
)

// AuditReport represents the result of an architecture audit
type AuditReport struct {
	Timestamp      time.Time     `json:"timestamp"`
	ProjectRoot    string        `json:"project_root"`
	ConfigHash     string        `json:"config_hash"`
	Violations     []Violation   `json:"violations"`
	CouplingMatrix CouplingMatrix `json:"coupling_matrix"`
	DebtScore      DebtScore     `json:"debt_score"`
	TrendReport    *TrendReport  `json:"trend_report,omitempty"`
}

// CouplingMatrix represents dependencies between layers
type CouplingMatrix struct {
	FromTo map[string]map[string]int `json:"from_to"` // fromLayer -> toLayer -> count
}

// Add increments the dependency count between two layers
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
	if toMap, ok := m.FromTo[fromLayer]; ok {
		return toMap[toLayer]
	}
	return 0
}

// Count returns total number of dependencies
func (m *CouplingMatrix) Count() int {
	total := 0
	for _, toMap := range m.FromTo {
		for _, count := range toMap {
			total += count
		}
	}
	return total
}

// DebtScore represents technical debt estimation
type DebtScore struct {
	Total       int            `json:"total"`
	BySeverity  map[string]int `json:"by_severity"` // severity -> count
	Trend       string         `json:"trend"`       // "improved", "degraded", "unchanged"
	TrendDelta  int            `json:"trend_delta"` // change from previous audit
}

// TrendReport represents comparison with previous audits
type TrendReport struct {
	ViolationDelta int    `json:"violation_delta"` // negative = improved
	DebtDelta      int    `json:"debt_delta"`
	Status         string `json:"status"` // "improved", "degraded", "unchanged"
	Summary        string `json:"summary"`
}

// IsImproved returns true if the trend is positive
func (t *TrendReport) IsImproved() bool {
	return t.Status == "improved"
}

// IsDegraded returns true if the trend is negative
func (t *TrendReport) IsDegraded() bool {
	return t.Status == "degraded"
}

// NewTrendReport creates a trend report comparing current and previous audits
func NewTrendReport(current, previous *AuditReport) *TrendReport {
	if previous == nil {
		return &TrendReport{
			Status:  "unchanged",
			Summary: "No previous audit data available",
		}
	}

	violationDelta := len(current.Violations) - len(previous.Violations)
	debtDelta := current.DebtScore.Total - previous.DebtScore.Total

	var status string
	if violationDelta < 0 || debtDelta < 0 {
		status = "improved"
	} else if violationDelta > 0 || debtDelta > 0 {
		status = "degraded"
	} else {
		status = "unchanged"
	}

	summary := fmt.Sprintf("Violations: %d (Δ %d), Debt: %d (Δ %d)",
		len(current.Violations), violationDelta,
		current.DebtScore.Total, debtDelta)

	return &TrendReport{
		ViolationDelta: violationDelta,
		DebtDelta:      debtDelta,
		Status:         status,
		Summary:        summary,
	}
}
