package main

import (
	"fmt"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor [path]",
	Short: "Run diagnostics on arx project",
	Long: `Run diagnostics to check the health of your arx project.

Checks:
  1. Project root exists and is accessible
  2. Config file (arx.yaml) exists and is valid
  3. Language detectors can find source files
  4. Git repository status (if applicable)
  5. Arx version information

Exit codes:
  0 - All critical checks passed
  1 - One or more critical checks failed

Examples:
  arx doctor                    # Check current directory
  arx doctor ./my-project       # Check specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
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

	// Create doctor service
	service := application.NewDoctorService(VersionString(), detector.GetDetectors())

	// Run diagnostics
	result := service.Check(projectRoot)

	// Print results with icons
	printDoctorCheckResult(cmd, "Project root", result.ProjectRoot)
	printDoctorCheckResult(cmd, "Config file", result.ConfigFile)
	printDoctorCheckResult(cmd, "Detectors", result.Detectors)
	printDoctorCheckResult(cmd, "Git status", result.GitStatus)
	printDoctorCheckResult(cmd, "Version", result.Version)

	// Print summary
	fmt.Fprintln(cmd.OutOrStdout())
	if result.AllChecksPassed {
		fmt.Fprintln(cmd.OutOrStdout(), "✅ All critical checks passed")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "❌ Some checks failed")
	return fmt.Errorf("one or more critical checks failed")
}

// printDoctorCheckResult prints a single check result with appropriate icon
func printDoctorCheckResult(cmd *cobra.Command, name string, result application.CheckResult) {
	icon := "✅"
	if !result.OK {
		icon = "❌"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %s\n", icon, name, result.Message)
}
