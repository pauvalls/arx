package lsp

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestViolationToDiagnostic(t *testing.T) {
	v := domain.Violation{
		ID:          "D-01",
		RuleID:      "domain-imports-infrastructure",
		File:        "internal/domain/user.go",
		Line:        10,
		SourceLayer: "domain",
		TargetLayer: "infrastructure",
		Import:      "github.com/pauvalls/arx/internal/infrastructure/db",
		Message:     "Domain layer must not import infrastructure packages",
		Severity:    domain.SeverityError,
	}

	d := ViolationToDiagnostic(v, 80)
	if d.Code != "domain-imports-infrastructure" {
		t.Errorf("Code = %q, want %q", d.Code, "domain-imports-infrastructure")
	}
	if d.Source != "arx" {
		t.Errorf("Source = %q, want %q", d.Source, "arx")
	}
	if d.Message != "Domain layer must not import infrastructure packages" {
		t.Errorf("Message = %q, want %q", d.Message, "Domain layer must not import infrastructure packages")
	}
	if d.Severity != DSError {
		t.Errorf("Severity = %d, want %d", d.Severity, DSError)
	}
	// Range: line 10 → 0-based = 9, character 0 to lineLength
	if d.Range.Start.Line != 9 {
		t.Errorf("Range.Start.Line = %d, want %d", d.Range.Start.Line, 9)
	}
	if d.Range.End.Line != 9 {
		t.Errorf("Range.End.Line = %d, want %d", d.Range.End.Line, 9)
	}
	if d.Range.End.Character != 80 {
		t.Errorf("Range.End.Character = %d, want %d", d.Range.End.Character, 80)
	}
}

func TestViolationToDiagnostic_ZeroLine(t *testing.T) {
	v := domain.Violation{
		RuleID:   "test-rule",
		Line:     0,
		Message:  "test",
		Severity: domain.SeverityWarning,
	}

	d := ViolationToDiagnostic(v, 50)
	if d.Range.Start.Line != 0 {
		t.Errorf("Range.Start.Line = %d, want 0 for line 0", d.Range.Start.Line)
	}
	if d.Severity != DSWarning {
		t.Errorf("Severity = %d, want %d", d.Severity, DSWarning)
	}
}

func TestViolationToDiagnostic_NegativeLine(t *testing.T) {
	v := domain.Violation{
		RuleID:   "test-rule",
		Line:     -1,
		Message:  "test",
		Severity: domain.SeverityInfo,
	}

	d := ViolationToDiagnostic(v, 30)
	if d.Range.Start.Line < 0 {
		t.Errorf("Range.Start.Line should not be negative, got %d", d.Range.Start.Line)
	}
}

func TestViolationsToDiagnostics_Empty(t *testing.T) {
	diags := ViolationsToDiagnostics([]domain.Violation{})
	if diags == nil {
		t.Fatal("expected non-nil slice for empty violations")
	}
	if len(diags) != 0 {
		t.Errorf("got %d diagnostics, want 0", len(diags))
	}
}

func TestViolationsToDiagnostics_Multiple(t *testing.T) {
	violations := []domain.Violation{
		{
			RuleID:   "rule-1",
			File:     "a.go",
			Line:     5,
			Message:  "first violation",
			Severity: domain.SeverityError,
		},
		{
			RuleID:   "rule-2",
			File:     "b.go",
			Line:     10,
			Message:  "second violation",
			Severity: domain.SeverityWarning,
		},
		{
			RuleID:   "rule-3",
			File:     "c.go",
			Line:     15,
			Message:  "third violation",
			Severity: domain.SeverityInfo,
		},
	}

	diags := ViolationsToDiagnostics(violations)
	if len(diags) != 3 {
		t.Fatalf("got %d diagnostics, want 3", len(diags))
	}

	// Check severity mapping
	if diags[0].Severity != DSError {
		t.Errorf("diags[0].Severity = %d, want %d (error)", diags[0].Severity, DSError)
	}
	if diags[1].Severity != DSWarning {
		t.Errorf("diags[1].Severity = %d, want %d (warning)", diags[1].Severity, DSWarning)
	}
	if diags[2].Severity != DSInfo {
		t.Errorf("diags[2].Severity = %d, want %d (info)", diags[2].Severity, DSInfo)
	}
}

func TestSeverityMapping_DomainToLSP(t *testing.T) {
	tests := []struct {
		domain domain.Severity
		lsp    DiagnosticSeverity
	}{
		{domain.SeverityError, DSError},
		{domain.SeverityWarning, DSWarning},
		{domain.SeverityInfo, DSInfo},
		{domain.Severity("unknown"), DSInfo}, // fallback
		{"", DSInfo},                          // empty fallback
	}
	for _, tt := range tests {
		got := domainSeverityToLSP(tt.domain)
		if got != tt.lsp {
			t.Errorf("domainSeverityToLSP(%q) = %d, want %d", tt.domain, got, tt.lsp)
		}
	}
}

func TestViolationsToDiagnostics_NilInput(t *testing.T) {
	diags := ViolationsToDiagnostics(nil)
	if diags == nil {
		t.Fatal("expected non-nil slice for nil violations")
	}
	if len(diags) != 0 {
		t.Errorf("got %d diagnostics, want 0", len(diags))
	}
}
