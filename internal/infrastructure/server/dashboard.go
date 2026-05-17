package server

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

//go:embed dashboard.html
var dashboardHTML string

// dashboardData holds the data passed to the dashboard template.
type dashboardData struct {
	Version        string
	Uptime         string
	LastCheck      time.Time
	Violations     []dashboardViolation
	ErrorCount     int
	WarningCount   int
	InfoCount      int
	CouplingEntries []couplingEntry
	DebtTotal      int
	DebtErrors     int
	DebtWarnings   int
	DebtInfo       int
	HasDebt        bool
}

// dashboardViolation is a flattened violation for template rendering.
type dashboardViolation struct {
	Severity     string
	SeverityClass string
	RuleID       string
	File         string
	Line         int
	SourceLayer  string
	TargetLayer  string
	Message      string
}

// couplingEntry represents a single coupling matrix row.
type couplingEntry struct {
	From       string
	To         string
	Count      int
	Percentage string
	Class      string
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(dashboardHTML))

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	violations := s.state.Violations()
	coupling := s.state.Coupling()
	debt := s.state.Debt()
	lastCheck := s.state.LastCheck()
	version := s.state.Version()
	uptime := s.state.Uptime()

	data := buildDashboardData(violations, coupling, debt, lastCheck, version, uptime)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(w, data); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
	}
}

func buildDashboardData(
	violations []domain.Violation,
	coupling domain.CouplingMatrix,
	debt domain.DebtScore,
	lastCheck time.Time,
	version VersionInfo,
	uptime time.Time,
) dashboardData {
	errorCount := 0
	warningCount := 0
	infoCount := 0

	dv := make([]dashboardViolation, 0, len(violations))
	for _, v := range violations {
		severityClass := "error"
		switch v.Severity {
		case domain.SeverityWarning:
			warningCount++
			severityClass = "warning"
		case domain.SeverityInfo:
			infoCount++
			severityClass = "info"
		default:
			errorCount++
		}
		dv = append(dv, dashboardViolation{
			Severity:      string(v.Severity),
			SeverityClass: severityClass,
			RuleID:        v.RuleID,
			File:          v.File,
			Line:          v.Line,
			SourceLayer:   v.SourceLayer,
			TargetLayer:   v.TargetLayer,
			Message:       v.Message,
		})
	}

	entries := buildCouplingEntries(coupling)

	debtBySev := debt.BySeverity
	if debtBySev == nil {
		debtBySev = map[string]int{}
	}
	debtErrors := debtBySev["error"]
	debtWarnings := debtBySev["warning"]
	debtInfo := debtBySev["info"]

	return dashboardData{
		Version:         version.Version,
		Uptime:          time.Since(uptime).Truncate(time.Second).String(),
		LastCheck:       lastCheck,
		Violations:      dv,
		ErrorCount:      errorCount,
		WarningCount:    warningCount,
		InfoCount:       infoCount,
		CouplingEntries: entries,
		DebtTotal:       debt.Total,
		DebtErrors:      debtErrors,
		DebtWarnings:    debtWarnings,
		DebtInfo:        debtInfo,
		HasDebt:         debt.Total > 0 || len(debtBySev) > 0,
	}
}

func buildCouplingEntries(matrix domain.CouplingMatrix) []couplingEntry {
	raw := matrix.GetEntriesWithPercentage()
	if len(raw) == 0 {
		return nil
	}

	entries := make([]couplingEntry, 0, len(raw))
	for _, e := range raw {
		class := ""
		if e.Count > 10 {
			class = "coupling-high"
		} else if e.Count > 5 {
			class = "coupling-medium"
		}
		entries = append(entries, couplingEntry{
			From:       e.FromLayer,
			To:         e.ToLayer,
			Count:      e.Count,
			Percentage: formatPercentage(e.Percentage),
			Class:      class,
		})
	}
	return entries
}

func formatPercentage(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}
