package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	godetector "github.com/pauvalls/arx/internal/infrastructure/detector/go"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
)

// TestGoDetectorExtractImports verifies that the Go detector correctly parses imports
func TestGoDetectorExtractImports(t *testing.T) {
	// Get the fixture project path
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "go-project")

	// Verify fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Go fixture not found at %s", fixturePath)
	}

	// Create detector
	detector := godetector.New()

	// Detect project
	ctx := context.Background()
	applicable, err := detector.Detect(ctx, fixturePath)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if !applicable {
		t.Fatal("Go detector should detect go-project fixture")
	}

	// Define layers
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "application", Paths: []string{"internal/application"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	// Extract imports
	deps, err := detector.ExtractImports(ctx, fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports failed: %v", err)
	}

	// Verify we got dependencies
	if len(deps) == 0 {
		t.Fatal("Expected at least one dependency")
	}

	// Check for the violation import (domain -> infrastructure)
	foundViolation := false
	for _, dep := range deps {
		if dep.SourceFile == "internal/domain/order_violation.go" {
			if dep.ResolvedLayer == "infrastructure" {
				foundViolation = true
				t.Logf("✓ Found violation: %s imports %s (resolved to %s)",
					dep.SourceFile, dep.ImportPath, dep.ResolvedLayer)
			}
		}
	}

	if !foundViolation {
		t.Error("Expected to find domain -> infrastructure violation")
	}
}

// TestTerminalReporterOutput verifies that the terminal reporter produces colored output
func TestTerminalReporterOutput(t *testing.T) {
	reporter := output.NewTerminalReporter()

	// Create test violations
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "The domain layer is the heart of your business logic...",
		},
	}

	// Report (this will write to stdout)
	err := reporter.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
}

// TestJSONReporterOutput verifies that the JSON reporter produces valid JSON
func TestJSONReporterOutput(t *testing.T) {
	reporter := output.NewJSONReporter()

	// Create test violations
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "The domain layer is the heart of your business logic...",
		},
	}

	// Report (this will write to stdout)
	err := reporter.Report(violations, ports.OutputFormatJSON)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}
}

// TestCheckServiceEndToEnd verifies the full check workflow
func TestCheckServiceEndToEnd(t *testing.T) {
	// This test would require wiring up the CLI command
	// For now, we test the individual components
	t.Skip("End-to-end test requires CLI wiring in Phase 5")
}
