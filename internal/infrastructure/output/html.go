package output

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// HTMLReporter implements the ports.Reporter interface for HTML output
type HTMLReporter struct {
	tool    string
	version string
}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{
		version: "1.0",
		tool:    "arx",
	}
}

// htmlReportData holds the data for HTML template rendering
type htmlReportData struct {
	ProjectRoot      string
	Timestamp        string
	ConfigHash       string
	Violations       []htmlViolation
	ErrorCount       int
	WarningCount     int
	InfoCount        int
	TotalCount       int
	CouplingMatrix   interface{}
	CouplingRows     []couplingRow
	DebtScore        *debtScoreData
	TrendReport      *trendReportData
	CSS              string
}

// htmlViolation represents a violation for HTML rendering
type htmlViolation struct {
	ID               string
	RuleID           string
	SeverityClass    string
	File             string
	Line             int
	SourceLayer      string
	TargetLayer      string
	Import           string
	Message          string
	Overridden       bool
	OriginalSeverity string
}

// couplingRow represents a row in the coupling matrix table
type couplingRow struct {
	From       string
	To         string
	Count      int
	Percentage string
	Class      string
}

// debtScoreData holds technical debt score information
type debtScoreData struct {
	Total      int
	Class      string
	Trend      string
	TrendClass string
	TrendDelta string
	Breakdown  []debtBreakdownItem
}

// debtBreakdownItem represents a single debt breakdown item
type debtBreakdownItem struct {
	Label string
	Value string
}

// trendReportData holds trend report information
type trendReportData struct {
	Status             string
	Class              string
	Summary            string
	NewViolations      int
	ResolvedViolations int
}

// htmlStyles contains embedded CSS for HTML reports
const htmlStyles = `
:root { --color-error: #dc3545; --color-warning: #ffc107; --color-info: #17a2b8; --color-success: #28a745; --color-bg: #f8f9fa; --color-border: #dee2e6; --color-text: #212529; --color-text-muted: #6c757d; }
* { box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: var(--color-text); background: var(--color-bg); margin: 0; padding: 20px; }
.container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); padding: 30px; }
header { border-bottom: 2px solid var(--color-border); padding-bottom: 20px; margin-bottom: 30px; }
header h1 { margin: 0 0 10px 0; color: var(--color-text); font-size: 2rem; }
header .meta { color: var(--color-text-muted); font-size: 0.9rem; }
.violation { background: #fff; border: 1px solid var(--color-border); border-left: 4px solid var(--color-error); border-radius: 4px; padding: 15px; margin-bottom: 15px; }
.violation.warning { border-left-color: var(--color-warning); }
.violation.info { border-left-color: var(--color-info); }
.violation-header { font-weight: 600; margin-bottom: 8px; }
.violation-file { font-family: monospace; font-size: 0.9rem; color: var(--color-text-muted); }
.coupling-table { width: 100%; border-collapse: collapse; }
.coupling-table th, .coupling-table td { border: 1px solid var(--color-border); padding: 10px; text-align: left; }
.coupling-table th { background: var(--color-bg); font-weight: 600; }
.debt-score { font-size: 1.5rem; font-weight: 600; margin-bottom: 15px; }
.debt-score.good { color: var(--color-success); }
.debt-score.moderate { color: var(--color-warning); }
.debt-score.poor { color: var(--color-error); }
.debt-breakdown { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; margin-top: 15px; }
.debt-breakdown-item { background: var(--color-bg); padding: 10px; border-radius: 4px; text-align: center; }
.debt-breakdown-value { font-size: 1.25rem; font-weight: 700; display: block; }
.debt-breakdown-label { font-size: 0.85rem; color: var(--color-text-muted); }
.trend-section { margin-top: 20px; }
.trend-status { font-size: 1.25rem; font-weight: 600; padding: 10px 15px; border-radius: 4px; display: inline-block; }
.trend-status.improved { background: #d4edda; color: #155724; }
.trend-status.degraded { background: #f8d7da; color: #721c24; }
.trend-status.unchanged { background: #fff3cd; color: #856404; }
.trend-deltas { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 10px; margin-top: 15px; }
.trend-delta { background: var(--color-bg); padding: 10px; border-radius: 4px; text-align: center; }
.trend-delta-value { font-size: 1.1rem; font-weight: 600; }
.summary-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 20px; }
.summary-card { background: var(--color-bg); padding: 15px; border-radius: 4px; text-align: center; }
.summary-card-value { font-size: 2rem; font-weight: 700; display: block; }
.summary-card.errors .summary-card-value { color: var(--color-error); }
.summary-card.warnings .summary-card-value { color: var(--color-warning); }
.summary-card.info .summary-card-value { color: var(--color-info); }
.no-data { color: var(--color-text-muted); font-style: italic; }
@media print { body { background: white; padding: 0; } .container { box-shadow: none; padding: 0; } }
`

// htmlTemplate is the main HTML template
var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"safeCSS": func(s string) template.CSS {
		return template.CSS(s)
	},
}).Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Arx Architecture Report</title>
<style>{{.CSS | safeCSS}}</style>
</head>
<body>
<div class="container">
<header>
<h1>Architecture Audit Report</h1>
<div class="meta">
<span><strong>Project:</strong> {{.ProjectRoot}}</span>
<span><strong>Date:</strong> {{.Timestamp}}</span>
{{if .ConfigHash}}<span><strong>Config:</strong> {{.ConfigHash}}</span>{{end}}
</div>
</header>

<section id="summary">
<h2>Summary</h2>
<div class="summary-cards">
<div class="summary-card errors"><span class="summary-card-value">{{.ErrorCount}}</span><span class="summary-card-label">Errors</span></div>
<div class="summary-card warnings"><span class="summary-card-value">{{.WarningCount}}</span><span class="summary-card-label">Warnings</span></div>
<div class="summary-card info"><span class="summary-card-value">{{.InfoCount}}</span><span class="summary-card-label">Info</span></div>
</div>
</section>

{{if .Violations}}
<section id="violations">
<h2>Violations ({{.TotalCount}})</h2>
{{range .Violations}}
<div class="violation {{.SeverityClass}}">
<div class="violation-header">{{.ID | html}} — {{.RuleID | html}}{{if .Overridden}} (overridden from {{.OriginalSeverity | html}}){{end}}</div>
<div class="violation-file">{{.File | html}}:{{.Line}}</div>
<div class="violation-message">{{.Message | html}}</div>
</div>
{{end}}
</section>
{{else}}
<section id="violations">
<h2>Violations</h2>
<p>No violations found.</p>
</section>
{{end}}

<section id="coupling">
<h2>Coupling Matrix</h2>
{{if .CouplingRows}}
<table class="coupling-table">
<thead>
<tr><th>From</th><th>To</th><th>Count</th><th>Percentage</th></tr>
</thead>
<tbody>
{{range .CouplingRows}}
<tr class="{{.Class}}"><td>{{.From | html}}</td><td>{{.To | html}}</td><td>{{.Count}}</td><td>{{.Percentage}}</td></tr>
{{end}}
</tbody>
</table>
{{else}}
<p class="no-data">(no data)</p>
{{end}}
</section>

{{if .DebtScore}}
<section id="debt">
<h2>Technical Debt</h2>
<div class="debt-score {{.DebtScore.Class}}">Total: {{.DebtScore.Total}}</div>
{{if .DebtScore.Trend}}
<p>Trend: {{.DebtScore.Trend}} {{.DebtScore.TrendDelta}}</p>
{{end}}
{{if .DebtScore.Breakdown}}
<div class="debt-breakdown">
{{range .DebtScore.Breakdown}}
<div class="debt-breakdown-item">
<span class="debt-breakdown-value">{{.Value}}</span>
<span class="debt-breakdown-label">{{.Label}}</span>
</div>
{{end}}
</div>
{{end}}
</section>
{{end}}

{{if .TrendReport}}
<section id="trend">
<h2>Trend</h2>
<div class="trend-section">
<div class="trend-status {{.TrendReport.Class}}">{{.TrendReport.Status | html}}</div>
{{if .TrendReport.Summary}}
<p>{{.TrendReport.Summary | html}}</p>
{{end}}
<div class="trend-deltas">
<div class="trend-delta">
<div class="trend-delta-value">{{.TrendReport.NewViolations}}</div>
<div>New Violations</div>
</div>
<div class="trend-delta">
<div class="trend-delta-value">{{.TrendReport.ResolvedViolations}}</div>
<div>Resolved Violations</div>
</div>
</div>
</div>
</section>
{{end}}

</div>
</body>
</html>
`))

// Report outputs violations in HTML format
func (r *HTMLReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	if format != "html" {
		return fmt.Errorf("html reporter only supports html format")
	}

	errorCount := 0
	warningCount := 0
	infoCount := 0
	for _, v := range violations {
		switch v.Severity {
		case domain.SeverityWarning:
			warningCount++
		case domain.SeverityInfo:
			infoCount++
		default:
			errorCount++
		}
	}

	htmlViolations := make([]htmlViolation, 0, len(violations))
	for _, v := range violations {
		severityClass := "error"
		switch v.Severity {
		case domain.SeverityWarning:
			severityClass = "warning"
		case domain.SeverityInfo:
			severityClass = "info"
		}
		htmlViolations = append(htmlViolations, htmlViolation{
			ID:               v.ID,
			RuleID:           v.RuleID,
			SeverityClass:    severityClass,
			File:             v.File,
			Line:             v.Line,
			SourceLayer:      v.SourceLayer,
			TargetLayer:      v.TargetLayer,
			Import:           v.Import,
			Message:          v.Message,
			Overridden:       v.Overridden,
			OriginalSeverity: string(v.OriginalSeverity),
		})
	}

	sort.Slice(htmlViolations, func(i, j int) bool {
		if htmlViolations[i].File != htmlViolations[j].File {
			return htmlViolations[i].File < htmlViolations[j].File
		}
		return htmlViolations[i].Line < htmlViolations[j].Line
	})

	data := htmlReportData{
		CSS:          htmlStyles,
		ProjectRoot:  ".",
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Violations:   htmlViolations,
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		InfoCount:    infoCount,
		TotalCount:   len(violations),
	}

	if err := htmlTemplate.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}

	return nil
}

// ReportAudit renders a full audit report including coupling matrix, debt score, and trends.
func (r *HTMLReporter) ReportAudit(report *domain.AuditReport) error {
	errorCount := 0
	warningCount := 0
	infoCount := 0
	for _, v := range report.Violations {
		switch v.Severity {
		case domain.SeverityWarning:
			warningCount++
		case domain.SeverityInfo:
			infoCount++
		default:
			errorCount++
		}
	}

	htmlViolations := make([]htmlViolation, 0, len(report.Violations))
	for _, v := range report.Violations {
		severityClass := "error"
		switch v.Severity {
		case domain.SeverityWarning:
			severityClass = "warning"
		case domain.SeverityInfo:
			severityClass = "info"
		}
		htmlViolations = append(htmlViolations, htmlViolation{
			ID:               v.ID,
			RuleID:           v.RuleID,
			SeverityClass:    severityClass,
			File:             v.File,
			Line:             v.Line,
			SourceLayer:      v.SourceLayer,
			TargetLayer:      v.TargetLayer,
			Import:           v.Import,
			Message:          v.Message,
			Overridden:       v.Overridden,
			OriginalSeverity: string(v.OriginalSeverity),
		})
	}

	sort.Slice(htmlViolations, func(i, j int) bool {
		if htmlViolations[i].File != htmlViolations[j].File {
			return htmlViolations[i].File < htmlViolations[j].File
		}
		return htmlViolations[i].Line < htmlViolations[j].Line
	})

	// Build coupling rows
	var couplingRows []couplingRow
	totalDeps := report.CouplingMatrix.Count()
	if totalDeps > 0 {
		var fromLayers []string
		for from := range report.CouplingMatrix.FromTo {
			fromLayers = append(fromLayers, from)
		}
		sort.Strings(fromLayers)
		for _, from := range fromLayers {
			var toLayers []string
			for to := range report.CouplingMatrix.FromTo[from] {
				toLayers = append(toLayers, to)
			}
			sort.Strings(toLayers)
			for _, to := range toLayers {
				count := report.CouplingMatrix.FromTo[from][to]
				pct := ""
				if totalDeps > 0 {
					pct = fmt.Sprintf("%.1f%%", float64(count)/float64(totalDeps)*100)
				}
				class := ""
				if count > 10 {
					class = "high"
				} else if count > 5 {
					class = "medium"
				}
				couplingRows = append(couplingRows, couplingRow{
					From:       from,
					To:         to,
					Count:      count,
					Percentage: pct,
					Class:      class,
				})
			}
		}
	}

	// Build debt score data
	var debtData *debtScoreData
	if report.DebtScore.Total > 0 || len(report.DebtScore.BySeverity) > 0 {
		debtClass := "good"
		if report.DebtScore.Total > 50 {
			debtClass = "poor"
		} else if report.DebtScore.Total > 20 {
			debtClass = "moderate"
		}
		trendDelta := ""
		if report.DebtScore.TrendDelta != 0 {
			if report.DebtScore.TrendDelta > 0 {
				trendDelta = fmt.Sprintf("(+%d)", report.DebtScore.TrendDelta)
			} else {
				trendDelta = fmt.Sprintf("(%d)", report.DebtScore.TrendDelta)
			}
		}
		var breakdown []debtBreakdownItem
		for _, sev := range []string{"error", "warning", "info"} {
			if count, ok := report.DebtScore.BySeverity[sev]; ok && count > 0 {
				breakdown = append(breakdown, debtBreakdownItem{
					Label: sev,
					Value: fmt.Sprintf("%d", count),
				})
			}
		}
		debtData = &debtScoreData{
			Total:      report.DebtScore.Total,
			Class:      debtClass,
			Trend:      report.DebtScore.Trend,
			TrendDelta: trendDelta,
			Breakdown:  breakdown,
		}
	}

	// Build trend report data
	var trendData *trendReportData
	if report.TrendReport.Status != "" {
		trendClass := string(report.TrendReport.Status)
		newV := 0
		resolvedV := 0
		if report.TrendReport.ViolationDelta > 0 {
			newV = report.TrendReport.ViolationDelta
		} else if report.TrendReport.ViolationDelta < 0 {
			resolvedV = -report.TrendReport.ViolationDelta
		}
		trendData = &trendReportData{
			Status:             string(report.TrendReport.Status),
			Class:              trendClass,
			Summary:            report.TrendReport.Summary,
			NewViolations:      newV,
			ResolvedViolations: resolvedV,
		}
	}

	data := htmlReportData{
		CSS:          htmlStyles,
		ProjectRoot:  report.ProjectRoot,
		Timestamp:    report.Timestamp.Format("2006-01-02 15:04:05"),
		ConfigHash:   report.ConfigHash,
		Violations:   htmlViolations,
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		InfoCount:    infoCount,
		TotalCount:   len(report.Violations),
		CouplingRows: couplingRows,
		DebtScore:    debtData,
		TrendReport:  trendData,
	}

	if err := htmlTemplate.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}

	return nil
}

// Ensure HTMLReporter implements ports.Reporter interface
var _ ports.Reporter = (*HTMLReporter)(nil)
