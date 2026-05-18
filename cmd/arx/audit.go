package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/history"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// auditCmd represents the audit command
var auditCmd = &cobra.Command{
	Use:   "audit [path]",
	Short: "Run architecture audit with health metrics and trends",
	Long: `Run a comprehensive architecture audit on a project by loading the configuration,
detecting dependencies, evaluating rules, and generating a health report with metrics.

The audit includes:
  - Architecture violations (same as 'arx check')
  - Coupling matrix showing dependencies between layers
  - Technical debt score calculation
  - Trend comparison with previous audits (if history exists)

If no path is provided, the current directory is used.

Output formats:
  - terminal: Human-readable output with ASCII tables (default)
  - json: Machine-readable JSON output
  - html: HTML report for browser viewing

Flags:
  --trend          Show only trend comparison with previous audit
  --since DATE     Show trends since date (format: YYYY-MM-DD)

Exit codes:
  0 - No violations found
  1 - Violations found or error occurred

Example:
  arx audit                    # Audit current directory (terminal output)
  arx audit ./my-project       # Audit specific directory
  arx audit --format json      # JSON output for CI/CD
  arx audit --format html      # HTML report for browsers
  arx audit -o report.html     # Write HTML output to file
  arx audit --trend            # Show only trend comparison
  arx audit --since 2026-04-01 # Show audits since April 1st`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAudit,
}

var (
	auditOutput string
	auditFormat string
	auditTrend  bool
	auditSince  string
)

func init() {
	auditCmd.Flags().StringVarP(&auditOutput, "output", "o", "", "Output file path (default: stdout)")
	auditCmd.Flags().StringVarP(&auditFormat, "format", "f", "terminal", "Output format: terminal|json|html")
	auditCmd.Flags().BoolVar(&auditTrend, "trend", false, "Show only trend comparison with previous audit")
	auditCmd.Flags().StringVar(&auditSince, "since", "", "Show trends since date (format: YYYY-MM-DD)")
	rootCmd.AddCommand(auditCmd)
}

func runAudit(cmd *cobra.Command, args []string) error {
	// Determine project root
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Determine config path (use same logic as check command)
	configPath := "arx.yaml"
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectRoot, configPath)
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nRun 'arx init' to generate a configuration file", configPath)
	}

	// Validate output format
	var format ports.OutputFormat
	switch auditFormat {
	case "json":
		format = ports.OutputFormatJSON
	case "terminal":
		format = ports.OutputFormatTerminal
	case "html":
		format = ports.OutputFormatHTML
	default:
		return fmt.Errorf("unsupported output format %q (use: terminal, json, html)", auditFormat)
	}

	// Create service with all dependencies wired
	service := newAuditService()

	// Load config to get max_violations threshold
	configReader := config.NewYAMLReader()
	cfg, err := configReader.Read(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if err := configReader.Validate(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Run the audit
	ctx := context.Background()
	report, err := service.Audit(ctx, projectRoot, configPath)
	if err != nil {
		return fmt.Errorf("audit failed: %w", err)
	}

	// Prepare output writer
	var out io.Writer = os.Stdout
	if auditOutput != "" {
		file, err := os.Create(auditOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file %q: %w", auditOutput, err)
		}
		defer file.Close()
		out = file
	}

	// Render the report
	if err := renderAuditReport(out, report, format, auditTrend, auditSince); err != nil {
		return fmt.Errorf("failed to render report: %w", err)
	}

	// Exit with code 1 if violations exceed threshold
	if len(report.Violations) > 0 {
		os.Exit(output.ExitCode(report.Violations, cfg.MaxViolations))
	}

	return nil
}

// newAuditService creates an AuditService with all dependencies wired
func newAuditService() *application.AuditService {
	configReader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	historyStorage := history.NewFileSystemHistory(".arx-history")

	return application.NewAuditService(configReader, detectors, historyStorage, ".arx-history")
}

// renderAuditReport outputs the audit report with support for trend flags
func renderAuditReport(out io.Writer, report *domain.AuditReport, format ports.OutputFormat, trendOnly bool, since string) error {
	// If trend-only, show only trend section
	if trendOnly {
		return renderTrendOnly(out, report, format)
	}

	// If --since is specified, we need to load historical audits
	if since != "" {
		return renderReportWithSince(out, report, format, since)
	}

	// Default: render full report
	switch format {
	case ports.OutputFormatJSON:
		return renderJSON(out, report)
	case ports.OutputFormatTerminal:
		return renderTerminal(out, report)
	default:
		return fmt.Errorf("unsupported format: %v", format)
	}
}

// renderTrendOnly outputs only the trend comparison
func renderTrendOnly(out io.Writer, report *domain.AuditReport, format ports.OutputFormat) error {
	if report.TrendReport.Status == "" {
		fmt.Fprintln(out, "No trend data available (no previous audit for comparison)")
		return nil
	}

	if format == ports.OutputFormatJSON {
		type TrendJSON struct {
			Status         string `json:"status"`
			ViolationDelta int    `json:"violation_delta"`
			DebtDelta      int    `json:"debt_delta"`
			Summary        string `json:"summary"`
		}
		data, err := json.MarshalIndent(TrendJSON{
			Status:         string(report.TrendReport.Status),
			ViolationDelta: report.TrendReport.ViolationDelta,
			DebtDelta:      report.TrendReport.DebtDelta,
			Summary:        report.TrendReport.Summary,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	}

	// Terminal output
	trendStyle := "\033[1m"
	resetStyle := "\033[0m"
	dimStyle := "\033[2m"

	var statusStyle string
	switch report.TrendReport.Status {
	case domain.TrendImproved:
		statusStyle = "\033[92m" // Green
	case domain.TrendDegraded:
		statusStyle = "\033[91m" // Red
	default:
		statusStyle = "\033[93m" // Yellow
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, trendStyle+"─ TREND COMPARISON ────────────────────────────────────────"+resetStyle)
	fmt.Fprintf(out, "  Status: %s%s%s\n", statusStyle, report.TrendReport.Status, resetStyle)
	fmt.Fprintf(out, "  Violations: %d", report.TrendReport.ViolationDelta)
	if report.TrendReport.ViolationDelta < 0 {
		fmt.Fprintln(out, " (improved)")
	} else if report.TrendReport.ViolationDelta > 0 {
		fmt.Fprintln(out, " (degraded)")
	} else {
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "  Debt: %d", report.TrendReport.DebtDelta)
	if report.TrendReport.DebtDelta < 0 {
		fmt.Fprintln(out, " (improved)")
	} else if report.TrendReport.DebtDelta > 0 {
		fmt.Fprintln(out, " (degraded)")
	} else {
		fmt.Fprintln(out)
	}
	if report.TrendReport.Summary != "" {
		fmt.Fprintf(out, "  %s%s%s\n", dimStyle, report.TrendReport.Summary, resetStyle)
	}
	fmt.Fprintln(out)

	return nil
}

// renderReportWithSince loads historical audits and shows trends since the specified date
func renderReportWithSince(out io.Writer, report *domain.AuditReport, format ports.OutputFormat, since string) error {
	// Load history
	historyStorage := history.NewFileSystemHistory(".arx-history")
	ctx := context.Background()

	dates, err := historyStorage.List(ctx)
	if err != nil {
		// If no history, just render current report
		return renderReport(out, report, format)
	}

	// Filter dates since the specified date and load audits
	var filteredAudits []*domain.AuditReport
	for _, date := range dates {
		if date.Format("2006-01-02") >= since {
			audit, err := historyStorage.Load(ctx, date)
			if err == nil && audit != nil {
				filteredAudits = append(filteredAudits, audit)
			}
		}
	}

	// If no audits found since date, render current report
	if len(filteredAudits) == 0 {
		return renderReport(out, report, format)
	}

	// Render with trend context
	if format == ports.OutputFormatJSON {
		type SinceJSON struct {
			CurrentReport  *domain.AuditReport  `json:"current"`
			TrendPeriod    string               `json:"trend_period"`
			HistoricalData []domain.AuditReport `json:"historical"`
		}
		historical := make([]domain.AuditReport, len(filteredAudits))
		for i, a := range filteredAudits {
			historical[i] = *a
		}
		data, err := json.MarshalIndent(SinceJSON{
			CurrentReport:  report,
			TrendPeriod:    since,
			HistoricalData: historical,
		}, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		_, err = fmt.Fprintln(out, string(data))
		return err
	}

	// Terminal output with since context
	boldStyle := "\033[1m"
	resetStyle := "\033[0m"

	fmt.Fprintln(out)
	fmt.Fprintln(out, boldStyle+"─ TREND PERIOD ─────────────────────────────────────────────"+resetStyle)
	fmt.Fprintf(out, "  Showing audits since: %s\n", since)
	fmt.Fprintf(out, "  Historical audits found: %d\n", len(filteredAudits))
	fmt.Fprintln(out)

	// Render current report
	return renderTerminal(out, report)
}

// renderReport outputs the audit report in the specified format
func renderReport(out io.Writer, report *domain.AuditReport, format ports.OutputFormat) error {
	switch format {
	case ports.OutputFormatJSON:
		return renderJSON(out, report)
	case ports.OutputFormatTerminal:
		return renderTerminal(out, report)
	case ports.OutputFormatHTML:
		return renderHTML(out, report)
	default:
		return fmt.Errorf("unsupported format: %v", format)
	}
}

// renderJSON outputs the audit report as JSON
func renderJSON(out io.Writer, report *domain.AuditReport) error {
	type JSONReport struct {
		Timestamp      string                `json:"timestamp"`
		ProjectRoot    string                `json:"project_root"`
		ConfigHash     string                `json:"config_hash"`
		Violations     []domain.Violation    `json:"violations"`
		CouplingMatrix domain.CouplingMatrix `json:"coupling_matrix"`
		DebtScore      domain.DebtScore      `json:"debt_score"`
		TrendReport    domain.TrendReport    `json:"trend_report,omitempty"`
	}

	jsonReport := JSONReport{
		Timestamp:      report.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		ProjectRoot:    report.ProjectRoot,
		ConfigHash:     report.ConfigHash,
		Violations:     report.Violations,
		CouplingMatrix: report.CouplingMatrix,
		DebtScore:      report.DebtScore,
		TrendReport:    report.TrendReport,
	}

	data, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	_, err = fmt.Fprintln(out, string(data))
	return err
}

// renderTerminal outputs the audit report in human-readable terminal format
func renderTerminal(out io.Writer, report *domain.AuditReport) error {
	// Styles (using same approach as terminal.go)
	headerStyle := "\033[1m"
	boldStyle := "\033[1m"
	dimStyle := "\033[2m"
	resetStyle := "\033[0m"
	errorStyle := "\033[91m"
	warningStyle := "\033[93m"
	successStyle := "\033[92m"

	// Header
	fmt.Fprintln(out)
	fmt.Fprintln(out, headerStyle+"╔═══════════════════════════════════════════════════════════╗"+resetStyle)
	fmt.Fprintln(out, headerStyle+"║              ARCHITECTURE AUDIT REPORT                    ║"+resetStyle)
	fmt.Fprintln(out, headerStyle+"╚═══════════════════════════════════════════════════════════╝"+resetStyle)
	fmt.Fprintln(out)

	// Project info
	fmt.Fprintln(out, boldStyle+"Project:"+resetStyle, report.ProjectRoot)
	fmt.Fprintln(out, boldStyle+"Config: "+resetStyle+report.ConfigHash[:16]+"...")
	fmt.Fprintln(out, boldStyle+"Timestamp:"+resetStyle, report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintln(out)

	// Violations summary
	fmt.Fprintln(out, boldStyle+"─ VIOLATIONS ──────────────────────────────────────────────"+resetStyle)
	if len(report.Violations) == 0 {
		fmt.Fprintln(out, successStyle+"✓ No violations found"+resetStyle)
	} else {
		errors := 0
		warnings := 0
		infos := 0
		for _, v := range report.Violations {
			switch v.Severity {
			case domain.SeverityWarning:
				warnings++
			case domain.SeverityInfo:
				infos++
			default:
				errors++
			}
		}

		fmt.Fprintf(out, errorStyle+"  Errors:   %d"+resetStyle+"\n", errors)
		fmt.Fprintf(out, warningStyle+"  Warnings: %d"+resetStyle+"\n", warnings)
		fmt.Fprintf(out, dimStyle+"  Info:     %d"+resetStyle+"\n", infos)
		fmt.Fprintf(out, boldStyle+"  Total:    %d"+resetStyle+"\n", len(report.Violations))

		// Show first few violations
		if len(report.Violations) > 0 {
			fmt.Fprintln(out)
			fmt.Fprintln(out, dimStyle+"Recent violations:"+resetStyle)
			maxShow := 5
			if len(report.Violations) < maxShow {
				maxShow = len(report.Violations)
			}
			for i := 0; i < maxShow; i++ {
				v := report.Violations[i]
				fmt.Fprintf(out, "  - [%s] %s:%d\n", v.RuleID, v.File, v.Line)
			}
			if len(report.Violations) > maxShow {
				fmt.Fprintf(out, dimStyle+"  ... and %d more"+resetStyle+"\n", len(report.Violations)-maxShow)
			}
		}
	}
	fmt.Fprintln(out)

	// Coupling matrix
	fmt.Fprintln(out, boldStyle+"─ COUPLING MATRIX ─────────────────────────────────────────"+resetStyle)
	couplingCount := report.CouplingMatrix.Count()
	if couplingCount == 0 {
		fmt.Fprintln(out, dimStyle+"  No inter-layer dependencies detected"+resetStyle)
	} else {
		fmt.Fprintf(out, "  Total dependencies: %d\n", couplingCount)
		fmt.Fprintln(out, dimStyle+"  From → To: Count"+resetStyle)
		for fromLayer, targets := range report.CouplingMatrix.FromTo {
			for toLayer, count := range targets {
				fmt.Fprintf(out, "  %s → %s: %d\n", fromLayer, toLayer, count)
			}
		}
	}
	fmt.Fprintln(out)

	// Debt score
	fmt.Fprintln(out, boldStyle+"─ TECHNICAL DEBT ──────────────────────────────────────────"+resetStyle)
	fmt.Fprintf(out, "  Score: %d\n", report.DebtScore.Total)
	fmt.Fprintf(out, "  Trend: %s", report.DebtScore.Trend)
	if report.DebtScore.TrendDelta != 0 {
		if report.DebtScore.TrendDelta > 0 {
			fmt.Fprintf(out, " (+%d)", report.DebtScore.TrendDelta)
		} else {
			fmt.Fprintf(out, " (%d)", report.DebtScore.TrendDelta)
		}
	}
	fmt.Fprintln(out)

	// Trend report
	if report.TrendReport.Status != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, boldStyle+"─ TRENDS ──────────────────────────────────────────────────"+resetStyle)
		fmt.Fprintf(out, "  Status: %s\n", report.TrendReport.Status)
		fmt.Fprintf(out, "  Violations: %d", report.TrendReport.ViolationDelta)
		if report.TrendReport.ViolationDelta < 0 {
			fmt.Fprintln(out, " (improved)")
		} else if report.TrendReport.ViolationDelta > 0 {
			fmt.Fprintln(out, " (degraded)")
		} else {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "  Debt: %d", report.TrendReport.DebtDelta)
		if report.TrendReport.DebtDelta < 0 {
			fmt.Fprintln(out, " (improved)")
		} else if report.TrendReport.DebtDelta > 0 {
			fmt.Fprintln(out, " (degraded)")
		} else {
			fmt.Fprintln(out)
		}
		if report.TrendReport.Summary != "" {
			fmt.Fprintf(out, "  %s\n", report.TrendReport.Summary)
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, dimStyle+"Run `arx explain <violation-id>` for detailed guidance."+resetStyle)
	fmt.Fprintln(out)

	return nil
}

// renderHTML outputs the audit report as HTML
func renderHTML(out io.Writer, report *domain.AuditReport) error {
	// Redirect stdout temporarily since HTML reporter writes to stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating pipe: %w", err)
	}
	os.Stdout = w

	reporter := output.NewHTMLReporter()
	if err := reporter.ReportAudit(report); err != nil {
		os.Stdout = oldStdout
		w.Close()
		r.Close()
		return err
	}

	w.Close()
	os.Stdout = oldStdout
	
	if _, err := io.Copy(out, r); err != nil {
		r.Close()
		return fmt.Errorf("copying HTML output: %w", err)
	}
	r.Close()

	return nil
}
