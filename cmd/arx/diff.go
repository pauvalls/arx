package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	arxbaseline "github.com/pauvalls/arx/internal/infrastructure/baseline"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff [ref-before] [ref-after]",
	Short: "Compare architecture between two git refs",
	Long: `Compare architecture violations between two git refs.

Uses git worktree to isolate each ref, runs a full architecture audit,
and compares the results using fingerprint matching.

Exit codes:
  0 - No new violations (all changes are resolved or unchanged)
  1 - New violations detected

Example:
  arx diff HEAD~1 HEAD              # Compare last commit
  arx diff main feature-branch      # Compare branch to main
  arx diff v1.0.0 v2.0.0            # Compare releases
  arx diff HEAD~1 HEAD --format json  # Machine-readable output`,
	Args: cobra.MaximumNArgs(2),
	RunE: runDiff,
}

var (
	diffFormat string
	diffConfig string
)

func init() {
	diffCmd.Flags().StringVarP(&diffFormat, "format", "f", "terminal", "Output format: terminal|json")
	diffCmd.Flags().StringVarP(&diffConfig, "config", "c", "arx.yaml", "Config file path")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Determine project root (current directory)
	projectRoot := "."
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	projectRoot = absPath

	// Determine refs (defaults: HEAD~1, HEAD)
	refBefore := "HEAD~1"
	refAfter := "HEAD"
	if len(args) >= 1 {
		refBefore = args[0]
	}
	if len(args) >= 2 {
		refAfter = args[1]
	}

	// Resolve config path
	configPath := diffConfig
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectRoot, configPath)
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nRun 'arx init' to generate a configuration file", configPath)
	}

	// Create AuditService for running audits on worktrees
	auditSvc := newAuditServiceForDiff()
	baselineSvc := application.NewBaselineService(arxbaseline.NewStorage())

	// Create DiffService
	diffSvc := application.NewDiffService(auditSvc, baselineSvc)

	// Run the diff
	ctx := context.Background()
	result, err := diffSvc.Compare(ctx, projectRoot, configPath, refBefore, refAfter)
	if err != nil {
		return err
	}

	// Render output
	renderer := output.NewDiffRenderer()
	diffData := ports.DiffResultData{
		Added:         result.Added,
		Resolved:      result.Resolved,
		Unchanged:     result.Unchanged,
		RefBefore:     result.RefBefore,
		RefAfter:      result.RefAfter,
		ConfigChanged: result.ConfigChanged,
	}
	if diffFormat == "json" {
		if err := renderer.RenderJSON(diffData); err != nil {
			return fmt.Errorf("rendering JSON output: %w", err)
		}
	} else {
		renderer.Render(diffData)
	}

	// Exit code: 1 if added violations exist
	if len(result.Added) > 0 {
		os.Exit(1)
	}

	return nil
}

// newAuditServiceForDiff creates an AuditService configured for diff worktree audits.
// History storage is nil since we don't need to persist audit history during diff.
func newAuditServiceForDiff() *application.AuditService {
	reader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	return application.NewAuditService(reader, detectors, nil, "")
}
