package lsp

import (
	"github.com/pauvalls/arx/internal/domain"
)

// ViolationToDiagnostic maps a single domain.Violation to an LSP Diagnostic.
// lineLength is the length of the line where the violation occurs; used for range end.
func ViolationToDiagnostic(v domain.Violation, lineLength int) Diagnostic {
	line := v.Line
	if line <= 1 {
		line = 1
	}
	zeroBased := line - 1

	return Diagnostic{
		Range: Range{
			Start: Position{Line: zeroBased, Character: 0},
			End:   Position{Line: zeroBased, Character: lineLength},
		},
		Severity: domainSeverityToLSP(v.Severity),
		Code:     v.RuleID,
		Source:   "arx",
		Message:  v.Message,
	}
}

// ViolationsToDiagnostics maps a slice of domain.Violation to an LSP Diagnostic slice.
// Returns an empty (non-nil) slice when there are no violations.
func ViolationsToDiagnostics(violations []domain.Violation) []Diagnostic {
	if len(violations) == 0 {
		return []Diagnostic{}
	}

	diags := make([]Diagnostic, len(violations))
	for i, v := range violations {
		// Use a reasonable default line length when not available
		lineLen := 80
		diags[i] = ViolationToDiagnostic(v, lineLen)
	}
	return diags
}

// domainSeverityToLSP maps a domain.Severity to an LSP DiagnosticSeverity.
func domainSeverityToLSP(sev domain.Severity) DiagnosticSeverity {
	switch sev {
	case domain.SeverityError:
		return DSError
	case domain.SeverityWarning:
		return DSWarning
	case domain.SeverityInfo:
		return DSInfo
	default:
		return DSInfo
	}
}
