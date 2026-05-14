package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// JSONReporter implements the ports.Reporter interface for JSON output
type JSONReporter struct {
	version string
	tool    string
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{
		version: "1.0",
		tool:    "arx",
	}
}

// JSONOutput represents the structured JSON output format
type JSONOutput struct {
	Version   string             `json:"version"`
	Tool      string             `json:"tool"`
	Violations []JSONViolation  `json:"violations"`
	Summary   Summary            `json:"summary"`
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
}

// Summary represents the summary statistics
type Summary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
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

	for _, v := range violations {
		// Determine severity
		severity := "error"
		if v.Message != "" {
			// Try to infer severity from the violation
			// In the future, Violation should have a Severity field
			if containsWarning(v.RuleID) {
				severity = "warning"
				warnings++
			} else if containsInfo(v.RuleID) {
				severity = "info"
				info++
			} else {
				errors++
			}
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
		})
	}

	// Create output structure
	output := JSONOutput{
		Version:    r.version,
		Tool:       r.tool,
		Violations: jsonViolations,
		Summary: Summary{
			Total:    len(violations),
			Errors:   errors,
			Warnings: warnings,
			Info:     info,
		},
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
