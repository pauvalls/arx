package main

import (
	"fmt"
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
	explainList bool
	explainLast bool
)

func init() {
	explainCmd.Flags().BoolVar(&explainList, "list", false, "List all cached violations")
	explainCmd.Flags().BoolVar(&explainLast, "last", false, "Show most recent violation")
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if explainList {
		return listViolations()
	}

	// Load violations from cache
	cache, err := output.LoadViolations()
	if err != nil {
		return fmt.Errorf("no cached violations found\nRun 'arx check' to generate violations cache")
	}

	// Handle --last flag or no arguments
	if explainLast || len(args) == 0 {
		if len(cache.Violations) == 0 {
			fmt.Println("✓ No violations in cache - your architecture is clean!")
			return nil
		}
		// Show the first (most recent) violation
		return explainViolation(cache.Violations[0])
	}

	// Lookup specific violation by ID
	violationID := args[0]
	violation, err := output.GetViolationByID(cache, violationID)
	if err != nil {
		return fmt.Errorf("violation %q not found\nRun 'arx check' to see all violations", violationID)
	}

	return explainViolation(*violation)
}

func listViolations() error {
	cache, err := output.LoadViolations()
	if err != nil {
		return err
	}

	if len(cache.Violations) == 0 {
		fmt.Println("✓ No violations in cache - your architecture is clean!")
		return nil
	}

	fmt.Println()
	fmt.Printf("Cached Violations (%d total)\n", len(cache.Violations))
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println()

	for _, v := range cache.Violations {
		severityIcon := "❌"
		if strings.ToLower(v.Severity) == "warning" {
			severityIcon = "⚠️"
		}

		fmt.Printf("%s %s\n", severityIcon, v.ID)
		fmt.Printf("   File: %s:%d\n", v.File, v.Line)
		fmt.Printf("   Rule: %s\n", v.RuleID)
		fmt.Printf("   Severity: %s\n", cases.Title(language.Und).String(strings.ToLower(v.Severity)))
		fmt.Println()
	}

	fmt.Printf("Run 'arx explain <id>' for detailed guidance on a specific violation.\n")
	fmt.Println()

	return nil
}

func explainViolation(v output.CachedViolation) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              ARCHITECTURE VIOLATION EXPLAINED                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Violation summary
	fmt.Printf("Violation: %s\n", v.ID)
	fmt.Printf("Severity:   %s\n", formatSeverity(v.Severity))
	fmt.Printf("File:       %s:%d\n", v.File, v.Line)
	fmt.Printf("Rule:       %s\n", v.RuleID)
	fmt.Printf("Import:     %s\n", v.Import)
	fmt.Println()

	// Code context (read actual file)
	showCodeContext(v.File, v.Line)

	// Why it matters
	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ WHY THIS MATTERS                                                 │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println(wrapText(v.Message, 70))
	fmt.Println()

	// Architectural context
	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ ARCHITECTURAL CONTEXT                                            │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println(explainArchitecturalContext(v.SourceLayer, v.TargetLayer))
	fmt.Println()

	// How to fix with code examples
	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ HOW TO FIX                                                       │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fixGuidance := getDetailedFixGuidance(v.RuleID, v.SourceLayer, v.TargetLayer)
	for i, step := range fixGuidance {
		fmt.Printf("%d. %s\n", i+1, step)
	}
	fmt.Println()

	// Code example
	codeExample := getCodeExample(v.RuleID, v.SourceLayer, v.TargetLayer, v.Import)
	if codeExample != "" {
		fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ CODE EXAMPLE                                                     │")
		fmt.Println("└──────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println(codeExample)
		fmt.Println()
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
		fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ AUTO-FIX SUGGESTION                                              │")
		fmt.Println("└──────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		fmt.Printf("  Run 'arx suggest %s' to auto-apply this fix.\n", v.ID)
		fmt.Println()
		fmt.Println(fix.Diff)
		fmt.Println()
	}

	// Related violations
	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ RELATED VIOLATIONS                                               │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	relatedViolations := findRelatedViolations(v)
	if len(relatedViolations) > 0 {
		for _, rv := range relatedViolations {
			if rv.ID != v.ID {
				fmt.Printf("  • %s: %s:%d (%s)\n", rv.ID, rv.File, rv.Line, rv.RuleID)
			}
		}
	} else {
		fmt.Println("  No related violations found.")
	}
	fmt.Println()

	// Next steps
	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ NEXT STEPS                                                       │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  1. Review the code example above")
	fmt.Println("  2. Apply the refactoring steps to your code")
	fmt.Println("  3. Run 'arx check' again to verify the violation is resolved")
	fmt.Println("  4. Commit the fix with a descriptive message")
	fmt.Println()
	fmt.Println("Run 'arx check' to see all violations in your project.")
	fmt.Println()

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
func showCodeContext(filePath string, line int) {
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

	fmt.Println("┌──────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ CODE CONTEXT                                                     │")
	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	for i := start; i < end; i++ {
		marker := " "
		if i+1 == line {
			marker = "→" // marks the violation line
		}
		fmt.Printf("  %4d %s %s\n", i+1, marker, lines[i])
	}
	fmt.Println()
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
