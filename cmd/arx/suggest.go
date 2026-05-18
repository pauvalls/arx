package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest [violation-id]",
	Short: "Show fix suggestions for architecture violations",
	Long: `Analyze violations and show concrete fix suggestions.

If a violation ID is provided, shows the fix for that specific violation.
Without arguments, shows fixes for all current violations.

Examples:
  arx suggest           # Show fixes for all violations
  arx suggest D-01      # Show fix for violation D-01
  arx suggest --apply   # Apply fixes with confirmation
  arx suggest --apply --force  # Apply fixes without confirmation
  arx suggest --output diff.patch  # Write diffs to file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSuggest,
}

var (
	suggestApply  bool
	suggestForce  bool
	suggestOutput string
)

func init() {
	suggestCmd.Flags().BoolVar(&suggestApply, "apply", false, "Apply the suggested fixes to files")
	suggestCmd.Flags().BoolVar(&suggestForce, "force", false, "Skip interactive confirmation when using --apply")
	suggestCmd.Flags().StringVarP(&suggestOutput, "output", "o", "", "Write diffs to file instead of stdout")
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	// Load violations from cache
	cache, err := output.LoadViolations()
	if err != nil {
		return fmt.Errorf("no violations found — run 'arx check' first: %w", err)
	}

	// Resolve target violations
	var targets []output.CachedViolation
	if len(args) > 0 {
		// Single violation by ID
		v, err := output.GetViolationByID(cache, args[0])
		if err != nil {
			return fmt.Errorf("violation %q not found — run 'arx check' to see all violations", args[0])
		}
		targets = []output.CachedViolation{*v}
	} else {
		targets = cache.Violations
	}

	if len(targets) == 0 {
		fmt.Println("✓ No violations found — nothing to suggest.")
		return nil
	}

	// Convert to domain.Violation for FixEngine
	violations := cachedToDomain(targets)

	// Generate suggestions
	engine := application.NewFixEngine()
	fixes := engine.SuggestAll(violations)

	if len(fixes) == 0 {
		fmt.Println("No fix suggestions available for current violations.")
		return nil
	}

	// Build diff output
	var diffParts []string
	for _, fix := range fixes {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Fix for %s (%s)\n", fix.ViolationID, fix.RuleID))
		sb.WriteString(fmt.Sprintf("File: %s\n", fix.File))
		sb.WriteString(fmt.Sprintf("Description: %s\n", fix.Description))
		sb.WriteString("\n")

		diff := fix.UnifiedDiff()
		if diff != "" {
			sb.WriteString(diff)
		} else {
			sb.WriteString("  (No diff available — see description above)")
		}
		sb.WriteString("\n")

		diffParts = append(diffParts, sb.String())
	}

	fullOutput := strings.Join(diffParts, "\n")
	fullOutput += fmt.Sprintf("\n%d fix suggestion(s) generated.\n", len(fixes))

	// Handle --output flag
	if suggestOutput != "" {
		if err := os.WriteFile(suggestOutput, []byte(fullOutput), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Diffs written to %s\n", suggestOutput)
		fmt.Printf("%d fix suggestion(s) generated.\n", len(fixes))
		return nil
	}

	// Print to stdout
	fmt.Print(fullOutput)

	// Handle --apply flag
	if !suggestApply {
		fmt.Println("Run 'arx suggest --apply' to apply these fixes.")
		return nil
	}

	// Interactive confirmation unless --force
	if !suggestForce {
		fmt.Print("\nApply these fixes? This will modify files. [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted. No changes made.")
			return nil
		}
	}

	// Apply fixes with backup/rollback
	backupDir := ".arx-backup"
	applied := 0
	for _, fix := range fixes {
		// Skip fixes without suggested content
		if fix.Suggested == "" && fix.Diff == "" {
			fmt.Printf("Skipping %s: no concrete fix available (generic advice only)\n", fix.ViolationID)
			continue
		}

		if err := engine.Apply(*fix, backupDir); err != nil {
			// Rollback all previously applied fixes
			for i := 0; i < applied; i++ {
				if rbErr := engine.Rollback(fixes[i].File, backupDir); rbErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: rollback failed for %s: %v\n", fixes[i].File, rbErr)
				}
			}
			return fmt.Errorf("apply failed: %w", err)
		}
		applied++
	}

	fmt.Printf("\n✓ Applied %d fix(es). Backups in %s/\n", applied, backupDir)
	return nil
}

// cachedToDomain converts cached violations to domain violations for the FixEngine.
func cachedToDomain(cached []output.CachedViolation) []domain.Violation {
	result := make([]domain.Violation, 0, len(cached))
	for _, cv := range cached {
		result = append(result, domain.Violation{
			ID:          cv.ID,
			RuleID:      cv.RuleID,
			File:        cv.File,
			Line:        cv.Line,
			SourceLayer: cv.SourceLayer,
			TargetLayer: cv.TargetLayer,
			Import:      cv.Import,
			Message:     cv.Message,
		})
	}
	return result
}

// backupDirFor returns the full path to the backup directory, creating it if needed.
func backupDirFor(projectRoot string) (string, error) {
	dir := filepath.Join(projectRoot, ".arx-backup")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create backup directory: %w", err)
	}
	return dir, nil
}
