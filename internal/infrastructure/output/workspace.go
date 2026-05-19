package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/charmbracelet/lipgloss"
)

// WorkspaceTerminalReporter renders a workspace report as a per-project terminal table.
type WorkspaceTerminalReporter struct{}

// NewWorkspaceTerminalReporter creates a new WorkspaceTerminalReporter.
func NewWorkspaceTerminalReporter() *WorkspaceTerminalReporter {
	return &WorkspaceTerminalReporter{}
}

// Render outputs a per-project table to the given writer.
func (r *WorkspaceTerminalReporter) Render(report *domain.WorkspaceReport, w io.Writer) error {
	return r.render(report, w, false)
}

// RenderVerbose outputs a detailed per-project breakdown including violations.
func (r *WorkspaceTerminalReporter) RenderVerbose(report *domain.WorkspaceReport, w io.Writer) error {
	return r.render(report, w, true)
}

func (r *WorkspaceTerminalReporter) render(report *domain.WorkspaceReport, w io.Writer, verbose bool) error {
	if len(report.Projects) == 0 {
		fmt.Fprintln(w, style(dimStyle, "No projects configured in workspace."))
		fmt.Fprintln(w)
		return nil
	}

	// Styles
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)  // Green
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)    // Red
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))             // Yellow
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Header
	fmt.Fprintln(w, style(headerBoldStyle, fmt.Sprintf("%-20s  %-8s  %6s  %8s  %4s  %5s",
		"Project", "Status", "Errors", "Warnings", "Info", "Total")))
	fmt.Fprintln(w, style(borderStyle, strings.Repeat("─", 60)))

	// Per-project rows
	totalErrors := 0
	totalWarnings := 0
	totalInfo := 0
	totalViolations := 0
	failCount := 0

	for _, p := range report.Projects {
		statusText := "PASS"
		statusStyle := successStyle
		if p.Status == "fail" || p.Error != "" {
			statusText = "FAIL"
			statusStyle = errorStyle
			failCount++
		}

		errs := p.Summary.Errors
		warns := p.Summary.Warnings
		infos := p.Summary.Info
		total := p.Summary.Total

		// If project has an error, show it in errors column
		if p.Error != "" && errs == 0 {
			errs = 1
			total = 1
		}

		fmt.Fprintf(w, "%-20s  %s  %6d  %8d  %4d  %5d\n",
			truncateString(p.Name, 20),
			style(statusStyle, statusText),
			errs, warns, infos, total)

		totalErrors += errs
		totalWarnings += warns
		totalInfo += infos
		totalViolations += total

		// Verbose: show violations per project
		if verbose && len(p.Violations) > 0 {
			for _, v := range p.Violations {
				sevStyle := errorStyle
				sevLabel := "ERROR"
				switch v.Severity {
				case domain.SeverityWarning:
					sevStyle = warningStyle
					sevLabel = "WARN"
				case domain.SeverityInfo:
					sevStyle = dimStyle
					sevLabel = "INFO"
				}
				fmt.Fprintf(w, "  %s [%s] %s:%d (%s)\n",
					style(sevStyle, sevLabel),
					v.RuleID,
					v.File, v.Line,
					style(dimStyle, v.Message))
			}
		}
	}

	// Summary line
	fmt.Fprintln(w, style(borderStyle, strings.Repeat("─", 60)))
	if failCount > 0 {
		fmt.Fprintf(w, "%s  %s\n",
			style(errorStyle, fmt.Sprintf("%d of %d projects FAIL", failCount, len(report.Projects))),
			style(dimStyle, fmt.Sprintf("Total violations: %d  (%d errors, %d warnings, %d info)",
				totalViolations, totalErrors, totalWarnings, totalInfo)))
	} else {
		fmt.Fprintf(w, "%s  %s\n",
			style(successStyle, fmt.Sprintf("All %d projects PASS", len(report.Projects))),
			style(dimStyle, fmt.Sprintf("Total violations: %d", totalViolations)))
	}

	fmt.Fprintln(w)
	return nil
}

// headerBoldStyle is a bold style for table headers.
var headerBoldStyle = lipgloss.NewStyle().Bold(true)

// dimStyle is used for less prominent text.
var dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// WorkspaceJSONReporter renders a workspace report as structured JSON.
type WorkspaceJSONReporter struct{}

// NewWorkspaceJSONReporter creates a new WorkspaceJSONReporter.
func NewWorkspaceJSONReporter() *WorkspaceJSONReporter {
	return &WorkspaceJSONReporter{}
}

// Render outputs the workspace report as indented JSON to the given writer.
func (r *WorkspaceJSONReporter) Render(report *domain.WorkspaceReport, w io.Writer) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling workspace report to JSON: %w", err)
	}

	_, err = fmt.Fprintln(w, string(jsonData))
	return err
}

// truncateString truncates a string to the given max length, appending "…" if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
