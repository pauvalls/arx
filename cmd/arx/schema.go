package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command group
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage JSON Schema for arx configuration",
	Long:  `Commands for managing JSON Schema files for arx configuration.`,
}

var (
	schemaGenerateOutput  string
	schemaGeneratePretty  bool
	schemaGenerateMinified bool
)

// schemaGenerateCmd represents the schema generate command
var schemaGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate JSON Schema for arx configuration",
	Long: `Generate a JSON Schema document that describes the arx configuration
structure. This schema can be used for IDE autocompletion, validation,
and documentation.

By default, the schema is printed to stdout in pretty-printed format.

Examples:
  arx schema generate
  arx schema generate --output arx-schema.json
  arx schema generate --minified`,
	Args: cobra.NoArgs,
	RunE: runSchemaGenerate,
}

func init() {
	schemaGenerateCmd.Flags().StringVarP(&schemaGenerateOutput, "output", "o", "", "Write schema to file instead of stdout")
	schemaGenerateCmd.Flags().BoolVar(&schemaGeneratePretty, "pretty", false, "Pretty-print the schema (default: auto-detect)")
	schemaGenerateCmd.Flags().BoolVar(&schemaGenerateMinified, "minified", false, "Minified output (no whitespace)")
	schemaCmd.AddCommand(schemaGenerateCmd)
	rootCmd.AddCommand(schemaCmd)
}

func runSchemaGenerate(cmd *cobra.Command, args []string) error {
	gen := &config.SchemaGeneratorImpl{}
	schema, err := gen.Generate("arx-config", domain.Config{})
	if err != nil {
		return fmt.Errorf("generating schema: %w", err)
	}

	// Determine output format
	var output []byte
	if schemaGenerateMinified {
		// Minified: compact JSON
		var compact map[string]interface{}
		if err := json.Unmarshal(schema, &compact); err != nil {
			return fmt.Errorf("unmarshaling schema: %w", err)
		}
		output, err = json.Marshal(compact)
		if err != nil {
			return fmt.Errorf("marshaling minified schema: %w", err)
		}
	} else {
		output = schema
	}

	// Write to file or stdout
	if schemaGenerateOutput != "" {
		if err := os.WriteFile(schemaGenerateOutput, output, 0644); err != nil {
			return fmt.Errorf("writing schema to %s: %w", schemaGenerateOutput, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Schema written to %s\n", schemaGenerateOutput)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), string(output))
	}

	return nil
}
