package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	arxcache "github.com/pauvalls/arx/internal/infrastructure/cache"
	arxbaseline "github.com/pauvalls/arx/internal/infrastructure/baseline"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/infrastructure/watcher"
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

Watch mode (--watch):
  Runs an initial check, then monitors file changes and re-runs automatically.
  Changes are debounced (default 500ms) to avoid rapid re-runs.
  Press Ctrl+C to stop.

Example:
  arx check                    # Check current directory
  arx check ./my-project       # Check specific directory
  arx check --ci               # JSON output for CI/CD
  arx check --format json      # Explicit JSON output
  arx check --format html      # HTML report for browsers
  arx check --verbose          # Show detailed dependency info
  arx check --no-baseline      # Ignore baseline, report all violations
  arx check --watch            # Watch mode: re-run on file changes
  arx check --watch --interval 1s  # Watch mode with 1s debounce`,
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
	checkWatch      bool
	checkInterval   time.Duration
	checkSeverity   string
)

func init() {
	checkCmd.Flags().StringVarP(&checkConfig, "config", "c", "arx.yaml", "Config file path")
	checkCmd.Flags().BoolVar(&checkCI, "ci", false, "Machine-readable JSON output for CI/CD (shorthand for --format json)")
	checkCmd.Flags().StringVarP(&checkFormat, "format", "f", "terminal", "Output format: terminal|json|sarif|md|junit|annotations|html")
	checkCmd.Flags().BoolVarP(&checkVerbose, "verbose", "v", false, "Show detailed dependency information")
	checkCmd.Flags().BoolVar(&checkNoCache, "no-cache", false, "Disable the performance cache")
	checkCmd.Flags().BoolVar(&checkNoBaseline, "no-baseline", false, "Ignore baseline file and report all violations")
	checkCmd.Flags().BoolVar(&checkWatch, "watch", false, "Watch mode: re-run check on file changes")
	checkCmd.Flags().DurationVar(&checkInterval, "interval", 500*time.Millisecond, "Debounce interval for watch mode")
	checkCmd.Flags().StringVar(&checkSeverity, "severity", "", "Filter by severity: error|warning|info")
	rootCmd.AddCommand(checkCmd)
}

// checkResult holds the output of a single check run.
type checkResult struct {
	violations      []domain.Violation
	suppressedCount int
	config          *domain.Config
	configHash      string
	projectRoot     string
	format          ports.OutputFormat
	duration        time.Duration
	detectorStatuses []application.DetectorStatus
}

// runCheck is the entry point for the check command.
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
		case "junit":
			format = ports.OutputFormatJUnit
		case "annotations":
			format = ports.OutputFormatGitHubAnnotations
		case "html":
			format = ports.OutputFormatHTML
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

	// Load config
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

	// Run initial check
	result := runCheckWithService(service, config, configHash, projectRoot, format, checkVerbose, checkNoBaseline, cache)

	// Report initial result
	printCheckResult(result, format, false)

	// In single-shot mode, exit based on violations
	if !checkWatch {
		if len(result.violations) > 0 {
			os.Exit(output.ExitCode(result.violations, result.config.MaxViolations))
		}
		// Print baseline summary if applicable
		if result.suppressedCount > 0 {
			fmt.Fprintf(os.Stderr, "%d violations suppressed by baseline\n", result.suppressedCount)
		}
		return nil
	}

	// --- Watch mode ---
	return watchMode(service, config, configHash, projectRoot, format, result)
}

// runCheckWithService performs a single check run using the provided service.
func runCheckWithService(service *application.CheckService, config *domain.Config, configHash, projectRoot string,
	format ports.OutputFormat, verbose, noBaseline bool, cache ports.Cache) checkResult {

	start := time.Now()
	ctx := context.Background()

	// Re-create service with cache if needed
	if cache != nil {
		service = newCheckService(format, cache)
	}

	var dependencies []domain.Dependency
	var detectorStatuses []application.DetectorStatus

	if verbose {
		result, err := service.DetectCachedWithStatus(ctx, projectRoot, config.Layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: detection failed: %v\n", err)
			// Still return whatever statuses we collected
			if result != nil {
				detectorStatuses = result.Statuses
			}
			return checkResult{projectRoot: projectRoot, format: format, detectorStatuses: detectorStatuses}
		}
		dependencies = result.Dependencies
		detectorStatuses = result.Statuses
	} else {
		var err error
		dependencies, err = service.DetectCached(ctx, projectRoot, config.Layers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: detection failed: %v\n", err)
			return checkResult{projectRoot: projectRoot, format: format}
		}
	}

	violations := service.Evaluate(dependencies, config.Rules, config.Layers)

	// Baseline filtering
	baselinePath := filepath.Join(projectRoot, application.DefaultBaselineFile)
	var suppressedCount int

	if !noBaseline {
		baselineStorage := arxbaseline.NewStorage()
		if baselineStorage.Exists(baselinePath) {
			loaded, err := baselineStorage.Load(baselinePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load baseline: %v\n", err)
			} else if loaded != nil {
				if loaded.IsStale(configHash) {
					fmt.Fprintf(os.Stderr, "Warning: baseline is stale (config changed). Using baseline anyway.\n")
				}
				totalCount := len(violations)
				violations = loaded.Filter(violations)
				suppressedCount = totalCount - len(violations)

				if verbose && suppressedCount > 0 {
					fmt.Fprintf(os.Stderr, "%d violations suppressed by baseline\n", suppressedCount)
				}
			}
		}
	}

	// Cache violations for arx explain command
	if err := output.CacheViolations(violations, projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache violations: %v\n", err)
	}

	return checkResult{
		violations:      violations,
		suppressedCount: suppressedCount,
		config:          config,
		configHash:      configHash,
		projectRoot:     projectRoot,
		format:          format,
		duration:        time.Since(start),
		detectorStatuses: detectorStatuses,
	}
}

// printCheckResult outputs the violations to the terminal or as JSON.
func printCheckResult(result checkResult, format ports.OutputFormat, isWatchUpdate bool) {
	violations := result.violations
	suppressedCount := result.suppressedCount

	// Filter by severity if flag is set
	if checkSeverity != "" {
		var filtered []domain.Violation
		for _, v := range violations {
			if string(v.Severity) == checkSeverity {
				filtered = append(filtered, v)
			}
		}
		violations = filtered
	}

	if isWatchUpdate {
		// Watch mode updates are printed to stderr
		if format == ports.OutputFormatJSON && len(violations) > 0 {
			// Skip full report for watch JSON — handled by watch loop
		} else if format == ports.OutputFormatJSON {
			// No violations: output empty array
			json.NewEncoder(os.Stdout).Encode(violations)
		}
		return
	}

	// Print detector status in verbose mode
	printDetectorStatuses(result.detectorStatuses)

	// Initial check report
	if format == ports.OutputFormatJSON {
		var reporter *output.JSONReporter
		if suppressedCount > 0 {
			// Use baseline-aware reporter
			reporter = output.NewJSONReporterWithBaseline(suppressedCount)
		} else if result.config.MaxViolations > 0 {
			// Use threshold-aware reporter
			reporter = output.NewJSONReporterWithThreshold(result.config.MaxViolations)
		} else {
			reporter = output.NewJSONReporter()
		}
		if err := reporter.ReportWithContext(violations, result.detectorStatuses); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: report generation failed: %v\n", err)
		}
	} else if format == ports.OutputFormatTerminal {
		// Use threshold-aware terminal reporter
		reporter := output.NewTerminalReporterWithThreshold(result.config.MaxViolations)
		if err := reporter.Report(violations, format); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: report generation failed: %v\n", err)
		}
	} else {
		// Other formats don't support threshold display yet
		service := newCheckService(format, nil)
		if err := service.Report(violations, format); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: report generation failed: %v\n", err)
		}
	}
}

// watchMode runs the watch loop: monitors file changes and re-runs checks.
func watchMode(service *application.CheckService, config *domain.Config, configHash, projectRoot string,
	format ports.OutputFormat, initialResult checkResult) error {

	dirs := []string{projectRoot}
	w, err := watcher.NewWatcher(dirs, checkInterval)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown via signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down watch mode...")
		cancel()
	}()

	// Keep track of previous violations for diff
	prevViolations := initialResult.violations

	// Print initial watch status
	if checkVerbose {
		fmt.Fprintf(os.Stderr, "Watching %s for changes (debounce: %s)...\n", projectRoot, checkInterval)
	}

	// Start watcher
	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}()

	// Event loop
	for {
		select {
		case evt := <-w.Events():
			// In verbose mode, print each file change event
			if checkVerbose {
				fmt.Fprintf(os.Stderr, "Change detected: %s (%s)\n", evt.Path, opToString(evt.Op))
			}

			// Re-run check
			result := runCheckWithService(service, config, configHash, projectRoot, format, checkVerbose, checkNoBaseline, nil)

			// Diff with previous run
			watchResult := domain.DiffViolations(prevViolations, result.violations)
			watchResult.Elapsed = result.duration

			// Print diff summary
			if format == ports.OutputFormatJSON {
				outputJSONWatchResult(watchResult)
			} else {
				printWatchResultSummary(watchResult)
			}

			// Update previous violations
			prevViolations = result.violations

		case err := <-w.Errors():
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)

		case <-ctx.Done():
			return nil
		}
	}
}

// printWatchResultSummary prints the watch result summary to stderr.
func printWatchResultSummary(r domain.WatchResult) {
	summary := r.Summary()

	// Color code: red for added, green for resolved
	var prefix string
	if len(r.Added) > 0 && len(r.Resolved) > 0 {
		prefix = "📋 "
	} else if len(r.Added) > 0 {
		prefix = "🔴 "
	} else if len(r.Resolved) > 0 {
		prefix = "🟢 "
	} else {
		prefix = "  "
	}

	fmt.Fprintf(os.Stderr, "%s%s\n", prefix, summary)
}

// outputJSONWatchResult outputs the watch result as JSON.
func outputJSONWatchResult(r domain.WatchResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to encode watch result: %v\n", err)
	}
}

// opToString returns a human-readable string for an operation type.
func opToString(op watcher.Op) string {
	switch op {
	case watcher.Create:
		return "created"
	case watcher.Write:
		return "modified"
	case watcher.Remove:
		return "deleted"
	case watcher.Rename:
		return "renamed"
	case watcher.Chmod:
		return "permissions changed"
	default:
		return "unknown"
	}
}

// printDetectorStatus outputs per-detector status in verbose mode.
// Only prints if there are statuses to show. Output goes to stderr.
func printDetectorStatuses(statuses []application.DetectorStatus) {
	if len(statuses) == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, "Detectors:")
	for _, s := range statuses {
		if s.Error != "" {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %s\n", capitalize(s.Name), s.Error)
		} else if s.Applicable {
			fmt.Fprintf(os.Stderr, "  ✓ %s: %d dependencies\n", capitalize(s.Name), s.DepCount)
		} else {
			fmt.Fprintf(os.Stderr, "  ✗ %s: no project files found\n", capitalize(s.Name))
		}
	}
	fmt.Fprintln(os.Stderr)
}

// capitalize returns the string with the first letter uppercased.
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
