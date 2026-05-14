package main

import (
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/application"
	"github.com/spf13/cobra"
)

// explainCmd represents the explain command
var explainCmd = &cobra.Command{
	Use:   "explain <violation-id>",
	Short: "Show detailed explanation for a violation",
	Long: `Show detailed explanation for a specific architectural violation.

Provide a violation ID (e.g., D-01, domain-imports-infrastructure) or a
rule ID to get:
  - Context about why the rule exists
  - Why the violation matters architecturally
  - Step-by-step guidance on how to fix it

Example:
  arx explain D-01                      # Explain violation D-01
  arx explain domain-imports-infra      # Explain a rule by ID`,
	Args: cobra.ExactArgs(1),
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	ruleID := args[0]

	// Try to extract rule ID from violation ID (e.g., "D-01" -> lookup)
	// For now, we treat the input as either a rule ID or we try to find it
	lookupID := normalizeRuleID(ruleID)

	explanation := application.GetExplanation(lookupID)
	guidance := application.GetFixGuidance(lookupID)

	fmt.Println()
	fmt.Printf("Violation: %s\n", ruleID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	fmt.Println("Why this matters:")
	fmt.Printf("  %s\n", explanation)
	fmt.Println()

	if len(guidance) > 0 {
		fmt.Println("How to fix:")
		for i, step := range guidance {
			fmt.Printf("  %d. %s\n", i+1, step)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Println("Run 'arx check' to see all violations in your project.")

	return nil
}

// normalizeRuleID converts violation IDs or partial names to rule IDs
func normalizeRuleID(input string) string {
	// If it looks like a violation ID (D-01), we can't map it back to a rule
	// without the audit context, so we just use it as-is and let the
	// explanation system try to match it
	if strings.HasPrefix(input, "D-") {
		// Try common patterns - this is a best-effort lookup
		return input
	}

	// Handle common shorthand patterns
	switch input {
	case "domain-imports-infra", "domain-infra":
		return "domain-imports-infrastructure"
	case "domain-imports-app", "domain-app":
		return "domain-imports-application"
	case "app-imports-infra", "app-infra":
		return "application-imports-infrastructure"
	case "pres-imports-infra", "pres-infra":
		return "presentation-imports-infrastructure"
	case "circular", "cycle":
		return "layer-circular"
	}

	return input
}
