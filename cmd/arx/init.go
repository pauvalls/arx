package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize Arx configuration for a project",
	Long: `Initialize Arx configuration for a project by scanning the directory
structure, detecting languages, and generating an arx.yaml file with
sensible defaults.

If no path is provided, the current directory is used.

The generated configuration includes:
  - Detected layers based on directory structure
  - Default architectural rules (domain cannot depend on infrastructure, etc.)
  - Language-specific overrides for Go and TypeScript
  - Common exclude patterns (vendor, node_modules, etc.)

Example:
  arx init                    # Initialize in current directory
  arx init ./my-project       # Initialize in specific directory
  arx init --output config/arx.yaml  # Write to custom path`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initOutput string
	initForce  bool
	initPreset string
)

func init() {
	initCmd.Flags().StringVarP(&initOutput, "output", "o", "arx.yaml", "Output file path for the generated configuration")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing configuration file")
	initCmd.Flags().StringVarP(&initPreset, "preset", "p", "", "Use preset template (clean, hexagonal, ddd)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
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

	// Check if project exists
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", projectRoot)
	}

	// Check if output file already exists
	outputPath := initOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(projectRoot, outputPath)
	}

	// Create service and run init
	service := newInitService()
	
	var config *domain.Config
	var initErr error
	if initPreset != "" {
		// Validate preset name
		validPresets := []string{"clean", "hexagonal", "ddd"}
		isValid := false
		for _, p := range validPresets {
			if p == initPreset {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("unknown preset %q, available presets: %s", initPreset, strings.Join(validPresets, ", "))
		}
		
		config, initErr = service.InitWithPreset(initPreset, outputPath, initForce)
		if initErr != nil {
			return fmt.Errorf("failed to initialize with preset: %w", initErr)
		}
	} else {
		config, initErr = service.Init(projectRoot, outputPath)
		if initErr != nil {
			return fmt.Errorf("failed to initialize: %w", initErr)
		}
	}

	// Print success message
	fmt.Printf("✓ Written to %s\n", outputPath)
	fmt.Printf("  Detected %d layer(s): ", len(config.Layers))
	for i, layer := range config.Layers {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(layer.Name)
	}
	fmt.Println()
	fmt.Printf("  Generated %d rule(s)\n", len(config.Rules))
	fmt.Println()
	fmt.Println("Review and adjust the configuration before running 'arx check'.")

	return nil
}
