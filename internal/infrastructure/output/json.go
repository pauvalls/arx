package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// JSONReporter implements the ports.Reporter interface for JSON output
type JSONReporter struct {
	version                 string
	schemaVersion           string
	tool                    string
	baselineSuppressedCount int
	maxViolations           int
	performance             *domain.PerformanceReport
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{
		version:       "1.0",
		schemaVersion: "1.0",
		tool:          "arx",
	}
}

// SetSchemaVersion sets the config schema version for JSON output.
func (r *JSONReporter) SetSchemaVersion(sv string) {
	r.schemaVersion = sv
}

// NewJSONReporterWithBaseline creates a JSON reporter with baseline suppression info
func NewJSONReporterWithBaseline(suppressedCount int) *JSONReporter {
	return &JSONReporter{
		version:                 "1.0",
		schemaVersion:           "1.0",
		tool:                    "arx",
		baselineSuppressedCount: suppressedCount,
	}
}

// NewJSONReporterWithThreshold creates a JSON reporter with threshold info
func NewJSONReporterWithThreshold(maxViolations int) *JSONReporter {
	return &JSONReporter{
		version:       "1.0",
		schemaVersion: "1.0",
		tool:          "arx",
		maxViolations: maxViolations,
	}
}

// SetPerformance sets an optional performance report on the JSON reporter.
// When set, it will be included in the JSON output.
func (r *JSONReporter) SetPerformance(pr *domain.PerformanceReport) {
	r.performance = pr
}

// JSONOutput represents the structured JSON output format
type JSONOutput struct {
	Version                 string                `json:"version"`
	SchemaVersion           string                `json:"schema_version"`
	Tool                    string                `json:"tool"`
	Violations              []JSONViolation       `json:"violations"`
	Summary                 Summary               `json:"summary"`
	BaselineSuppressedCount int                   `json:"baseline_suppressed_count,omitempty"`
	MaxViolations           int                   `json:"max_violations,omitempty"`
	CouplingMatrix          domain.CouplingMatrix `json:"coupling_matrix,omitempty"`
	DebtScore               domain.DebtScore      `json:"debt_score,omitempty"`
	TrendReport             domain.TrendReport    `json:"trend_report,omitempty"`
	Detectors               []DetectorInfo             `json:"detectors,omitempty"`
	Performance             *domain.PerformanceReport  `json:"performance,omitempty"`
}

// DetectorInfo mirrors application.DetectorStatus for JSON serialization
type DetectorInfo struct {
	Name       string `json:"name"`
	Applicable bool   `json:"applicable"`
	DepCount   int    `json:"dep_count"`
	Error      string `json:"error,omitempty"`
}

// JSONViolation represents a single violation in JSON format
type JSONViolation struct {
	ID           string `json:"id"`
	RuleID       string `json:"rule_id"`
	Severity     string `json:"severity"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	SourceLayer  string `json:"source_layer"`
	TargetLayer  string `json:"target_layer"`
	Import       string `json:"import"`
	Message      string `json:"message"`
	Overridden   bool   `json:"overridden,omitempty"`
}

// Summary represents the summary statistics
type Summary struct {
	Total           int `json:"total"`
	Errors          int `json:"errors"`
	Warnings        int `json:"warnings"`
	Info            int `json:"info"`
	OverriddenCount int `json:"overridden_count,omitempty"`
}

// Report outputs violations in JSON format suitable for CI/CD
func (r *JSONReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	if format != ports.OutputFormatJSON {
		return fmt.Errorf("json reporter only supports json format")
	}

	// Convert domain violations to JSON violations
	jsonViolations := make([]JSONViolation, 0, len(violations))
	errors := 0
	warnings := 0
	info := 0
	overriddenCount := 0

	for _, v := range violations {
		// Determine severity from the violation's Severity field
		var severity string
		switch v.Severity {
		case domain.SeverityWarning:
			severity = "warning"
			warnings++
		case domain.SeverityInfo:
			severity = "info"
			info++
		default:
			severity = "error"
			errors++
		}

		if v.Overridden {
			overriddenCount++
		}

		jsonViolations = append(jsonViolations, JSONViolation{
			ID:          v.ID,
			RuleID:      v.RuleID,
			Severity:    severity,
			File:        v.File,
			Line:        v.Line,
			SourceLayer: v.SourceLayer,
			TargetLayer: v.TargetLayer,
			Import:      v.Import,
			Message:     v.Message,
			Overridden:  v.Overridden,
		})
	}

	// Create output structure
	output := JSONOutput{
		Version:                 r.version,
		SchemaVersion:           r.schemaVersion,
		Tool:                    r.tool,
		Violations:              jsonViolations,
		Summary: Summary{
			Total:           len(violations),
			Errors:          errors,
			Warnings:        warnings,
			Info:            info,
			OverriddenCount: overriddenCount,
		},
		BaselineSuppressedCount: r.baselineSuppressedCount,
		MaxViolations:           r.maxViolations,
	}

	// Marshal to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	// Write to stdout
	fmt.Fprintln(os.Stdout, string(jsonData))

	return nil
}

// ReportAudit renders full audit report as JSON.
func (r *JSONReporter) ReportAudit(report *domain.AuditReport) error {
	output := r.buildJSONOutput(report.Violations)
	output.CouplingMatrix = report.CouplingMatrix
	output.DebtScore = report.DebtScore
	output.TrendReport = report.TrendReport

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(jsonData))
	return nil
}

// ReportWithContext renders check results with optional detector metadata.
func (r *JSONReporter) ReportWithContext(violations []domain.Violation, detectors []application.DetectorStatus) error {
	output := r.buildJSONOutput(violations)

	if len(detectors) > 0 {
		output.Detectors = make([]DetectorInfo, 0, len(detectors))
		for _, d := range detectors {
			output.Detectors = append(output.Detectors, DetectorInfo{
				Name:       d.Name,
				Applicable: d.Applicable,
				DepCount:   d.DepCount,
				Error:      d.Error,
			})
		}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(jsonData))
	return nil
}

// buildJSONOutput creates the base JSONOutput from violations.
func (r *JSONReporter) buildJSONOutput(violations []domain.Violation) JSONOutput {
	jsonViolations := make([]JSONViolation, 0, len(violations))
	errors := 0
	warnings := 0
	info := 0
	overriddenCount := 0

	for _, v := range violations {
		var severity string
		switch v.Severity {
		case domain.SeverityWarning:
			severity = "warning"
			warnings++
		case domain.SeverityInfo:
			severity = "info"
			info++
		default:
			severity = "error"
			errors++
		}

		if v.Overridden {
			overriddenCount++
		}

		jsonViolations = append(jsonViolations, JSONViolation{
			ID:          v.ID,
			RuleID:      v.RuleID,
			Severity:    severity,
			File:        v.File,
			Line:        v.Line,
			SourceLayer: v.SourceLayer,
			TargetLayer: v.TargetLayer,
			Import:      v.Import,
			Message:     v.Message,
			Overridden:  v.Overridden,
		})
	}

	out := JSONOutput{
		Version:       r.version,
		SchemaVersion: r.schemaVersion,
		Tool:          r.tool,
		Violations:    jsonViolations,
		Summary: Summary{
			Total:           len(violations),
			Errors:          errors,
			Warnings:        warnings,
			Info:            info,
			OverriddenCount: overriddenCount,
		},
		BaselineSuppressedCount: r.baselineSuppressedCount,
		MaxViolations:           r.maxViolations,
	}
	if r.performance != nil {
		out.Performance = r.performance
	}
	return out
}

// containsWarning checks if the rule ID suggests a warning severity
func containsWarning(ruleID string) bool {
	// Check for common warning patterns
	warningPatterns := []string{"warning", "warn"}
	for _, pattern := range warningPatterns {
		if containsIgnoreCase(ruleID, pattern) {
			return true
		}
	}
	return false
}

// containsInfo checks if the rule ID suggests an info severity
func containsInfo(ruleID string) bool {
	// Check for common info patterns
	infoPatterns := []string{"info", "suggestion"}
	for _, pattern := range infoPatterns {
		if containsIgnoreCase(ruleID, pattern) {
			return true
		}
	}
	return false
}

// Ensure JSONReporter implements ports.Reporter interface
var _ ports.Reporter = (*JSONReporter)(nil)
