package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestNewFixEngine_CreatesEngineWithTemplates(t *testing.T) {
	engine := NewFixEngine()

	if engine == nil {
		t.Fatal("expected non-nil FixEngine")
	}
	if engine.templates == nil {
		t.Fatal("expected non-nil templates map")
	}
	if len(engine.templates) < 2 {
		t.Errorf("expected at least 2 templates, got %d", len(engine.templates))
	}
	if _, ok := engine.templates["domain-imports-infrastructure"]; !ok {
		t.Error("expected domain-imports-infrastructure template")
	}
	if _, ok := engine.templates["application-imports-infrastructure"]; !ok {
		t.Error("expected application-imports-infrastructure template")
	}
	if _, ok := engine.templates["domain-no-infra"]; !ok {
		t.Error("expected domain-no-infra template")
	}
	if _, ok := engine.templates["app-no-infra"]; !ok {
		t.Error("expected app-no-infra template")
	}
}

func TestSuggestFix_KnownViolation_ReturnsTemplateFix(t *testing.T) {
	engine := NewFixEngine()

	v := domain.Violation{
		ID:          "D-01",
		RuleID:      "domain-imports-infrastructure",
		File:        "internal/domain/user.go",
		Line:        5,
		SourceLayer: "domain",
		TargetLayer: "infrastructure",
		Import:      "github.com/pauvalls/arx/internal/infrastructure/db",
	}

	fix := engine.SuggestFix(v)

	if fix == nil {
		t.Fatal("expected non-nil fix")
	}
	if fix.ViolationID != "D-01" {
		t.Errorf("expected ViolationID D-01, got %s", fix.ViolationID)
	}
	if fix.RuleID != "domain-imports-infrastructure" {
		t.Errorf("expected RuleID domain-imports-infrastructure, got %s", fix.RuleID)
	}
	if fix.File != "internal/domain/user.go" {
		t.Errorf("expected File internal/domain/user.go, got %s", fix.File)
	}
	if fix.Line != 5 {
		t.Errorf("expected Line 5, got %d", fix.Line)
	}
	if fix.Description == "" {
		t.Error("expected non-empty Description")
	}
	if fix.Diff == "" {
		t.Error("expected non-empty Diff")
	}
}

func TestSuggestFix_ApplicationInfrastructure_ReturnsTemplateFix(t *testing.T) {
	engine := NewFixEngine()

	v := domain.Violation{
		ID:          "A-03",
		RuleID:      "application-imports-infrastructure",
		File:        "internal/application/service.go",
		Line:        12,
		SourceLayer: "application",
		TargetLayer: "infrastructure",
		Import:      "github.com/pauvalls/arx/internal/infrastructure/cache",
	}

	fix := engine.SuggestFix(v)

	if fix == nil {
		t.Fatal("expected non-nil fix")
	}
	if fix.ViolationID != "A-03" {
		t.Errorf("expected ViolationID A-03, got %s", fix.ViolationID)
	}
	if fix.Line != 12 {
		t.Errorf("expected Line 12, got %d", fix.Line)
	}
	if fix.Description == "" {
		t.Error("expected non-empty Description")
	}
}

func TestSuggestFix_UnknownViolation_ReturnsGenericAdvice(t *testing.T) {
	engine := NewFixEngine()

	v := domain.Violation{
		ID:          "X-99",
		RuleID:      "some-unknown-rule",
		File:        "internal/unknown/file.go",
		Line:        1,
		SourceLayer: "presentation",
		TargetLayer: "domain",
	}

	fix := engine.SuggestFix(v)

	if fix == nil {
		t.Fatal("expected non-nil fix for unknown violation")
	}
	if fix.ViolationID != "X-99" {
		t.Errorf("expected ViolationID X-99, got %s", fix.ViolationID)
	}
	if fix.Diff == "" {
		t.Error("expected non-empty Diff for generic advice (template-based)")
	}
	if fix.Description == "" {
		t.Error("expected non-empty Description for generic advice")
	}
}

func TestSuggestAll_ReturnsFixesForMultipleViolations(t *testing.T) {
	engine := NewFixEngine()

	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-imports-infrastructure",
			File:        "internal/domain/user.go",
			Line:        5,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/pauvalls/arx/internal/infrastructure/db",
		},
		{
			ID:          "A-03",
			RuleID:      "application-imports-infrastructure",
			File:        "internal/application/service.go",
			Line:        12,
			SourceLayer: "application",
			TargetLayer: "infrastructure",
			Import:      "github.com/pauvalls/arx/internal/infrastructure/cache",
		},
		{
			ID:          "X-99",
			RuleID:      "unknown-rule",
			File:        "internal/unknown/file.go",
			Line:        1,
			SourceLayer: "presentation",
			TargetLayer: "domain",
		},
	}

	fixes := engine.SuggestAll(violations)

	if len(fixes) != 3 {
		t.Fatalf("expected 3 fixes, got %d", len(fixes))
	}
	if fixes[0].ViolationID != "D-01" {
		t.Errorf("expected first fix for D-01, got %s", fixes[0].ViolationID)
	}
	if fixes[1].ViolationID != "A-03" {
		t.Errorf("expected second fix for A-03, got %s", fixes[1].ViolationID)
	}
	if fixes[2].ViolationID != "X-99" {
		t.Errorf("expected third fix for X-99, got %s", fixes[2].ViolationID)
	}
}

func TestSuggestAll_EmptyViolations_ReturnsEmptyFixes(t *testing.T) {
	engine := NewFixEngine()

	fixes := engine.SuggestAll([]domain.Violation{})

	if len(fixes) != 0 {
		t.Errorf("expected 0 fixes for empty violations, got %d", len(fixes))
	}
}
