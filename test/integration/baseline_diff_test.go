package integration_test

import (
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
)

func TestBaseline_StaleDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create baseline with old config hash
	b := &domain.Baseline{
		Version:    "1.0",
		ConfigHash: "old-hash-value",
		Violations: []domain.BaselineViolation{
			{RuleID: "test-rule", SourceLayer: "domain", TargetLayer: "application", Import: "test"},
		},
	}

	// Check staleness
	if !b.IsStale("new-hash-value") {
		t.Error("Expected baseline to be stale with different config hash")
	}

	if b.IsStale("old-hash-value") {
		t.Error("Expected baseline to NOT be stale with same config hash")
	}
}

func TestBaseline_FilterSuppressesKnown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create baseline with one violation fingerprint
	b := &domain.Baseline{
		Version:    "1.0",
		ConfigHash: "test-hash",
		Violations: []domain.BaselineViolation{
			{RuleID: "domain-no-import-app", SourceLayer: "domain", TargetLayer: "application", Import: "com.example.app.Service"},
		},
	}

	// Create violations - one matches baseline, one is new
	violations := []domain.Violation{
		{RuleID: "domain-no-import-app", SourceLayer: "domain", TargetLayer: "application", Import: "com.example.app.Service", File: "domain/entity.go"},
		{RuleID: "domain-no-import-app", SourceLayer: "domain", TargetLayer: "application", Import: "com.example.app.NewService", File: "domain/other.go"},
	}

	filtered := b.Filter(violations)

	// Only the NEW violation should remain
	if len(filtered) != 1 {
		t.Errorf("Expected 1 remaining violation, got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].Import != "com.example.app.NewService" {
		t.Errorf("Expected new violation to remain, got: %s", filtered[0].Import)
	}
}

func TestDiffResult_Summary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	diff := &application.DiffResult{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
		Added:     []domain.Violation{{ID: "D-01"}, {ID: "D-02"}, {ID: "D-03"}},
		Resolved:  []domain.Violation{{ID: "D-04"}},
		Unchanged: []domain.Violation{{ID: "D-05"}, {ID: "D-06"}},
	}

	summary := diff.Summary()

	if !strings.Contains(summary, "+3") {
		t.Errorf("Summary should contain '+3', got: %s", summary)
	}
	if !strings.Contains(summary, "-1") {
		t.Errorf("Summary should contain '-1', got: %s", summary)
	}
	if !diff.HasChanges() {
		t.Error("Expected HasChanges() to return true")
	}
}

func TestDiffResult_NoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	diff := &application.DiffResult{
		RefBefore:   "HEAD~1",
		RefAfter:    "HEAD",
		Unchanged: []domain.Violation{{ID: "D-01"}},
	}

	if diff.HasChanges() {
		t.Error("Expected HasChanges() to return false when no added/resolved")
	}

	summary := diff.Summary()
	if !strings.Contains(summary, "0") {
		t.Errorf("Summary should show zeros, got: %s", summary)
	}
}

func TestBaseline_GenerateFromViolations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	violations := []domain.Violation{
		{RuleID: "rule-1", SourceLayer: "domain", TargetLayer: "app", Import: "x"},
		{RuleID: "rule-2", SourceLayer: "app", TargetLayer: "infra", Import: "y"},
	}

	b := domain.GenerateBaseline(violations, "config-hash-123")

	if b.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", b.Version)
	}
	if b.ConfigHash != "config-hash-123" {
		t.Errorf("Expected config hash, got %s", b.ConfigHash)
	}
	if len(b.Violations) != 2 {
		t.Errorf("Expected 2 baseline violations, got %d", len(b.Violations))
	}

	// Verify fingerprints are stored
	for _, bv := range b.Violations {
		if bv.RuleID == "" || bv.SourceLayer == "" || bv.TargetLayer == "" || bv.Import == "" {
			t.Errorf("Baseline violation missing fingerprint fields: %+v", bv)
		}
	}
}
