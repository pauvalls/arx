package main

import (
	"bufio"
	"fmt"
	"io"
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
  arx suggest                     # Show fixes for all violations
  arx suggest D-01                # Show fix for violation D-01
  arx suggest --apply             # Apply fixes with confirmation
  arx suggest --all               # Collect all fixes, detect conflicts, interactive loop
  arx suggest --dry-run           # Show all fixes without applying
  arx suggest --apply --force     # Apply fixes without confirmation`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSuggest,
}

var (
	suggestApply  bool
	suggestForce  bool
	suggestOutput string
	suggestAll    bool
	suggestDryRun bool

	// suggestStdout and suggestStdin allow test overrides for I/O.
	// When nil, os.Stdout / os.Stdin are used.
	suggestStdout io.Writer
	suggestStdin  io.Reader
)

func suggestOutputWriter() io.Writer {
	if suggestStdout != nil {
		return suggestStdout
	}
	return os.Stdout
}

func suggestInputReader() io.Reader {
	if suggestStdin != nil {
		return suggestStdin
	}
	return os.Stdin
}

func init() {
	suggestCmd.Flags().BoolVar(&suggestApply, "apply", false, "Apply the suggested fixes to files")
	suggestCmd.Flags().BoolVar(&suggestForce, "force", false, "Skip interactive confirmation when using --apply")
	suggestCmd.Flags().StringVarP(&suggestOutput, "output", "o", "", "Write diffs to file instead of stdout")
	suggestCmd.Flags().BoolVar(&suggestAll, "all", false, "Collect all fixes with conflict detection and interactive review")
	suggestCmd.Flags().BoolVar(&suggestDryRun, "dry-run", false, "Show all fixes without applying (table output)")
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	out := suggestOutputWriter()
	in := suggestInputReader()

	// Load violations from cache
	cache, err := output.LoadViolations()
	if err != nil {
		return fmt.Errorf("no violations found — run 'arx check' first: %w", err)
	}

	// Resolve target violations
	var targets []output.CachedViolation
	if len(args) > 0 {
		v, err := output.GetViolationByID(cache, args[0])
		if err != nil {
			return fmt.Errorf("violation %q not found — run 'arx check' to see all violations", args[0])
		}
		targets = []output.CachedViolation{*v}
	} else {
		targets = cache.Violations
	}

	if len(targets) == 0 {
		fmt.Fprintln(out, "✓ No violations found — nothing to suggest.")
		return nil
	}

	// Convert to domain.Violation for FixEngine
	violations := cachedToDomain(targets)

	// Generate suggestions
	engine := application.NewFixEngine()
	fixes := engine.SuggestAll(violations)

	if len(fixes) == 0 {
		fmt.Fprintln(out, "No fix suggestions available for current violations.")
		return nil
	}

	// --dry-run mode: show all fixes as a table without applying
	if suggestDryRun {
		printDryRunTable(out, fixes)
		fmt.Fprintf(out, "\n%d fix suggestion(s) generated. Run 'arx suggest --apply' to apply.\n", len(fixes))
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
		fmt.Fprintf(out, "Diffs written to %s\n", suggestOutput)
		fmt.Fprintf(out, "%d fix suggestion(s) generated.\n", len(fixes))
		return nil
	}

	// Print to stdout (unless we're in --all mode which has its own display)
	if !suggestAll {
		fmt.Fprint(out, fullOutput)
	}

	// Handle --all mode: conflict detection + staged review loop
	if suggestAll {
		return runAllMode(out, in, engine, fixes)
	}

	// --apply mode
	if !suggestApply {
		if !suggestAll {
			fmt.Fprintln(out, "Run 'arx suggest --apply' to apply these fixes.")
		}
		return nil
	}

	return applyFixes(out, in, engine, fixes)
}

// runAllMode runs --all mode: conflict detection, staged review loop.
func runAllMode(out io.Writer, in io.Reader, engine *application.FixEngine, fixes []*application.Fix) error {
	// Convert to domain.FixSuggestion for conflict detection
	suggestions := make([]domain.FixSuggestion, len(fixes))
	for i, f := range fixes {
		suggestions[i] = domain.FixSuggestion{
			ViolationID: f.ViolationID,
			RuleID:      f.RuleID,
			File:        f.File,
			Line:        f.Line,
			Description: f.Description,
			Diff:        f.UnifiedDiff(),
		}
	}

	// Detect conflicts
	conflicts := application.DetectConflicts(suggestions)

	if len(conflicts) > 0 {
		fmt.Fprintf(out, "\n⚠️  %d conflict(s) detected between fixes:\n", len(conflicts))
		for _, c := range conflicts {
			fmt.Fprintf(out, "  • %s: %s ↔ %s\n", c.File, c.Suggestions[0].ViolationID, c.Suggestions[1].ViolationID)
			fmt.Fprintf(out, "    %s\n", c.Description)
		}
		if !suggestForce {
			fmt.Fprintln(out, "Conflicted fixes will be skipped. Use --force to apply anyway.")
		}
		fmt.Fprintln(out)
	}

	// Track which fixes are conflicted
	conflicted := make(map[string]bool)
	for _, c := range conflicts {
		conflicted[c.Suggestions[0].ViolationID] = true
		conflicted[c.Suggestions[1].ViolationID] = true
	}

	// Staged review loop
	fmt.Fprintln(out, "Review fixes one by one:")
	fmt.Fprintln(out, "  y = apply this fix")
	fmt.Fprintln(out, "  N = skip this fix (default)")
	fmt.Fprintln(out, "  s = show full diff")
	fmt.Fprintln(out, "  e = show explain for this violation")
	fmt.Fprintln(out, "  q = quit batch")
	fmt.Fprintln(out)

	reader := bufio.NewReader(in)
	var applied, skipped int

	for _, fix := range fixes {
		isConflicted := conflicted[fix.ViolationID]

		if isConflicted && !suggestForce {
			fmt.Fprintf(out, "⚠️  [CONFLICTED] Fix for %s (%s) — %s\n", fix.ViolationID, fix.RuleID, fix.Description)
			fmt.Fprintf(out, "  File: %s\n", fix.File)
			fmt.Fprintln(out, "  Skipping conflicted fix (use --force to apply).")
			skipped++
			continue
		}

		prompt := fmt.Sprintf("Apply fix for %s (%s)? [y/N/s/e/q] ", fix.ViolationID, fix.RuleID)
		fmt.Fprint(out, prompt)

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y":
			if fix.Suggested == "" && fix.Diff == "" {
				fmt.Fprintf(out, "  Skipping %s: no concrete fix available.\n", fix.ViolationID)
				skipped++
				continue
			}
			backupDir := ".arx-backup"
			if err := engine.Apply(*fix, backupDir); err != nil {
				fmt.Fprintf(out, "  Error applying fix for %s: %v\n", fix.ViolationID, err)
				skipped++
			} else {
				fmt.Fprintf(out, "  ✅ Applied fix for %s.\n", fix.ViolationID)
				applied++
			}
		case "s":
			diff := fix.UnifiedDiff()
			if diff != "" {
				fmt.Fprintln(out, "  ── Diff ──")
				fmt.Fprintln(out, diff)
				fmt.Fprintln(out, "  ──────────")
			} else {
				fmt.Fprintln(out, "  (No diff available)")
			}
			fmt.Fprint(out, prompt)
			response2, _ := reader.ReadString('\n')
			response2 = strings.TrimSpace(strings.ToLower(response2))
			if response2 == "y" {
				backupDir := ".arx-backup"
				if err := engine.Apply(*fix, backupDir); err != nil {
					fmt.Fprintf(out, "  Error applying fix for %s: %v\n", fix.ViolationID, err)
					skipped++
				} else {
					fmt.Fprintf(out, "  ✅ Applied fix for %s.\n", fix.ViolationID)
					applied++
				}
			} else if response2 == "q" {
				fmt.Fprintln(out, "  Quitting batch.")
				goto done
			} else {
				fmt.Fprintf(out, "  Skipped %s.\n", fix.ViolationID)
				skipped++
			}
		case "e":
			explainViolationInline(out, fix.ViolationID)
			fmt.Fprint(out, prompt)
			response2, _ := reader.ReadString('\n')
			response2 = strings.TrimSpace(strings.ToLower(response2))
			if response2 == "y" {
				backupDir := ".arx-backup"
				if err := engine.Apply(*fix, backupDir); err != nil {
					fmt.Fprintf(out, "  Error applying fix for %s: %v\n", fix.ViolationID, err)
					skipped++
				} else {
					fmt.Fprintf(out, "  ✅ Applied fix for %s.\n", fix.ViolationID)
					applied++
				}
			} else if response2 == "q" {
				fmt.Fprintln(out, "  Quitting batch.")
				goto done
			} else {
				fmt.Fprintf(out, "  Skipped %s.\n", fix.ViolationID)
				skipped++
			}
		case "q":
			fmt.Fprintln(out, "  Quitting batch.")
			goto done
		default:
			fmt.Fprintf(out, "  Skipped %s.\n", fix.ViolationID)
			skipped++
		}
	}
done:

	// Summary
	fmt.Fprintln(out)
	fmt.Fprintln(out, "┌───────────────────────────────────────┐")
	fmt.Fprintln(out, "│ BATCH APPLY SUMMARY                     │")
	fmt.Fprintln(out, "├───────────────────────────────────────┤")
	fmt.Fprintf(out, "│ Applied:   %2d                            │\n", applied)
	fmt.Fprintf(out, "│ Skipped:   %2d                            │\n", skipped)
	fmt.Fprintln(out, "└───────────────────────────────────────┘")

	printRollbackInstructions(out)
	return nil
}

// applyFixes applies all fixes with a single confirmation (original --apply behavior).
func applyFixes(out io.Writer, in io.Reader, engine *application.FixEngine, fixes []*application.Fix) error {
	// Interactive confirmation unless --force
	if !suggestForce {
		fmt.Fprint(out, "\nApply these fixes? This will modify files. [y/N] ")
		reader := bufio.NewReader(in)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(out, "Aborted. No changes made.")
			return nil
		}
	}

	// Apply fixes with backup/rollback
	backupDir := ".arx-backup"
	applied := 0
	for _, fix := range fixes {
		if fix.Suggested == "" && fix.Diff == "" {
			fmt.Fprintf(out, "Skipping %s: no concrete fix available (generic advice only)\n", fix.ViolationID)
			continue
		}

		if err := engine.Apply(*fix, backupDir); err != nil {
			for i := 0; i < applied; i++ {
				if rbErr := engine.Rollback(fixes[i].File, backupDir); rbErr != nil {
					fmt.Fprintf(out, "Warning: rollback failed for %s: %v\n", fixes[i].File, rbErr)
				}
			}
			return fmt.Errorf("apply failed: %w", err)
		}
		applied++
	}

	fmt.Fprintf(out, "\n✓ Applied %d fix(es). Backups in %s/\n", applied, backupDir)
	printRollbackInstructions(out)
	return nil
}

// printRollbackInstructions prints instructions for undoing changes.
func printRollbackInstructions(out io.Writer) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To undo changes:")
	fmt.Fprintln(out, "  arx rollback <file>        Restore a single file")
	fmt.Fprintln(out, "  arx rollback --list        Show available backups")
	fmt.Fprintln(out, "  arx rollback --all         Restore everything")
}

// printDryRunTable prints a table of fixes for --dry-run mode.
func printDryRunTable(out io.Writer, fixes []*application.Fix) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "┌─────────────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│ DRY RUN — No files will be modified                              │")
	fmt.Fprintln(out, "├─────────────────────────────────────────────────────────────────┤")
	fmt.Fprintf(out, "│ %-5s │ %-30s │ %-8s │\n", "ID", "Description", "File")
	fmt.Fprintln(out, "├─────────────────────────────────────────────────────────────────┤")

	for _, fix := range fixes {
		desc := fix.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Fprintf(out, "│ %-5s │ %-30s │ %-8s │\n", fix.ViolationID, desc, fix.File)
	}

	fmt.Fprintln(out, "└─────────────────────────────────────────────────────────────────┘")
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

// explainViolationInline shows an explanation for a violation ID (used in staged review).
func explainViolationInline(out io.Writer, violationID string) {
	cache, err := output.LoadViolations()
	if err != nil {
		fmt.Fprintln(out, "  (Could not load violation details)")
		return
	}
	v, err := output.GetViolationByID(cache, violationID)
	if err != nil {
		fmt.Fprintf(out, "  (Violation %q not found)\n", violationID)
		return
	}
	fmt.Fprintln(out, "  ── Explanation ──")
	fmt.Fprintf(out, "  %s\n", v.Message)
	fmt.Fprintln(out, "  ────────────────")
}

// backupDirFor returns the full path to the backup directory, creating it if needed.
func backupDirFor(projectRoot string) (string, error) {
	dir := filepath.Join(projectRoot, ".arx-backup")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create backup directory: %w", err)
	}
	return dir, nil
}
