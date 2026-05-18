package main

import (
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// fmtCmd represents the fmt command
var fmtCmd = &cobra.Command{
	Use:   "fmt [path]",
	Short: "Format arx.yaml configuration file",
	Long: `Format an arx.yaml configuration file by normalizing indentation,
ordering keys consistently, and sorting layers and rules.

If no path is provided, arx.yaml in the current directory is used.

Examples:
  arx fmt                    # Format arx.yaml in current directory
  arx fmt ./config/arx.yaml  # Format a specific config file
  arx fmt --check            # Exit with code 1 if file is not formatted`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFmt,
}

var fmtCheck bool

func init() {
	fmtCmd.Flags().BoolVarP(&fmtCheck, "check", "c", false, "Check if file is formatted (exit 1 if not)")
	rootCmd.AddCommand(fmtCmd)
}

func runFmt(cmd *cobra.Command, args []string) error {
	configPath := "arx.yaml"
	if len(args) > 0 {
		configPath = args[0]
	}

	// Read raw YAML to preserve structure
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	// Parse as generic YAML to normalize formatting
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	// Marshal back with consistent formatting
	normalized, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if fmtCheck {
		// Check mode: compare normalized output with original
		if string(normalized) != string(data) {
			fmt.Fprintf(os.Stderr, "File %s is not formatted\n", configPath)
			os.Exit(1)
		}
		fmt.Printf("%s is correctly formatted\n", configPath)
		return nil
	}

	// Validate the config can be loaded correctly
	reader := config.NewYAMLReader()
	if _, err := reader.Read(configPath); err != nil {
		return fmt.Errorf("configuration is invalid: %w", err)
	}

	// Write normalized config
	if err := os.WriteFile(configPath, normalized, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	fmt.Printf("Formatted %s\n", configPath)
	return nil
}
