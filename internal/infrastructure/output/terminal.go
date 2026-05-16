package output

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// noColor disables all ANSI styling when NO_COLOR env var is set (per https://no-color.org/).
var noColor bool

func init() {
	v := os.Getenv("NO_COLOR")
	noColor = v != "" && v != "0"
}

// SetNoColor overrides the noColor flag for testing purposes.
func SetNoColor(val bool) { noColor = val }

// GetNoColor returns the current noColor state (for testing).
func GetNoColor() bool { return noColor }

// style applies a lipgloss style to text, returning plain text when noColor is true.
func style(s lipgloss.Style, text string) string {
	if noColor {
		return text
	}
	return s.Render(text)
}

// TerminalReporter implements the ports.Reporter interface for terminal output
type TerminalReporter struct {
	width         int
	maxViolations int
}

// NewTerminalReporter creates a new terminal reporter
func NewTerminalReporter() *TerminalReporter {
	return &TerminalReporter{
		width: 80,
	}
}

// NewTerminalReporterWithThreshold creates a terminal reporter with violation threshold
func NewTerminalReporterWithThreshold(maxViolations int) *TerminalReporter {
	return &TerminalReporter{
		width:         80,
		maxViolations: maxViolations,
	}
}

// Report outputs violations in human-readable terminal format with colors
func (r *TerminalReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	if format != ports.OutputFormatTerminal {
		return fmt.Errorf("terminal reporter only supports terminal format")
	}

	// Styles
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))          // Yellow
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))              // Blue
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")) // Yellow-white

	// Group violations by file for better readability
	violationsByFile := make(map[string][]domain.Violation)
	for _, v := range violations {
		violationsByFile[v.File] = append(violationsByFile[v.File], v)
	}

	// Sort files for consistent output
	files := make([]string, 0, len(violationsByFile))
	for file := range violationsByFile {
		files = append(files, file)
	}
	sort.Strings(files)

	// Output header
	fmt.Println()
	fmt.Println(style(headerStyle, "╔═══════════════════════════════════════════════════════════╗"))
	fmt.Println(style(headerStyle, "║         ARCHITECTURE VIOLATIONS DETECTED                  ║"))
	fmt.Println(style(headerStyle, "╚═══════════════════════════════════════════════════════════╝"))
	fmt.Println()

	// Output violations
	totalErrors := 0
	totalWarnings := 0
	totalInfo := 0

	for _, file := range files {
		fileViolations := violationsByFile[file]

		for _, v := range fileViolations {
			// Choose style based on severity
			var severityStyle lipgloss.Style
			var severityIcon string

			switch v.Severity {
			case domain.SeverityWarning:
				severityStyle = warningStyle
				severityIcon = "⚠️"
				totalWarnings++
			case domain.SeverityInfo:
				severityStyle = infoStyle
				severityIcon = "ℹ️"
				totalInfo++
			default:
				severityStyle = errorStyle
				severityIcon = "❌"
				totalErrors++
			}

			// Override tag for violations whose severity was changed by a rule override
			overrideTag := ""
			if v.Overridden && v.OriginalSeverity != "" {
				overrideTag = fmt.Sprintf(" (overridden from %s)", v.OriginalSeverity)
			}

			// Violation header
			fmt.Println(style(severityStyle, fmt.Sprintf("%s [%s] %s:%d%s",
				severityIcon,
				v.ID,
				v.File,
				v.Line,
				overrideTag,
			)))

			// Rule info
			fmt.Println(style(borderStyle, "   ───────────────────────────────────────────────────────"))
			fmt.Println(style(dimStyle, fmt.Sprintf("   Rule: %q → %q", v.SourceLayer, v.TargetLayer)))
			fmt.Println(style(dimStyle, fmt.Sprintf("   Import: %s", v.Import)))
			fmt.Println()

			// Explanation (Why this matters)
			fmt.Println(style(headerStyle, "   Why this matters:"))
			fmt.Println(r.wrapText(v.Message, 70, dimStyle))
			fmt.Println()

			// How to fix (if available from built-in explanations)
			fixGuidance := r.getFixGuidance(v)
			if fixGuidance != "" {
				fmt.Println(style(headerStyle, "   How to fix:"))
				fmt.Println(r.wrapText(fixGuidance, 70, dimStyle))
				fmt.Println()
			}
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(style(borderStyle, "═══════════════════════════════════════════════════════════"))

	if len(violations) == 0 {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
		fmt.Println(style(successStyle, "✓ No violations found!"))
	} else {
		summary := fmt.Sprintf("Found %d violation", len(violations))
		if len(violations) > 1 {
			summary += "s"
		}

		// Count overridden violations
		overriddenCount := 0
		for _, v := range violations {
			if v.Overridden {
				overriddenCount++
			}
		}

		if overriddenCount > 0 {
			summary += fmt.Sprintf(" (%d overridden)", overriddenCount)
		}

		// Add threshold info if set
		if r.maxViolations > 0 {
			summary += fmt.Sprintf(" (threshold: %d violations)", r.maxViolations)
		}

		details := fmt.Sprintf(" (%d errors, %d warnings, %d info)", totalErrors, totalWarnings, totalInfo)
		fmt.Println(style(errorStyle, summary+style(dimStyle, details)))

		uniqueFiles := len(violationsByFile)
		fileText := "file"
		if uniqueFiles > 1 {
			fileText = "files"
		}
		fmt.Println(style(dimStyle, fmt.Sprintf("Across %d %s", uniqueFiles, fileText)))
	}

	fmt.Println()
	fmt.Println(style(dimStyle, "Run `arx explain <violation-id>` for detailed guidance."))
	fmt.Println()

	return nil
}

// wrapText wraps text to the specified width
func (r *TerminalReporter) wrapText(text string, width int, style lipgloss.Style) string {
	words := splitWords(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Apply style and indent
	var result string
	for i, line := range lines {
		indent := "   "
		if i > 0 {
			indent = "   "
		}
		if noColor {
			result += indent + line + "\n"
		} else {
			result += indent + style.Render(line) + "\n"
		}
	}

	return result
}

// splitWords splits text into words, preserving paragraphs
func splitWords(text string) []string {
	// Simple word splitting
	var words []string
	current := ""

	for _, ch := range text {
		if ch == ' ' || ch == '\n' || ch == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
			if ch == '\n' {
				// Preserve paragraph breaks
				words = append(words, "\n")
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	// Filter out empty strings and standalone newlines used as markers
	var filtered []string
	for _, w := range words {
		if w != "" && w != "\n" {
			filtered = append(filtered, w)
		}
	}

	return filtered
}

// getFixGuidance returns fix guidance based on the violation pattern
func (r *TerminalReporter) getFixGuidance(violation domain.Violation) string {
	// Check for common patterns in the rule ID or message
	ruleID := violation.RuleID
	sourceLayer := violation.SourceLayer
	targetLayer := violation.TargetLayer

	// Domain depending on infrastructure
	if sourceLayer == "domain" && targetLayer == "infrastructure" {
		return "1. Define an interface in the domain layer (e.g., Repository, Service)\n2. Move the concrete implementation to infrastructure\n3. Inject the implementation via constructor (Dependency Inversion)"
	}

	// Domain depending on cmd
	if sourceLayer == "domain" && targetLayer == "cmd" {
		return "1. Remove the import from domain\n2. If domain needs configuration, pass it as a parameter\n3. Keep CLI concerns isolated in the cmd layer"
	}

	// Application depending on infrastructure
	if sourceLayer == "application" && targetLayer == "infrastructure" {
		return "1. Depend on abstractions (ports) instead of concrete implementations\n2. Define interfaces in the ports layer\n3. Inject implementations at the composition root (cmd layer)"
	}

	// Circular dependency patterns
	if containsIgnoreCase(ruleID, "circular") {
		return "1. Identify the shared abstraction both layers need\n2. Extract it to a separate layer (e.g., 'contracts' or 'ports')\n3. Have both layers depend on the abstraction, not each other"
	}

	// Default guidance
	return "Review the architectural boundaries between these layers. Consider using dependency inversion or extracting shared abstractions."
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure TerminalReporter implements ports.Reporter interface
var _ ports.Reporter = (*TerminalReporter)(nil)

// ExitCode returns the appropriate exit code based on violations and threshold.
// Returns 0 if no violations exist or all remaining violations are overridden.
// Returns 0 if non-overridden violations <= maxViolations (when maxViolations > 0).
// Returns 1 if non-overridden violations > maxViolations.
// maxViolations of 0 means no threshold (unlimited, backward-compatible).
func ExitCode(violations []domain.Violation, maxViolations int) int {
	if len(violations) == 0 {
		return 0
	}

	// Count non-overridden violations
	nonOverriddenCount := 0
	for _, v := range violations {
		if !v.Overridden {
			nonOverriddenCount++
		}
	}

	// If threshold is set (maxViolations > 0), check against it
	if maxViolations > 0 {
		if nonOverriddenCount <= maxViolations {
			return 0
		}
		return 1
	}

	// Backward-compatible behavior: no threshold
	// Return 0 only if ALL violations are overridden
	if nonOverriddenCount == 0 {
		return 0
	}

	return 1
}
