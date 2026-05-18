package main

import (
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

Exit codes:
  0 - Baseline created successfully
  1 - Error occurred

Example:
  arx baseline                    # Create baseline in current directory
  arx baseline ./my-project       # Create baseline for specific directory
  arx baseline --reset            # Overwrite existing baseline`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBaseline,
}

var (
	baselineReset bool
	baselineOutput string
)

func init() {
	baselineCmd.Flags().BoolVar(&baselineReset, "reset", false, "Overwrite existing baseline")
	baselineCmd.Flags().StringVarP(&baselineOutput, "output", "o", "", "Custom output path for baseline file")
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

	fmt.Printf("Baseline created with %d violations. New violations will be reported on subsequent checks.\n", len(violations))
	fmt.Printf("Baseline saved to: %s\n", baselinePath)

	return nil
}

// newBaselineService creates a BaselineService for the CLI.
func newBaselineService() *baselineServiceWrapper {
	return &baselineServiceWrapper{
		storage: arxbaseline.NewStorage(),
	}
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
