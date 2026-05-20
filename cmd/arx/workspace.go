package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// workspaceCmd represents the workspace command
var workspaceCmd = &cobra.Command{
	Use:   "workspace [path]",
	Short: "Run architecture audit across workspace projects",
	Long: `Run architecture audit across multiple projects defined in arx-workspace.yaml.
Projects are checked sequentially. Results are aggregated and reported as a table or JSON.

The workspace config is loaded from arx-workspace.yaml (or the workspace: field in arx.yaml).
Projects are discovered via glob patterns, with optional per-project overrides for
layers and rules.

Exit codes:
  0 - All projects pass (no violations found)
  1 - Any project has violations or errors

Examples:
  arx workspace                    # Discover projects at current directory
  arx workspace ./monorepo         # Discover projects at specified directory
  arx workspace --json             # JSON output to stdout
  arx workspace --verbose          # Detailed per-project breakdown
  arx workspace --output report.json  # Write JSON report to file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkspace,
}

var (
	workspaceJSON    bool
	workspaceVerbose bool
	workspaceOutput  string
)

func init() {
	workspaceCmd.Flags().BoolVarP(&workspaceJSON, "json", "j", false, "Output JSON report to stdout")
	workspaceCmd.Flags().BoolVarP(&workspaceVerbose, "verbose", "v", false, "Show detailed per-project breakdown")
	workspaceCmd.Flags().StringVarP(&workspaceOutput, "output", "o", "", "Write report to file")

	rootCmd.AddCommand(workspaceCmd)
}

func runWorkspace(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Resolve workspace root
	workspaceRoot := "."
	if len(args) > 0 {
		workspaceRoot = args[0]
	}

	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("resolving workspace path: %w", err)
	}

	// Load root-level arx.yaml config for plugin detectors
	reader := config.NewYAMLReader()
	var cfg *domain.Config
	rootConfigPath := filepath.Join(absRoot, "arx.yaml")
	if _, statErr := os.Stat(rootConfigPath); statErr == nil {
		if loadedCfg, loadErr := reader.Read(rootConfigPath); loadErr == nil {
			cfg = loadedCfg
		}
	}

	// Create service with plugin detectors if configured
	var detectors []ports.Detector
	if cfg != nil {
		detectors = detector.GetDetectorsForConfig(cfg)
	} else {
		detectors = detector.GetDetectors()
	}
	svc := application.NewWorkspaceService(detectors)

	opts := application.WorkspaceOptions{
		Verbose: workspaceVerbose,
	}

	// Load workspace config
	wc, rootPath, err := svc.LoadWorkspace(absRoot)
	if err != nil {
		return err
	}

	// Resolve projects
	projects, err := svc.ResolveProjects(wc, rootPath)
	if err != nil {
		return err
	}

	// Run workspace audit
	report, err := svc.Run(ctx, wc, projects, opts)
	if err != nil {
		return err
	}

	// Output: JSON or terminal
	if workspaceJSON || workspaceOutput != "" {
		jsonReporter := output.NewWorkspaceJSONReporter()

		if workspaceOutput != "" {
			// Write to file
			f, err := os.Create(workspaceOutput)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer f.Close()

			if err := jsonReporter.Render(report, f); err != nil {
				return err
			}
		}

		if workspaceJSON {
			if err := jsonReporter.Render(report, os.Stdout); err != nil {
				return err
			}
		}
	} else {
		termReporter := output.NewWorkspaceTerminalReporter()
		if workspaceVerbose {
			if err := termReporter.RenderVerbose(report, os.Stdout); err != nil {
				return err
			}
		} else {
			if err := termReporter.Render(report, os.Stdout); err != nil {
				return err
			}
		}
	}

	// Exit code
	if !report.Passed() {
		return fmt.Errorf("workspace: %d of %d projects FAIL",
			report.Summary.FailedProjects, report.Summary.TotalProjects)
	}

	return nil
}
