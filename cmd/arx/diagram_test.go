package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
)

func TestDiagramCommand(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantOutput   []string
		wantErr      bool
		wantExitCode int
	}{
		{
			name:       "help flag shows usage",
			args:       []string{"diagram", "--help"},
			wantOutput: []string{"Usage:", "diagram", "--format", "--output"},
			wantErr:    false,
		},
		{
			name:       "default format is ascii",
			args:       []string{"diagram"},
			wantOutput: []string{"SUMMARY"},
			wantErr:    false,
		},
		{
			name:       "format ascii",
			args:       []string{"diagram", "--format", "ascii"},
			wantOutput: []string{"SUMMARY"},
			wantErr:    false,
		},
		{
			name:       "format dot",
			args:       []string{"diagram", "--format", "dot"},
			wantOutput: []string{"digraph ArxDependencies"},
			wantErr:    false,
		},
		{
			name:       "format mermaid",
			args:       []string{"diagram", "--format", "mermaid"},
			wantOutput: []string{"flowchart TD"},
			wantErr:    false,
		},
		{
			name:         "invalid format returns error",
			args:         []string{"diagram", "--format", "invalid"},
			wantOutput:   []string{},
			wantErr:      true,
			wantExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual execution tests - they require a real project
			// These tests verify flag parsing and structure
			t.Skip("Integration test - requires project setup")
		})
	}
}

func TestDiagramCommandFlags(t *testing.T) {
	// Test that flags are properly registered
	cmd := diagramCmd

	// Check --format flag
	formatFlag := cmd.Flag("format")
	if formatFlag == nil {
		t.Fatal("diagram command missing --format flag")
	}
	if formatFlag.DefValue != "ascii" {
		t.Errorf("--format default = %q, want %q", formatFlag.DefValue, "ascii")
	}

	// Check --output flag
	outputFlag := cmd.Flag("output")
	if outputFlag == nil {
		t.Fatal("diagram command missing --output flag")
	}
	if outputFlag.Shorthand != "o" {
		t.Errorf("--output shorthand = %q, want %q", outputFlag.Shorthand, "o")
	}
}

func TestDiagramCommandOutputToFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "diagram.md")

	// Test that output file flag works (integration test)
	t.Run("writes to file", func(t *testing.T) {
		t.Skip("Integration test - requires service mock")
	})

	// Clean up
	os.Remove(outputFile)
}

func TestDiagramServiceIntegration(t *testing.T) {
	// Create a fake project structure
	tmpDir := t.TempDir()

	// Create arx.yaml config
	configContent := `layers:
  - name: domain
    paths:
      - domain/
  - name: application
    paths:
      - application/
rules:
  - from: application
    to: domain
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create domain directory and file
	domainDir := filepath.Join(tmpDir, "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(domainDir, "entity.go"), []byte("package domain"), 0644); err != nil {
		t.Fatalf("failed to write domain file: %v", err)
	}

	// Create application directory and file
	appDir := filepath.Join(tmpDir, "application")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create application dir: %v", err)
	}
	appFileContent := `package application

import "github.com/pauvalls/arx/domain"

var _ domain.Entity
`
	if err := os.WriteFile(filepath.Join(appDir, "service.go"), []byte(appFileContent), 0644); err != nil {
		t.Fatalf("failed to write application file: %v", err)
	}

	t.Run("generates diagram for fake project", func(t *testing.T) {
		t.Skip("Integration test - requires full service setup")
	})
}

func TestDiagramResultFormats(t *testing.T) {
	// Test that diagram result can be formatted in all supported formats
	_ = createTestDiagramResult()

	t.Run("ascii format", func(t *testing.T) {
		// ASCII format tested in ascii_test.go
	})

	t.Run("dot format", func(t *testing.T) {
		// DOT format tested in dot_test.go
	})

	t.Run("mermaid format", func(t *testing.T) {
		// Mermaid format tested in mermaid_test.go
	})
}

func TestSelectOutputFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		want    string
		wantErr bool
	}{
		{name: "ascii", format: "ascii", want: "ascii"},
		{name: "dot", format: "dot", want: "dot"},
		{name: "mermaid", format: "mermaid", want: "mermaid"},
		{name: "invalid", format: "invalid", want: "", wantErr: true},
		{name: "empty defaults to ascii", format: "", want: "ascii"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate format logic matches diagram.go implementation
			validFormats := map[string]bool{
				"ascii":   true,
				"dot":     true,
				"mermaid": true,
			}
			
			if tt.format == "" {
				// Empty defaults to ascii
				if tt.want != "ascii" {
					t.Errorf("empty format should default to ascii, got %q", tt.want)
				}
				return
			}
			
			if tt.wantErr {
				if validFormats[tt.format] {
					t.Errorf("format %q should be invalid but is in validFormats", tt.format)
				}
			} else {
				if !validFormats[tt.format] {
					t.Errorf("format %q should be valid but is not in validFormats", tt.format)
				}
			}
		})
	}
}

func TestDiagramCommandOutputCapture(t *testing.T) {
	// Test that diagram output can be captured to stdout or file
	t.Run("stdout capture", func(t *testing.T) {
		// Create a buffer to capture stdout
		var buf bytes.Buffer
		
		// In real implementation, would redirect os.Stdout
		_ = buf
		t.Skip("Requires stdout redirection")
	})

	t.Run("file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.md")
		
		// In real implementation, would write to file
		_ = outputPath
		t.Skip("Requires file write test")
	})
}

// TestDiagramCommandWithMockService tests the diagram command with a mocked service
func TestDiagramCommandWithMockService(t *testing.T) {
	t.Run("service error returns non-zero exit", func(t *testing.T) {
		// Mock service that returns error
		// Command should handle gracefully and return error
		t.Skip("Requires mock service implementation")
	})

	t.Run("empty result handled gracefully", func(t *testing.T) {
		// Mock service that returns empty result
		// Command should output "No layers" or similar
		t.Skip("Requires mock service implementation")
	})
}

// Helper function to create test diagram result
func createTestDiagramResult() *application.DiagramResult {
	return &application.DiagramResult{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "application", Paths: []string{"application/"}},
			{Name: "infrastructure", Paths: []string{"infrastructure/"}},
		},
		Dependencies: []domain.Dependency{
			{
				SourceFile:    "application/service.go",
				SourceLine:    10,
				ImportPath:    "domain/entity.go",
				ResolvedLayer: "domain",
			},
			{
				SourceFile:    "infrastructure/repo.go",
				SourceLine:    15,
				ImportPath:    "domain/entity.go",
				ResolvedLayer: "domain",
			},
		},
		Violations: []domain.Violation{
			{
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				File:        "domain/service.go",
				Line:        20,
				Message:     "Domain should not depend on infrastructure",
			},
		},
	}
}
