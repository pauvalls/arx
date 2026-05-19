package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestCompareViolations_Empty(t *testing.T) {
	result := CompareViolations(nil, nil)
	if result.HasChanges() {
		t.Error("expected no changes for empty sets")
	}
	if result.Summary() != "+0 violations, -0 resolved, 0 unchanged" {
		t.Errorf("unexpected summary: %s", result.Summary())
	}
}

func TestCompareViolations_AllNew(t *testing.T) {
	after := []domain.Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"},
	}
	result := CompareViolations(nil, after)
	if !result.HasChanges() {
		t.Error("expected changes")
	}
	if len(result.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(result.Added))
	}
	if len(result.Resolved) != 0 {
		t.Errorf("expected 0 resolved, got %d", len(result.Resolved))
	}
}

func TestCompareViolations_AllResolved(t *testing.T) {
	before := []domain.Violation{
		{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"},
	}
	result := CompareViolations(before, nil)
	if !result.HasChanges() {
		t.Error("expected changes")
	}
	if len(result.Resolved) != 1 {
		t.Errorf("expected 1 resolved, got %d", len(result.Resolved))
	}
	if len(result.Added) != 0 {
		t.Errorf("expected 0 added, got %d", len(result.Added))
	}
}

func TestCompareViolations_Unchanged(t *testing.T) {
	v := domain.Violation{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"}
	result := CompareViolations([]domain.Violation{v}, []domain.Violation{v})
	if result.HasChanges() {
		t.Error("expected no changes for identical sets")
	}
	if len(result.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.Unchanged))
	}
}

func TestCompareViolations_Mixed(t *testing.T) {
	v1 := domain.Violation{RuleID: "R1", SourceLayer: "domain", TargetLayer: "infra", Import: "pkg/db"}
	v2 := domain.Violation{RuleID: "R2", SourceLayer: "app", TargetLayer: "infra", Import: "pkg/cache"}
	vAdded := domain.Violation{RuleID: "R3", SourceLayer: "infra", TargetLayer: "domain", Import: "pkg/new"} // new

	// before = [v1, v2], after = [v2, vAdded] → v1 resolved, v2 unchanged, vAdded new
	result := CompareViolations([]domain.Violation{v1, v2}, []domain.Violation{v2, vAdded})
	if !result.HasChanges() {
		t.Error("expected changes")
	}
	if len(result.Resolved) != 1 {
		t.Errorf("expected 1 resolved, got %d", len(result.Resolved))
	}
	if len(result.Added) != 1 {
		t.Errorf("expected 1 added, got %d", len(result.Added))
	}
	if len(result.Unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.Unchanged))
	}
}

func TestViolationFingerprint(t *testing.T) {
	v := domain.Violation{
		RuleID:      "R1",
		SourceLayer: "domain",
		TargetLayer: "infrastructure",
		Import:      "github.com/pkg/db",
	}
	fp := violationFingerprint(v)
	expected := "R1:domain:infrastructure:github.com/pkg/db"
	if fp != expected {
		t.Errorf("fingerprint = %q, want %q", fp, expected)
	}
}
