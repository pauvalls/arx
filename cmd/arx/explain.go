package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// explainCmd represents the explain command
var explainCmd = &cobra.Command{
	Use:   "explain <violation-id>",
	Short: "Show detailed explanation for a specific violation",
	Long: `Show detailed explanation for a specific architectural violation.

Provide a violation ID (e.g., D-01) to get comprehensive guidance:
  - Full violation context (file, line, import)
  - Why this specific violation matters architecturally
  - Step-by-step refactoring guidance with code examples
  - Related violations (same rule or same file)

If no violation ID is provided, shows the most recent violation from cache.

Examples:
  arx explain D-01                      # Explain specific violation
  arx explain                           # Show most recent violation
  arx explain --last                    # Alias for most recent
  arx explain --list                    # List all cached violations`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExplain,
}

var (
	explainList    bool
	explainLast    bool
	explainSuggest string

	// explainStdout allows test override for output.
	explainStdout io.Writer
)

func explainOutputWriter() io.Writer {
	if explainStdout != nil {
		return explainStdout
	}
	return os.Stdout
}

func init() {
	explainCmd.Flags().BoolVar(&explainList, "list", false, "List all cached violations")
	explainCmd.Flags().BoolVar(&explainLast, "last", false, "Show most recent violation")
	explainCmd.Flags().StringVar(&explainSuggest, "suggest", "", "Show fix suggestion for a specific rule")
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	out := explainOutputWriter()

	// Handle --suggest flag: show fix suggestion for a rule
	if explainSuggest != "" {
		return showSuggestForRule(out, explainSuggest)
	}

	// Handle --list flag
	if explainList {
		return listViolations(out)
	}

	// Load violations from cache
	cache, err := output.LoadViolations()
	if err != nil {
		return fmt.Errorf("no cached violations found\nRun 'arx check' to generate violations cache")
	}

	// Handle --last flag or no arguments
	if explainLast || len(args) == 0 {
		if len(cache.Violations) == 0 {
			fmt.Fprintln(out, "✓ No violations in cache - your architecture is clean!")
			return nil
		}
		// Show the first (most recent) violation
		return explainViolation(out, cache.Violations[0])
	}

	// Lookup specific violation by ID
	violationID := args[0]
	violation, err := output.GetViolationByID(cache, violationID)
	if err != nil {
		return fmt.Errorf("violation %q not found\nRun 'arx check' to see all violations", violationID)
	}

	return explainViolation(out, *violation)
}

// showSuggestForRule shows the fix suggestion for a rule ID.
func showSuggestForRule(out io.Writer, ruleID string) error {
	cache, err := output.LoadViolations()
	if err != nil {
		return fmt.Errorf("no cached violations found\nRun 'arx check' to generate violations cache")
	}

	// Find the first violation matching the rule ID (or the rule itself if no violations match)
	var target *output.CachedViolation
	for _, v := range cache.Violations {
		if v.RuleID == ruleID {
			target = &v
			break
		}
	}

	if target == nil && len(cache.Violations) > 0 {
		target = &cache.Violations[0]
	}

	if target == nil {
		fmt.Fprintln(out, "No violations found.")
		return nil
	}

	fixEngine := application.NewFixEngine()
	fix := fixEngine.SuggestFix(domain.Violation{
		ID:          target.ID,
		RuleID:      target.RuleID,
		File:        target.File,
		Line:        target.Line,
		SourceLayer: target.SourceLayer,
		TargetLayer: target.TargetLayer,
		Import:      target.Import,
		Severity:    domain.Severity(target.Severity),
	})

	if fix == nil || fix.Diff == "" {
		fmt.Fprintf(out, "No fix suggestion available for rule %q.\n", ruleID)
		return nil
	}

	fmt.Fprintf(out, "Fix suggestion for rule %s:\n", ruleID)
	fmt.Fprintf(out, "  File: %s\n", fix.File)
	fmt.Fprintf(out, "  Description: %s\n", fix.Description)
	fmt.Fprintln(out)
	fmt.Fprintln(out, fix.Diff)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Run 'arx suggest %s' to apply.\n", target.ID)
	return nil
}

func listViolations(out io.Writer) error {
	cache, err := output.LoadViolations()
	if err != nil {
		return err
	}

	if len(cache.Violations) == 0 {
		fmt.Fprintln(out, "✓ No violations in cache - your architecture is clean!")
		return nil
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Cached Violations (%d total)\n", len(cache.Violations))
	fmt.Fprintln(out, strings.Repeat("─", 70))
	fmt.Fprintln(out)

	for _, v := range cache.Violations {
		severityIcon := "❌"
		if strings.ToLower(v.Severity) == "warning" {
			severityIcon = "⚠️"
		}

		fmt.Fprintf(out, "%s %s\n", severityIcon, v.ID)
		fmt.Fprintf(out, "   File: %s:%d\n", v.File, v.Line)
		fmt.Fprintf(out, "   Rule: %s\n", v.RuleID)
		fmt.Fprintf(out, "   Severity: %s\n", cases.Title(language.Und).String(strings.ToLower(v.Severity)))
		fmt.Fprintln(out)
	}

	fmt.Fprintf(out, "Run 'arx explain <id>' for detailed guidance on a specific violation.\n")
	fmt.Fprintln(out)

	return nil
}

func explainViolation(out io.Writer, v output.CachedViolation) error {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "╔══════════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "║              ARCHITECTURE VIOLATION EXPLAINED                    ║")
	fmt.Fprintln(out, "╚══════════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(out)

	// Violation summary
	fmt.Fprintf(out, "Violation: %s\n", v.ID)
	fmt.Fprintf(out, "Severity:   %s\n", formatSeverity(v.Severity))
	fmt.Fprintf(out, "File:       %s:%d\n", v.File, v.Line)
	fmt.Fprintf(out, "Rule:       %s\n", v.RuleID)
	fmt.Fprintf(out, "Import:     %s\n", v.Import)
	fmt.Fprintln(out)

	// Code context (read actual file)
	showCodeContext(out, v.File, v.Line)

	// Why it matters
	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ WHY THIS MATTERS                                                 │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	fmt.Fprintln(out, wrapText(v.Message, 70))
	fmt.Fprintln(out)

	// Architectural context
	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ ARCHITECTURAL CONTEXT                                            │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	fmt.Fprintln(out, explainArchitecturalContext(v.SourceLayer, v.TargetLayer))
	fmt.Fprintln(out)

	// How to fix with code examples
	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ HOW TO FIX                                                       │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	fixGuidance := getDetailedFixGuidance(v.RuleID, v.SourceLayer, v.TargetLayer)
	for i, step := range fixGuidance {
		fmt.Fprintf(out, "%d. %s\n", i+1, step)
	}
	fmt.Fprintln(out)

	// Code example
	codeExample := getCodeExample(v.RuleID, v.SourceLayer, v.TargetLayer, v.Import)
	if codeExample != "" {
		fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
		fmt.Fprintln(out, "│ CODE EXAMPLE                                                     │")
		fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
		fmt.Fprintln(out)
		fmt.Fprintln(out, codeExample)
		fmt.Fprintln(out)
	}

	// Fix suggestion from suggest engine
	fixEngine := application.NewFixEngine()
	fix := fixEngine.SuggestFix(domain.Violation{
		ID:          v.ID,
		RuleID:      v.RuleID,
		File:        v.File,
		Line:        v.Line,
		SourceLayer: v.SourceLayer,
		TargetLayer: v.TargetLayer,
		Import:      v.Import,
		Severity:    domain.Severity(v.Severity),
	})
	if fix != nil && fix.Diff != "" {
		fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
		fmt.Fprintln(out, "│ AUTO-FIX SUGGESTION                                              │")
		fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  Run 'arx suggest %s' to auto-apply this fix.\n", v.ID)
		fmt.Fprintln(out)
		fmt.Fprintln(out, fix.Diff)
		fmt.Fprintln(out)
	}

	// Related violations
	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ RELATED VIOLATIONS                                               │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	relatedViolations := findRelatedViolations(v)
	if len(relatedViolations) > 0 {
		for _, rv := range relatedViolations {
			if rv.ID != v.ID {
				fmt.Fprintf(out, "  • %s: %s:%d (%s)\n", rv.ID, rv.File, rv.Line, rv.RuleID)
			}
		}
	} else {
		fmt.Fprintln(out, "  No related violations found.")
	}
	fmt.Fprintln(out)

	// Next steps
	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ NEXT STEPS                                                       │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  1. Review the code example above")
	fmt.Fprintln(out, "  2. Apply the refactoring steps to your code")
	fmt.Fprintln(out, "  3. Run 'arx check' again to verify the violation is resolved")
	fmt.Fprintln(out, "  4. Commit the fix with a descriptive message")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run 'arx check' to see all violations in your project.")
	fmt.Fprintln(out)

	return nil
}

func formatSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "❌ Error"
	case "warning":
		return "⚠️  Warning"
	case "info":
		return "ℹ️  Info"
	default:
		return severity
	}
}

func explainArchitecturalContext(sourceLayer, targetLayer string) string {
	contexts := map[string]string{
		"domain-infrastructure": `The Dependency Inversion Principle (SOLID-D) states that high-level 
modules (domain) should not depend on low-level modules (infrastructure). 
Both should depend on abstractions.

When domain depends on infrastructure:
  • Business logic becomes coupled to implementation details
  • Changing databases requires modifying business rules
  • Testing domain logic requires real infrastructure
  • The stability of the system is compromised`,

		"domain-application": `The domain layer should be the most stable layer in your architecture.
It contains enterprise business rules that rarely change.

When domain depends on application:
  • Use case changes can break business rules
  • Domain becomes less stable and harder to test
  • Circular dependencies may form between layers`,

		"application-infrastructure": `The application layer should depend on abstractions (ports/interfaces),
not concrete infrastructure implementations.

When application depends on infrastructure:
  • Dependency Inversion Principle is violated
  • Swapping infrastructure requires changing application code
  • Testing application logic becomes harder`,

		"circular": `Circular dependencies create tightly coupled code that is:
  • Hard to understand and modify
  • Difficult to test in isolation
  • Prone to breaking when changes are made
  • Impossible to deploy incrementally`,
	}

	key := fmt.Sprintf("%s-%s", sourceLayer, targetLayer)
	if ctx, ok := contexts[key]; ok {
		return ctx
	}

	if strings.Contains(key, "circular") {
		return contexts["circular"]
	}

	return `This dependency violates the architectural rules defined in your arx.yaml.
Review your layer boundaries and ensure dependencies flow in the correct
direction according to your chosen architecture (Clean, Hexagonal, DDD).`
}

func getDetailedFixGuidance(ruleID, sourceLayer, targetLayer string) []string {
	guidance := map[string][]string{
		"domain-infrastructure": {
			"Identify the infrastructure concern being imported (database, HTTP client, etc.)",
			"Define an interface in the domain layer that captures the needed behavior",
			"Move the concrete implementation to the infrastructure layer",
			"Inject the implementation via constructor (Dependency Injection)",
			"Update tests to use mocks based on the domain interface",
		},
		"domain-application": {
			"Identify why domain needs application functionality",
			"Extract the needed interface to the domain layer",
			"Keep application-specific logic in the application layer",
			"Use events or callbacks if domain needs to trigger application behavior",
		},
		"application-infrastructure": {
			"Identify the infrastructure dependency",
			"Define a port (interface) that the application needs",
			"Implement the port in the infrastructure layer",
			"Wire up the implementation at the composition root (main.go, wire.go)",
		},
		"circular": {
			"Draw your dependency graph to visualize the cycle",
			"Identify the weakest link in the cycle",
			"Extract shared abstractions to a new layer or package",
			"Use dependency injection to break the cycle",
			"Consider event-driven architecture for loose coupling",
		},
	}

	key := fmt.Sprintf("%s-%s", sourceLayer, targetLayer)
	if g, ok := guidance[key]; ok {
		return g
	}

	if strings.Contains(key, "circular") {
		return guidance["circular"]
	}

	return []string{
		"Review the rule definition in arx.yaml to understand the intent",
		"Trace the dependency path to find where the rule is violated",
		"Refactor to ensure dependencies flow in the correct direction",
		"Run 'arx check' again to verify the fix",
	}
}

func getCodeExample(ruleID, sourceLayer, targetLayer, importPath string) string {
	if sourceLayer == "domain" && targetLayer == "infrastructure" {
		return `BEFORE (Violation):
  // internal/domain/order.go
  package domain
  
  import "github.com/example/app/internal/infrastructure/postgres"
  
  type Order struct {
      repo *postgres.OrderRepository  // ❌ Domain depends on infrastructure
  }

AFTER (Fixed):
  // internal/domain/order.go
  package domain
  
  type OrderRepository interface {  // ✅ Interface in domain
      Save(order *Order) error
      FindByID(id string) (*Order, error)
  }
  
  type Order struct {
      repo OrderRepository  // ✅ Domain depends on abstraction
  }

  // internal/infrastructure/postgres/order_repo.go
  package postgres
  
  type OrderRepository struct {  // ✅ Implementation in infrastructure
      db *sql.DB
  }
  
  func (r *OrderRepository) Save(order *Order) error {
      // PostgreSQL implementation
  }`
	}

	if sourceLayer == "application" && targetLayer == "infrastructure" {
		return `BEFORE (Violation):
  // internal/application/order_service.go
  package application
  
  import "github.com/example/app/internal/infrastructure/postgres"
  
  type OrderService struct {
      repo *postgres.OrderRepository  // ❌ Application depends on concrete infra
  }

AFTER (Fixed):
  // internal/application/order_service.go
  package application
  
  import "github.com/example/app/internal/domain"
  
  type OrderService struct {
      repo domain.OrderRepository  // ✅ Application depends on domain abstraction
  }`
	}

	return ""
}

func findRelatedViolations(v output.CachedViolation) []output.CachedViolation {
	cache, err := output.LoadViolations()
	if err != nil {
		return nil
	}

	var related []output.CachedViolation
	for _, cv := range cache.Violations {
		// Same rule or same file
		if cv.RuleID == v.RuleID || cv.File == v.File {
			related = append(related, cv)
		}
	}

	return related
}

// showCodeContext reads the file and displays lines around the violation.
// It tries common project roots if the file path is relative.
func showCodeContext(out io.Writer, filePath string, line int) {
	if filePath == "" || line <= 0 {
		return
	}

	// Try to find the file (it might be relative to current dir or project root)
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		// Try relative to current directory
		if cwd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(cwd, filePath)
			if _, statErr := os.Stat(candidate); statErr == nil {
				absPath = candidate
			}
		}
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return // Can't read file, skip context
	}

	lines := strings.Split(string(content), "\n")
	if line > len(lines) {
		return
	}

	// Show 3 lines before and 3 after the violation
	start := line - 4
	if start < 0 {
		start = 0
	}
	end := line + 3
	if end > len(lines) {
		end = len(lines)
	}

	fmt.Fprintln(out, "┌──────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ CODE CONTEXT                                                     │")
	fmt.Fprintln(out, "└──────────────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)

	for i := start; i < end; i++ {
		marker := " "
		if i+1 == line {
			marker = "→" // marks the violation line
		}
		fmt.Fprintf(out, "  %4d %s %s\n", i+1, marker, lines[i])
	}
	fmt.Fprintln(out)
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	words := strings.Fields(strings.TrimSpace(text))
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

	return strings.Join(lines, "\n")
}
