package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
)

// DiffRenderer renders a DiffResult to the terminal with color-coded output.
type DiffRenderer struct{}

// NewDiffRenderer creates a new DiffRenderer.
func NewDiffRenderer() *DiffRenderer {
	return &DiffRenderer{}
}

// Render outputs the diff result with color-coded terminal formatting.
func (r *DiffRenderer) Render(result application.DiffResult) {
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)     // Red
	resolvedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)    // Green
	unchangedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))              // Dim gray
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))      // Yellow-white
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Header
	fmt.Println()
	fmt.Println(headerStyle.Render(fmt.Sprintf("Architecture diff: %s → %s", result.RefBefore, result.RefAfter)))
	fmt.Println()

	if result.ConfigChanged {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render("⚠ Config changed between refs — baseline fingerprints may differ"))
		fmt.Println()
	}

	// Added violations (red)
	if len(result.Added) > 0 {
		fmt.Println(addedStyle.Render(fmt.Sprintf("+ %d NEW violations:", len(result.Added))))
		for _, v := range result.Added {
			fmt.Println(addedStyle.Render(fmt.Sprintf("  + [%s] %s:%d %s → %s (%s)",
				v.RuleID, v.File, v.Line, v.SourceLayer, v.TargetLayer, v.Import,
			)))
		}
		fmt.Println()
	}

	// Resolved violations (green)
	if len(result.Resolved) > 0 {
		fmt.Println(resolvedStyle.Render(fmt.Sprintf("- %d RESOLVED violations:", len(result.Resolved))))
		for _, v := range result.Resolved {
			fmt.Println(resolvedStyle.Render(fmt.Sprintf("  - [%s] %s:%d %s → %s (%s)",
				v.RuleID, v.File, v.Line, v.SourceLayer, v.TargetLayer, v.Import,
			)))
		}
		fmt.Println()
	}

	// Unchanged violations (dim)
	if len(result.Unchanged) > 0 {
		fmt.Println(unchangedStyle.Render(fmt.Sprintf("= %d UNCHANGED violations:", len(result.Unchanged))))
		for _, v := range result.Unchanged {
			fmt.Println(unchangedStyle.Render(fmt.Sprintf("  = [%s] %s:%d %s → %s (%s)",
				v.RuleID, v.File, v.Line, v.SourceLayer, v.TargetLayer, v.Import,
			)))
		}
		fmt.Println()
	}

	// Summary
	fmt.Println(borderStyle.Render("───────────────────────────────────────────────────────"))
	fmt.Println(headerStyle.Render(result.Summary()))

	if !result.HasChanges() {
		fmt.Println(resolvedStyle.Render("No architecture changes detected between refs."))
	} else if len(result.Added) > 0 {
		fmt.Println(addedStyle.Render("Architecture regression detected!"))
	}

	fmt.Println()
	fmt.Println(dimStyle.Render("Use --format json for machine-readable output."))
	fmt.Println()
}

// DiffJSONOutput represents the JSON output structure for diff results.
type DiffJSONOutput struct {
	RefBefore     string             `json:"ref_before"`
	RefAfter      string             `json:"ref_after"`
	ConfigChanged bool               `json:"config_changed"`
	Added         []JSONViolation    `json:"added"`
	Resolved      []JSONViolation    `json:"resolved"`
	Unchanged     []JSONViolation    `json:"unchanged"`
	Summary       string             `json:"summary"`
}

// RenderJSON outputs the diff result as JSON to stdout.
func (r *DiffRenderer) RenderJSON(result application.DiffResult) error {
	output := DiffJSONOutput{
		RefBefore:     result.RefBefore,
		RefAfter:      result.RefAfter,
		ConfigChanged: result.ConfigChanged,
		Added:         violationsToJSON(result.Added),
		Resolved:      violationsToJSON(result.Resolved),
		Unchanged:     violationsToJSON(result.Unchanged),
		Summary:       result.Summary(),
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(jsonData))
	return nil
}

// violationsToJSON converts domain violations to JSON violations.
func violationsToJSON(violations []domain.Violation) []JSONViolation {
	result := make([]JSONViolation, 0, len(violations))
	for _, v := range violations {
		result = append(result, JSONViolation{
			ID:          v.ID,
			RuleID:      v.RuleID,
			Severity:    string(v.Severity),
			File:        v.File,
			Line:        v.Line,
			SourceLayer: v.SourceLayer,
			TargetLayer: v.TargetLayer,
			Import:      v.Import,
			Message:     v.Message,
		})
	}
	return result
}
