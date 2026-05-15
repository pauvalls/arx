package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	arxcache "github.com/pauvalls/arx/internal/infrastructure/cache"
	arxbaseline "github.com/pauvalls/arx/internal/infrastructure/baseline"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Run architecture audit on a project",
	Long: `Run architecture audit on a project by loading the configuration,
detecting dependencies, evaluating rules, and reporting violations.

If no path is provided, the current directory is used.

The audit process:
  1. Load configuration from arx.yaml (or --config path)
  2. Run language detectors (Go, TypeScript) on the project
  3. Evaluate architectural rules against detected dependencies
  4. Generate a report with selected format

Exit codes:
  0 - No violations found (or only info/warnings with --ci)
  1 - Violations found or error occurred

When a baseline exists (.arx-baseline.json):
  0 - No NEW violations found (baseline violations are suppressed)
  1 - NEW violations found

Example:
  arx check                    # Check current directory
  arx check ./my-project       # Check specific directory
  arx check --ci               # JSON output for CI/CD
  arx check --format json      # Explicit JSON output
  arx check --verbose          # Show detailed dependency info
  arx check --no-baseline      # Ignore baseline, report all violations`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCheck,
}

var (
	checkConfig     string
	checkCI         bool
	checkFormat     string
	checkVerbose    bool
	checkNoCache    bool
	checkNoBaseline bool
)

func init() {
	checkCmd.Flags().StringVarP(&checkConfig, "config", "c", "arx.yaml", "Config file path")
	checkCmd.Flags().BoolVar(&checkCI, "ci", false, "Machine-readable JSON output for CI/CD (shorthand for --format json)")
	checkCmd.Flags().StringVarP(&checkFormat, "format", "f", "terminal", "Output format: terminal|json|sarif|md")
	checkCmd.Flags().BoolVarP(&checkVerbose, "verbose", "v", false, "Show detailed dependency information")
	checkCmd.Flags().BoolVar(&checkNoCache, "no-cache", false, "Disable the performance cache")
	checkCmd.Flags().BoolVar(&checkNoBaseline, "no-baseline", false, "Ignore baseline file and report all violations")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
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

	// Determine output format
	format := ports.OutputFormatTerminal
	if checkCI {
		format = ports.OutputFormatJSON
	} else {
		switch checkFormat {
		case "json":
			format = ports.OutputFormatJSON
		case "sarif":
			format = ports.OutputFormatSARIF
		case "md", "markdown":
			format = ports.OutputFormatMarkdown
		}
	}

	// Create service with nil cache for initial config load
	service := newCheckService(format, nil)

	// If verbose, print config info
	if checkVerbose {
		config, err := service.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Configuration: %s\n", configPath)
		fmt.Fprintf(os.Stderr, "Layers: %d\n", len(config.Layers))
		fmt.Fprintf(os.Stderr, "Rules: %d\n", len(config.Rules))
		fmt.Fprintf(os.Stderr, "Project: %s\n", projectRoot)
		fmt.Fprintln(os.Stderr)
	}

	// Run the check steps manually so we can determine exit code
	ctx := context.Background()

	config, err := service.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Compute config hash for cache invalidation
	configHash, err := config.Hash()
	if err != nil {
		return fmt.Errorf("failed to compute config hash: %w", err)
	}

	// Set up cache (unless --no-cache)
	var cache ports.Cache
	if !checkNoCache {
		cacheDir := filepath.Join(projectRoot, ".arx-cache")
		fileCache := arxcache.NewFileCache(cacheDir)
		if err := fileCache.SetConfigHash(configHash); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set cache config hash: %v\n", err)
		}
		cache = fileCache
	}

	// Re-create service with cache
	service = newCheckService(format, cache)

	dependencies, err := service.DetectCached(ctx, projectRoot, config.Layers)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	violations := service.Evaluate(dependencies, config.Rules, config.Layers)

	// Baseline filtering: load baseline and suppress known violations
	baselinePath := filepath.Join(projectRoot, application.DefaultBaselineFile)
	var baseline *domain.Baseline
	var suppressedCount int

	if !checkNoBaseline {
		baselineStorage := arxbaseline.NewStorage()
		if baselineStorage.Exists(baselinePath) {
			loaded, err := baselineStorage.Load(baselinePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load baseline: %v\n", err)
			} else if loaded != nil {
				// Check if baseline is stale (config changed)
				if loaded.IsStale(configHash) {
					fmt.Fprintf(os.Stderr, "Warning: baseline is stale (config changed). Using baseline anyway.\n")
				}

				// Count suppressed violations before filtering
				totalCount := len(violations)
				violations = loaded.Filter(violations)
				suppressedCount = totalCount - len(violations)

				if checkVerbose && suppressedCount > 0 {
					fmt.Fprintf(os.Stderr, "%d violations suppressed by baseline\n", suppressedCount)
				}
				baseline = loaded
			}
		}
	}

	// Cache violations for arx explain command (cache ALL violations, not just new ones)
	if err := output.CacheViolations(violations, projectRoot); err != nil {
		// Log warning but don't fail the check
		fmt.Fprintf(os.Stderr, "Warning: failed to cache violations: %v\n", err)
	}

	// Report violations
	// For JSON output with baseline, use a reporter that includes suppressed count
	if format == ports.OutputFormatJSON && suppressedCount > 0 {
		reporter := output.NewJSONReporterWithBaseline(suppressedCount)
		if err := reporter.Report(violations, format); err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
	} else {
		if err := service.Report(violations, format); err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
	}

	// Exit with code 1 if NEW violations found
	if len(violations) > 0 {
		os.Exit(output.ExitCode(violations))
	}

	// If baseline exists and there were suppressed violations, print summary
	if baseline != nil && suppressedCount > 0 {
		fmt.Fprintf(os.Stderr, "%d violations suppressed by baseline\n", suppressedCount)
	}

	return nil
}
