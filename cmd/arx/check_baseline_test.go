package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	arxbaseline "github.com/pauvalls/arx/internal/infrastructure/baseline"
)
func TestCheckWithBaseline_SuppressedViolations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal config
	configPath := filepath.Join(tmpDir, "arx.yaml")
	configContent := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
rules:
  - id: R001
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create a baseline with one violation
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	storage := arxbaseline.NewStorage()
	baseline := &domain.Baseline{
		Version:    "1.0",
		ConfigHash: "test-hash",
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user.go"},
		},
	}
	if err := storage.Save(baseline, baselinePath); err != nil {
		t.Fatalf("failed to save baseline: %v", err)
	}

	// Verify baseline exists
	if !storage.Exists(baselinePath) {
		t.Fatal("baseline file should exist")
	}

	// Verify filtering works
	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user.go", Line: 10},
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "github.com/example/entity", File: "service.go", Line: 20},
	}

	filtered := baseline.Filter(violations)
	if len(filtered) != 1 {
		t.Errorf("Filter() returned %d violations, want 1 (R001 should be suppressed)", len(filtered))
	}
	if filtered[0].RuleID != "R002" {
		t.Errorf("Filter() kept %q, want R002", filtered[0].RuleID)
	}
}

func TestCheckWithBaseline_StaleWarning(t *testing.T) {
	baseline := &domain.Baseline{
		Version:    "1.0",
		ConfigHash: "old-hash",
		Violations: []domain.BaselineViolation{},
	}

	// Test IsStale
	if !baseline.IsStale("new-hash") {
		t.Error("baseline should be stale with different config hash")
	}
	if baseline.IsStale("old-hash") {
		t.Error("baseline should NOT be stale with same config hash")
	}
}

func TestCheckWithBaseline_NoBaselinePassthrough(t *testing.T) {
	// When no baseline exists, all violations should pass through
	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go", Line: 2},
	}

	var b *domain.Baseline
	filtered := b.Filter(violations)

	if len(filtered) != 2 {
		t.Errorf("nil baseline Filter() returned %d violations, want 2", len(filtered))
	}
}
