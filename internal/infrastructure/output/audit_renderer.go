package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pauvalls/arx/internal/domain"
)

// AuditReportRenderer renders a complete audit report for terminal display
type AuditReportRenderer struct {
	width int
}

// NewAuditReportRenderer creates a new audit report renderer
func NewAuditReportRenderer() *AuditReportRenderer {
	return &AuditReportRenderer{
		width: 80,
	}
}

// Render produces a complete terminal output of the audit report
func (r *AuditReportRenderer) Render(report domain.AuditReport) string {
	var sections []string

	// Header
	sections = append(sections, r.renderHeader(report))

	// Violation summary
	sections = append(sections, r.renderViolationSummary(report))

	// Debt score
	sections = append(sections, r.renderDebtScore(report))

	// Coupling matrix
	sections = append(sections, r.renderCouplingMatrix(report))

	// Trend report (if available)
	if report.TrendReport.Status != "" {
		sections = append(sections, r.renderTrendReport(report))
	}

	return strings.Join(sections, "\n")
}

func (r *AuditReportRenderer) renderHeader(report domain.AuditReport) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	title := headerStyle.Render("Architecture Audit Report")
	project := dimStyle.Render(fmt.Sprintf("Project: %s", report.ProjectRoot))
	timestamp := dimStyle.Render(fmt.Sprintf("Date: %s", report.Timestamp.Format("2006-01-02 15:04")))

	return fmt.Sprintf("%s\n%s\n%s", title, project, timestamp)
}

func (r *AuditReportRenderer) renderViolationSummary(report domain.AuditReport) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	errors := 0
	warnings := 0
	infos := 0

	for _, v := range report.Violations {
		switch v.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		case "info":
			infos++
		}
	}

	total := len(report.Violations)

	header := sectionStyle.Render("Violations")

	summary := fmt.Sprintf("%d total: %s, %s, %s",
		total,
		errorStyle.Render(fmt.Sprintf("%d errors", errors)),
		warningStyle.Render(fmt.Sprintf("%d warnings", warnings)),
		infoStyle.Render(fmt.Sprintf("%d info", infos)),
	)

	return fmt.Sprintf("\n%s\n%s", header, summary)
}

func (r *AuditReportRenderer) renderDebtScore(report domain.AuditReport) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	score := report.DebtScore.Total
	trend := report.DebtScore.Trend
	delta := report.DebtScore.TrendDelta

	// Determine trend indicator
	var trendIndicator string
	var trendStyle lipgloss.Style

	switch trend {
	case "up":
		trendIndicator = "↑"
		trendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red for bad
	case "down":
		trendIndicator = "↓"
		trendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green for good
	default:
		trendIndicator = "→"
		trendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow for stable
	}

	trendText := trendStyle.Render(fmt.Sprintf("%s %d", trendIndicator, delta))

	scoreStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	scoreText := scoreStyle.Render(fmt.Sprintf("%d", score))

	header := sectionStyle.Render("Technical Debt")
	body := fmt.Sprintf("Score: %s (Trend: %s)", scoreText, trendText)

	return fmt.Sprintf("\n%s\n%s", header, body)
}

func (r *AuditReportRenderer) renderCouplingMatrix(report domain.AuditReport) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	matrix := report.CouplingMatrix
	if matrix.FromTo == nil || len(matrix.FromTo) == 0 {
		header := sectionStyle.Render("Coupling Matrix")
		return fmt.Sprintf("\n%s\n(no dependencies detected)", header)
	}

	// Get all unique layers
	layerSet := make(map[string]bool)
	for from, targets := range matrix.FromTo {
		layerSet[from] = true
		for to := range targets {
			layerSet[to] = true
		}
	}

	layers := make([]string, 0, len(layerSet))
	for layer := range layerSet {
		layers = append(layers, layer)
	}

	// Calculate column widths
	maxFrom := 15
	maxTo := 15
	for from, targets := range matrix.FromTo {
		for to := range targets {
			if len(from) > maxFrom {
				maxFrom = len(from)
			}
			if len(to) > maxTo {
				maxTo = len(to)
			}
		}
	}

	total := matrix.Count()

	// Build table
	var sb strings.Builder

	header := sectionStyle.Render("Coupling Matrix")
	sb.WriteString(fmt.Sprintf("\n%s\n", header))

	// Table header
	sb.WriteString(fmt.Sprintf("┌%s┬%s┬%s┬%s┐\n",
		strings.Repeat("─", maxFrom+2),
		strings.Repeat("─", maxTo+2),
		strings.Repeat("─", 7),
		strings.Repeat("─", 6)))

	sb.WriteString(fmt.Sprintf("│ %-*s │ %-*s │ Count │ Pct  │\n",
		maxFrom, "From", maxTo, "To"))

	sb.WriteString(fmt.Sprintf("├%s┼%s┼%s┼%s┤\n",
		strings.Repeat("─", maxFrom+2),
		strings.Repeat("─", maxTo+2),
		strings.Repeat("─", 7),
		strings.Repeat("─", 6)))

	// Data rows
	for from, targets := range matrix.FromTo {
		for to, count := range targets {
			pct := 0.0
			if total > 0 {
				pct = float64(count) / float64(total) * 100
			}

			// Color code based on percentage
			var pctStyle lipgloss.Style
			switch {
			case pct <= 5:
				pctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Green
			case pct <= 15:
				pctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow
			default:
				pctStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
			}

			pctText := pctStyle.Render(fmt.Sprintf("%5.1f%%", pct))

			sb.WriteString(fmt.Sprintf("│ %-*s │ %-*s │ %5d │ %s │\n",
				maxFrom, from, maxTo, to, count, pctText))
		}
	}

	// Table footer
	sb.WriteString(fmt.Sprintf("└%s┴%s┴%s┴%s┘",
		strings.Repeat("─", maxFrom+2),
		strings.Repeat("─", maxTo+2),
		strings.Repeat("─", 7),
		strings.Repeat("─", 6)))

	// Circular dependencies warning
	circularPairs := matrix.FindCircularPairs()
	if len(circularPairs) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		sb.WriteString(fmt.Sprintf("\n\n%s Circular dependencies detected: %d",
			warnStyle.Render("⚠"), len(circularPairs)))
		for _, pair := range circularPairs {
			sb.WriteString(fmt.Sprintf("\n  %s ↔ %s", pair[0], pair[1]))
		}
	}

	return sb.String()
}

func (r *AuditReportRenderer) renderTrendReport(report domain.AuditReport) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true)

	trend := report.TrendReport

	var statusStyle lipgloss.Style
	var statusText string

	switch trend.Status {
	case domain.TrendImproved:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		statusText = "✓ Improved"
	case domain.TrendDegraded:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		statusText = "✗ Degraded"
	default:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		statusText = "→ Unchanged"
	}

	header := sectionStyle.Render("Trend Report")
	body := fmt.Sprintf("Status: %s\nViolations: %d | Debt: %d\n%s",
		statusStyle.Render(statusText),
		trend.ViolationDelta,
		trend.DebtDelta,
		trend.Summary)

	return fmt.Sprintf("\n%s\n%s", header, body)
}
