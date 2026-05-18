package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/spf13/cobra"
)

var (
	diagramFormat string
	diagramOutput string
)

// diagramCmd represents the diagram command
var diagramCmd = &cobra.Command{
	Use:   "diagram [path]",
	Short: "Generate architecture dependency diagram",
	Long: `Generate an architecture dependency diagram for a project.

The diagram command analyzes your project's dependencies and generates
a visual representation showing how different layers depend on each other.

Supported output formats:
  - ascii: ASCII art diagram (default)
  - dot: Graphviz DOT format for visualization tools
  - mermaid: Mermaid flowchart syntax for markdown

If no path is provided, the current directory is used.

Examples:
  arx diagram                           # ASCII diagram to stdout
  arx diagram --format mermaid          # Mermaid diagram to stdout
  arx diagram --format dot -o deps.dot  # DOT diagram to file
  arx diagram --format ascii -o diagram.txt
  arx diagram ./my-project --format mermaid`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDiagram,
}

func init() {
	diagramCmd.Flags().StringVarP(&diagramFormat, "format", "f", "ascii", "Output format: ascii|dot|mermaid")
	diagramCmd.Flags().StringVarP(&diagramOutput, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.AddCommand(diagramCmd)
}

func runDiagram(cmd *cobra.Command, args []string) error {
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

	// Validate format
	validFormats := map[string]bool{
		"ascii":   true,
		"dot":     true,
		"mermaid": true,
	}
	if !validFormats[diagramFormat] {
		return fmt.Errorf("invalid format %q: must be one of ascii, dot, mermaid", diagramFormat)
	}

	// Determine config path
	configPath := filepath.Join(projectRoot, "arx.yaml")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\nRun 'arx init' to generate a configuration file", configPath)
	}

	// Load config
	reader := config.NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := reader.Validate(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create diagram service
	detectors := detector.GetDetectors()
	diagramService := application.NewDiagramService(detectors)

	// Generate diagram
	result, err := diagramService.Generate(projectRoot, cfg.Layers, cfg)
	if err != nil {
		return fmt.Errorf("failed to generate diagram: %w", err)
	}

	// Format output
	var outputContent string
	switch diagramFormat {
	case "ascii":
		outputContent = output.GenerateASCII(result)
	case "dot":
		outputContent = output.GenerateDOT(result)
	case "mermaid":
		outputContent = output.GenerateMermaid(result)
	}

	// Write output
	if diagramOutput != "" {
		// Write to file
		if err := os.WriteFile(diagramOutput, []byte(outputContent), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Diagram written to %s\n", diagramOutput)
	} else {
		// Write to stdout
		fmt.Fprint(cmd.OutOrStdout(), outputContent)
	}

	return nil
}
