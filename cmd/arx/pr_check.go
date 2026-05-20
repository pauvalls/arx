package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/tabwriter"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/spf13/cobra"
)

// prCheckCmd represents the pr-check command
var prCheckCmd = &cobra.Command{
	Use:   "pr-check",
	Short: "Run architecture check on pull request changes",
	Long: `Run an architecture check scoped to the changes introduced by a pull request.

Uses git to get the diff between two refs, runs a full architecture check,
and filters violations to only those on lines changed by the PR.

Exit codes:
  0 - No new violations introduced
  1 - New violations found

Example:
  arx pr-check --base HEAD~1 --head HEAD
  arx pr-check --base origin/main --head feature/branch --json
  arx pr-check --base main --head feature --repo /path/to/repo`,
	Args: cobra.NoArgs,
	RunE: runPRCheck,
}

var (
	prCheckBase    string
	prCheckHead    string
	prCheckRepo    string
	prCheckJSON    bool
	prCheckVerbose bool
	prCheckApprove bool
)

func init() {
	prCheckCmd.Flags().StringVar(&prCheckBase, "base", "", "Base ref (e.g., main, HEAD~1)")
	prCheckCmd.Flags().StringVar(&prCheckHead, "head", "", "Head ref (e.g., feature/branch, HEAD)")
	prCheckCmd.Flags().StringVarP(&prCheckRepo, "repo", "r", ".", "Project root path")
	prCheckCmd.Flags().BoolVarP(&prCheckJSON, "json", "j", false, "Output in JSON format")
	prCheckCmd.Flags().BoolVarP(&prCheckVerbose, "verbose", "v", false, "Show detailed information")
	prCheckCmd.Flags().BoolVar(&prCheckApprove, "approve", false, "Auto-approve PR via GitHub API when no violations")
	rootCmd.AddCommand(prCheckCmd)
}

// violationOutput is the JSON output format for a PR-check violation.
type violationOutput struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	RuleID    string `json:"rule_id"`
	Message   string `json:"message"`
	Severity  string `json:"severity,omitempty"`
	IsNew     bool   `json:"is_new"`
}

func runPRCheck(cmd *cobra.Command, args []string) error {
	if prCheckBase == "" {
		return fmt.Errorf("--base flag is required")
	}
	if prCheckHead == "" {
		return fmt.Errorf("--head flag is required")
	}

	// Resolve repo path
	repoPath := prCheckRepo
	if !filepath.IsAbs(repoPath) {
		absPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("invalid repo path: %w", err)
		}
		repoPath = absPath
	}

	// Verify it's a git repo
	if err := verifyGitRepo(repoPath); err != nil {
		return err
	}

	// Get diff
	diffOutput, err := application.GetGitDiff(repoPath, prCheckBase, prCheckHead)
	if err != nil {
		// Try diff-tree as fallback
		var diffErr error
		diffOutput, diffErr = application.GetGitDiffTree(repoPath, prCheckBase, prCheckHead)
		if diffErr != nil {
			return fmt.Errorf("getting git diff: %w (diff: %v)", err, diffErr)
		}
	}

	if prCheckVerbose {
		fmt.Fprintf(os.Stderr, "Diff size: %d bytes\n", len(diffOutput))
	}

	// Parse diff
	diffSummary, err := application.ParseDiff(diffOutput)
	if err != nil {
		return fmt.Errorf("parsing diff: %w", err)
	}

	if prCheckVerbose {
		if diffSummary.Stats != nil {
			fmt.Fprintf(os.Stderr, "Files changed: %d\n", diffSummary.Stats["files"])
			fmt.Fprintf(os.Stderr, "Insertions: %d\n", diffSummary.Stats["insertions"])
			fmt.Fprintf(os.Stderr, "Deletions: %d\n", diffSummary.Stats["deletions"])
		}
		fmt.Fprintf(os.Stderr, "Diff hunks: %d\n", len(diffSummary.Hunks))
	}

	// Create check service
	service := newPRCheckService()

	// Load config
	configPath := filepath.Join(repoPath, "arx.yaml")
	cfg, err := service.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Run check
	ctx := context.Background()
	dependencies, err := service.Detect(ctx, repoPath, cfg.Layers)
	if err != nil {
		return fmt.Errorf("detecting dependencies: %w", err)
	}

	allViolations := service.Evaluate(dependencies, cfg.Rules, cfg.Layers)

	if prCheckVerbose {
		fmt.Fprintf(os.Stderr, "Total violations in repo: %d\n", len(allViolations))
	}

	// Filter violations to PR-introduced only
	newViolations := application.FilterViolationsForDiff(allViolations, diffSummary)

	// Format and output
	if prCheckJSON {
		outputJSON(newViolations)
	} else {
		outputTable(newViolations, prCheckVerbose)
	}

	// Auto-approve if flag is set and no violations
	if prCheckApprove && len(newViolations) == 0 {
		fmt.Fprintln(os.Stderr, "✅ No violations — auto-approve would be triggered (requires GitHub API configuration)")
	}

	// Exit code
	if len(newViolations) > 0 {
		os.Exit(1)
	}

	return nil
}

// newPRCheckService creates a CheckService for PR check operations.
func newPRCheckService() *application.CheckService {
	reader := config.NewYAMLReader()
	d := detector.GetDetectors()
	reporter := output.NewTerminalReporter()
	return application.NewCheckService(reader, d, reporter)
}

// outputTable prints violations in a table format.
func outputTable(violations []domain.Violation, verbose bool) {
	if len(violations) == 0 {
		fmt.Println("✅ No new architecture violations in this PR.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "FILE\tLINE\tVIOLATION\tNEW")

	for _, v := range violations {
		fmt.Fprintf(w, "%s\t%d\t[%s] %s\t%s\n",
			v.File,
			v.Line,
			v.RuleID,
			v.Message,
			"Yes",
		)
	}
	w.Flush()

	fmt.Fprintf(os.Stderr, "\n❌ %d new architecture violation(s) found.\n", len(violations))
}

// outputJSON prints violations as JSON.
func outputJSON(violations []domain.Violation) {
	output := make([]violationOutput, len(violations))
	for i, v := range violations {
		output[i] = violationOutput{
			File:     v.File,
			Line:     v.Line,
			RuleID:   v.RuleID,
			Message:  v.Message,
			Severity: string(v.Severity),
			IsNew:    true,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to encode JSON: %v\n", err)
	}
}

// verifyGitRepo checks that the path is a git repository.
func verifyGitRepo(repoPath string) error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s is not a git repository: %w\n%s", repoPath, err, string(output))
	}
	return nil
}

// gitCommand creates a git exec.Command in the given directory.
func gitCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}
