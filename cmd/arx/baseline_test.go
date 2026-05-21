package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestBaselineCmd_Generation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal config
	configPath := filepath.Join(tmpDir, "arx.yaml")
	configContent := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: R001
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create a service with mock config reader that returns violations
	configReader := &mockConfigReader{
		config: &domain.Config{
			Version: domain.SchemaVersion{Major: 1, Minor: 0},
			Layers: []domain.Layer{
				{Name: "domain", Paths: []string{"domain/**"}},
				{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
			},
			Rules: []domain.Rule{
				{ID: "R001", From: "domain", To: []string{"infrastructure"}, Type: "Cannot", Severity: "error"},
			},
		},
	}

	detectors := []ports.Detector{
		&mockDetector{
			name:      "go",
			canDetect: true,
			dependencies: []domain.Dependency{
				{
					SourceFile:    "domain/entity.go",
					SourceLine:    5,
					ImportPath:    "github.com/example/infrastructure/db",
					ResolvedLayer: "infrastructure",
				},
			},
		},
	}

	// Use the real baseline service
	baselineSvc := newBaselineService()

	// Create a check service with our mocks
	checkSvc := &mockCheckService{
		reader:     configReader,
		detectors:  detectors,
		configPath: configPath,
		projectRoot: tmpDir,
	}

	// Run baseline generation manually (testing the logic, not the cobra command)
	configHash, err := configReader.config.Hash()
	if err != nil {
		t.Fatalf("config.Hash() error = %v", err)
	}
	baseline := baselineSvc.Generate(checkSvc.violations(), configHash)

	if baseline == nil {
		t.Fatal("baseline should not be nil")
	}

	if len(baseline.Violations) != 1 {
		t.Errorf("baseline should have 1 violation, got %d", len(baseline.Violations))
	}

	// Save and verify
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	if err := baselineSvc.Save(baseline, baselinePath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		t.Fatal("baseline file was not created")
	}

	// Verify content
	loaded, err := baselineSvc.Load(baselinePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("loaded baseline is nil")
	}

	if len(loaded.Violations) != 1 {
		t.Errorf("loaded baseline should have 1 violation, got %d", len(loaded.Violations))
	}
}

func TestBaselineCmd_Reset(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")

	svc := newBaselineService()

	// Create an existing baseline
	old := &domain.Baseline{
		Version: "1.0", ConfigHash: "old-hash",
		GeneratedAt: testTime,
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "old", File: "old.go"},
		},
	}
	if err := svc.Save(old, baselinePath); err != nil {
		t.Fatalf("failed to save old baseline: %v", err)
	}

	// Generate a new baseline (simulating --reset)
	newBaseline := svc.Generate([]domain.Violation{
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "new", File: "new.go", Line: 1},
	}, "new-hash")

	if err := svc.Save(newBaseline, baselinePath); err != nil {
		t.Fatalf("Save() with reset error = %v", err)
	}

	loaded, err := svc.Load(baselinePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ConfigHash != "new-hash" {
		t.Errorf("ConfigHash = %q, want %q (should be overwritten)", loaded.ConfigHash, "new-hash")
	}
	if len(loaded.Violations) != 1 || loaded.Violations[0].RuleID != "R002" {
		t.Errorf("Violations should be replaced, got %+v", loaded.Violations)
	}
}

func TestBaselineCmd_ExistingBaselineWarning(t *testing.T) {
	tmpDir := t.TempDir()
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")

	svc := newBaselineService()

	// Create an existing baseline
	old := &domain.Baseline{
		Version: "1.0", ConfigHash: "old-hash",
		GeneratedAt: testTime,
		Violations:  []domain.BaselineViolation{},
	}
	if err := svc.Save(old, baselinePath); err != nil {
		t.Fatalf("failed to save old baseline: %v", err)
	}

	// Check that exists returns true
	if !svc.Exists(baselinePath) {
		t.Error("Exists() should return true for existing baseline")
	}
}

// Helper to get a stable time for tests
var testTime = testTimeNow()

func testTimeNow() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

// mockCheckService provides a simple way to test baseline logic
type mockCheckService struct {
	reader      *mockConfigReader
	detectors   []ports.Detector
	configPath  string
	projectRoot string
}

func (m *mockCheckService) violations() []domain.Violation {
	// Simulate running a check and getting violations
	var violations []domain.Violation
	for _, d := range m.detectors {
		deps, _ := d.ExtractImports(nil, m.projectRoot, m.reader.config.Layers)
		for _, dep := range deps {
			// Check against rules
			for _, rule := range m.reader.config.Rules {
				if rule.From == dep.ResolvedLayer || ruleMatches(rule, dep) {
					violations = append(violations, domain.Violation{
						ID:          rule.ID,
						RuleID:      rule.ID,
						File:        dep.SourceFile,
						Line:        dep.SourceLine,
						SourceLayer: dep.ResolvedLayer,
						TargetLayer: "unknown",
						Import:      dep.ImportPath,
						Message:     rule.From + " cannot depend on " + strings.Join(rule.To, ", "),
					})
				}
			}
		}
	}
	return violations
}

func ruleMatches(rule domain.Rule, dep domain.Dependency) bool {
	// Simple check: if dependency's resolved layer matches any "to" layer
	for _, to := range rule.To {
		if to == dep.ResolvedLayer {
			return true
		}
	}
	return false
}
