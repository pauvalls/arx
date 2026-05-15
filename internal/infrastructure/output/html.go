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
.summary-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 20px; }
.summary-card { background: var(--color-bg); padding: 15px; border-radius: 4px; text-align: center; }
.summary-card-value { font-size: 2rem; font-weight: 700; display: block; }
.summary-card.errors .summary-card-value { color: var(--color-error); }
.summary-card.warnings .summary-card-value { color: var(--color-warning); }
.summary-card.info .summary-card-value { color: var(--color-info); }
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

// Ensure HTMLReporter implements ports.Reporter interface
var _ ports.Reporter = (*HTMLReporter)(nil)
