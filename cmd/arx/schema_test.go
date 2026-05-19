package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSchemaGenerateCommand_Exists(t *testing.T) {
	// Verify the schema command and subcommand are registered
	cmd := &cobra.Command{}
	cmd.AddCommand(schemaCmd)

	schemaFound, _, err := cmd.Find([]string{"schema"})
	if err != nil {
		t.Fatalf("cmd.Find([\"schema\"]) error = %v", err)
	}
	if schemaFound == nil {
		t.Fatal("schema command not found")
	}

	genFound, _, err := cmd.Find([]string{"schema", "generate"})
	if err != nil {
		t.Fatalf("cmd.Find([\"schema\", \"generate\"]) error = %v", err)
	}
	if genFound == nil {
		t.Fatal("schema generate command not found")
	}
}

func TestSchemaGenerate_Flags(t *testing.T) {
	// Verify flag presence
	cmd := &cobra.Command{}
	cmd.AddCommand(schemaCmd)

	genCmd, _, _ := cmd.Find([]string{"schema", "generate"})

	if genCmd.Flags().Lookup("output") == nil {
		t.Error("missing --output flag")
	}
	if genCmd.Flags().Lookup("pretty") == nil {
		t.Error("missing --pretty flag")
	}
	if genCmd.Flags().Lookup("minified") == nil {
		t.Error("missing --minified flag")
	}
}

func TestSchemaGenerate_OutputToStdout(t *testing.T) {
	schemaGenerateOutput = "" // reset from previous tests

	cmd := &cobra.Command{}
	cmd.AddCommand(schemaCmd)

	var stdout strings.Builder
	cmd.SetOut(&stdout)

	cmd.SetArgs([]string{"schema", "generate"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("schema generate error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "$schema") {
		t.Errorf("output should contain $schema field.\n got: %s", output)
	}
	if !strings.Contains(output, "properties") {
		t.Errorf("output should contain properties.\n got: %s", output)
	}
}

func TestSchemaGenerate_OutputToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "schema.json")

	cmd := &cobra.Command{}
	cmd.AddCommand(schemaCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	cmd.SetArgs([]string{"schema", "generate", "--output", outputPath})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("schema generate error = %v", err)
	}

	// File should exist with content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), "$schema") {
		t.Errorf("file should contain $schema field.\n got: %s", string(data))
	}
}

func TestSchemaGenerate_MinifiedFormat(t *testing.T) {
	schemaGenerateOutput = "" // reset from previous tests

	cmd := &cobra.Command{}
	cmd.AddCommand(schemaCmd)

	var stdout strings.Builder
	cmd.SetOut(&stdout)

	cmd.SetArgs([]string{"schema", "generate", "--minified"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("schema generate --minified error = %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	// Minified JSON has no indentation
	if strings.Contains(output, "\n") && strings.Contains(output, "  ") {
		t.Errorf("minified output should not have indentation:\n%s", output)
	}
}
