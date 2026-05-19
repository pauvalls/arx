package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	arxbaseline "github.com/pauvalls/arx/internal/infrastructure/baseline"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// baselineCmd represents the baseline command
var baselineCmd = &cobra.Command{
	Use:   "baseline [path]",
	Short: "Create a baseline of current violations",
	Long: `Create a baseline of current architecture violations.

Running 'arx baseline' captures all current violations and writes them
to .arx-baseline.json. On subsequent 'arx check' runs, violations that
match the baseline are suppressed — only NEW violations are reported.

The baseline fingerprint uses rule_id + source_layer + target_layer + import,
so file reorganizations (which change paths and line numbers) do not cause
previously baselined violations to reappear.

Use --diff to compare current violations against the latest history snapshot.
Use --history to show a trend table of baseline snapshots over time.

Exit codes:
  0 - Baseline created successfully
  1 - Error occurred

Example:
  arx baseline                    # Create baseline in current directory
  arx baseline ./my-project       # Create baseline for specific directory
  arx baseline --reset            # Overwrite existing baseline
  arx baseline --diff             # Show diff since last snapshot
  arx baseline --history          # Show baseline history trend`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBaseline,
}

var (
	baselineReset           bool
	baselineOutput          string
	baselineDiff            bool
	baselineHistory         bool
	baselineRefreshThreshold int
)

func init() {
	baselineCmd.Flags().BoolVar(&baselineReset, "reset", false, "Overwrite existing baseline")
	baselineCmd.Flags().StringVarP(&baselineOutput, "output", "o", "", "Custom output path for baseline file")
	baselineCmd.Flags().BoolVar(&baselineDiff, "diff", false, "Show violations added/resolved since last snapshot")
	baselineCmd.Flags().BoolVar(&baselineHistory, "history", false, "Show baseline history trend table")
	baselineCmd.Flags().IntVar(&baselineRefreshThreshold, "refresh-threshold", 3, "Number of consecutive clean checks before auto-refresh")
	rootCmd.AddCommand(baselineCmd)
}

func runBaseline(cmd *cobra.Command, args []string) error {
	// Determine project root
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Handle display-only modes
	if baselineDiff {
		return runBaselineDiff(projectRoot)
	}
	if baselineHistory {
		return runBaselineHistory(projectRoot)
	}

	// Determine config path
	configPath := checkConfig
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectRoot, configPath)
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nRun 'arx init' to generate a configuration file", configPath)
	}

	// Create service
	service := newCheckService(ports.OutputFormatTerminal, nil)

	// Load config
	config, err := service.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Compute config hash
	configHash, err := config.Hash()
	if err != nil {
		return fmt.Errorf("failed to compute config hash: %w", err)
	}

	// Run detection
	ctx := cmd.Context()
	dependencies, err := service.DetectCached(ctx, projectRoot, config.Layers)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Evaluate rules
	violations := service.Evaluate(dependencies, config.Rules, config.Layers)

	// Generate baseline
	baselineSvc := newBaselineService()
	baseline := baselineSvc.Generate(violations, configHash)

	// Determine output path
	baselinePath := baselineOutput
	if baselinePath == "" {
		baselinePath = baselineSvc.DefaultPath(projectRoot)
	}

	// Check if baseline already exists
	if !baselineReset && baselineSvc.Exists(baselinePath) {
		fmt.Fprintf(os.Stderr, "Warning: baseline already exists at %s\n", baselinePath)
		fmt.Fprintf(os.Stderr, "Use --reset to overwrite, or delete the existing file.\n")
		return fmt.Errorf("baseline already exists")
	}

	// Save baseline
	if err := baselineSvc.Save(baseline, baselinePath); err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	// Also save a snapshot to history for future --diff comparisons
	extSvc := newExtendedBaselineService()
	if snapshot, snapErr := extSvc.Snapshot(projectRoot); snapErr == nil && snapshot != nil {
		// Snapshot saved successfully (best effort)
		_ = snapshot
	}

	fmt.Printf("Baseline created with %d violations. New violations will be reported on subsequent checks.\n", len(violations))
	fmt.Printf("Baseline saved to: %s\n", baselinePath)

	return nil
}

// runBaselineDiff shows violations added/resolved since the last snapshot.
func runBaselineDiff(projectRoot string) error {
	extSvc := newExtendedBaselineService()

	snapshot, err := extSvc.LatestSnapshot(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load latest snapshot: %w", err)
	}
	if snapshot == nil {
		fmt.Println("No baseline snapshots found. Run 'arx baseline' first to create one.")
		return nil
	}

	// Run current check to get violations
	configPath := checkConfig
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectRoot, configPath)
	}

	service := newCheckService(ports.OutputFormatTerminal, nil)
	config, err := service.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	dependencies, err := service.DetectCached(context.Background(), projectRoot, config.Layers)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	currentViolations := service.Evaluate(dependencies, config.Rules, config.Layers)

	added, resolved, err := extSvc.DiffFromSnapshot(*snapshot, currentViolations)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	// Format output
	snapshotTime := snapshot.CreatedAt.Format("2006-01-02 15:04:05")
	renderDiffOutput(snapshotTime, added, resolved)

	return nil
}

// runBaselineHistory shows the trend table of baseline snapshots.
func runBaselineHistory(projectRoot string) error {
	extSvc := newExtendedBaselineService()

	trend, err := extSvc.Trend(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load trend: %w", err)
	}

	if len(trend) == 0 {
		fmt.Println("No baseline history available. Run 'arx baseline' to create the first snapshot.")
		return nil
	}

	// Limit to 20 entries
	maxEntries := 20
	if len(trend) > maxEntries {
		trend = trend[:maxEntries]
	}

	renderHistoryOutput(trend)

	return nil
}

// renderDiffOutput prints the diff table to stdout.
func renderDiffOutput(snapshotTime string, added, resolved []domain.Violation) {
	if len(added) == 0 && len(resolved) == 0 {
		fmt.Println("No changes since last snapshot.")
		return
	}

	fmt.Printf("BASELINE DIFF (since %s)\n", snapshotTime)
	fmt.Printf("  Added:    %d violation", len(added))
	if len(added) != 1 {
		fmt.Print("s")
	}
	fmt.Println()
	for _, v := range added {
		fmt.Printf("    %s  %s  %s → %s\n", v.ID, v.File, v.SourceLayer, v.TargetLayer)
	}
	fmt.Printf("  Resolved: %d violation", len(resolved))
	if len(resolved) != 1 {
		fmt.Print("s")
	}
	fmt.Println()
	for _, v := range resolved {
		if v.Message != "" {
			fmt.Printf("    %s  (%s)\n", v.ID, v.Message)
		} else {
			fmt.Printf("    %s  (%s → %s resolved)\n", v.ID, v.SourceLayer, v.TargetLayer)
		}
	}
}

// renderHistoryOutput prints the history trend table to stdout.
func renderHistoryOutput(trend []domain.TrendPoint) {
	if len(trend) == 0 {
		fmt.Println("No baseline history available. Run 'arx baseline' to create the first snapshot.")
		return
	}

	fmt.Println("BASELINE HISTORY")
	fmt.Println("Date          Total   Errors   Warnings   Info")
	fmt.Println("───────────────────────────────────────────────")
	for _, tp := range trend {
		fmt.Printf("%-14s %5d   %6d   %9d   %4d\n",
			tp.Date.Format("2006-01-02"),
			tp.Total, tp.Errors, tp.Warnings, tp.Info)
	}
}

// newBaselineService creates a BaselineService for the CLI.
func newBaselineService() *baselineServiceWrapper {
	return &baselineServiceWrapper{
		storage: arxbaseline.NewStorage(),
	}
}

// newExtendedBaselineService creates a BaselineService with all dependencies for the CLI.
func newExtendedBaselineService() *application.BaselineService {
	return application.NewBaselineServiceFull(
		arxbaseline.NewStorage(),
		arxbaseline.NewHistoryStorage(),
		arxbaseline.NewTrackStorage(),
	)
}

// baselineServiceWrapper wraps the application BaselineService for CLI use.
type baselineServiceWrapper struct {
	storage *arxbaseline.Storage
}

func (w *baselineServiceWrapper) Generate(violations []domain.Violation, configHash string) *domain.Baseline {
	return domain.GenerateBaseline(violations, configHash)
}

func (w *baselineServiceWrapper) Load(path string) (*domain.Baseline, error) {
	return w.storage.Load(path)
}

func (w *baselineServiceWrapper) Save(b *domain.Baseline, path string) error {
	return w.storage.Save(b, path)
}

func (w *baselineServiceWrapper) Exists(path string) bool {
	return w.storage.Exists(path)
}

func (w *baselineServiceWrapper) DefaultPath(projectRoot string) string {
	return filepath.Join(projectRoot, application.DefaultBaselineFile)
}


