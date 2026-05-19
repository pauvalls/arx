package ruletest

import (
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/ruletest"
)

// TableReporter formats test results as a human-readable table
type TableReporter struct{}

// NewTableReporter creates a new TableReporter
func NewTableReporter() *TableReporter {
	return &TableReporter{}
}

// Report formats test results as a table. If verbose is true, all results
// including passing tests are shown. Otherwise only failures are shown.
func (r *TableReporter) Report(results []ruletest.CaseResult, verbose bool) string {
	if len(results) == 0 {
		return "0/0 tests passed"
	}

	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "%-40s %-12s %s\n", "TEST", "STATUS", "DETAILS")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 80))

	var passed, total int
	for _, cr := range results {
		total++
		if cr.Passed {
			passed++
			if !verbose && passed != total {
				// Skip individual passes in non-verbose mode if there are failures
				// but show passing test name if all pass or verbose
				continue
			}
		}

		status := "PASS"
		if !cr.Passed {
			status = "FAIL"
		}

		detail := cr.Details
		if cr.Passed && verbose {
			detail = cr.Details
		} else if cr.Passed {
			detail = ""
		}

		fmt.Fprintf(&b, "%-40s %-12s %s\n", truncate(cr.Name, 38), status, detail)
	}

	// Summary
	fmt.Fprintf(&b, "\n%d/%d tests passed\n", passed, total)

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
