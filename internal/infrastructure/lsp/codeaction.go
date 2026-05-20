package lsp

import (
	"encoding/json"

	"github.com/pauvalls/arx/internal/domain"
)

// ComputeCodeActions computes code actions for a given code action request.
// It matches diagnostics against the FixEngine and returns applicable code actions.
func ComputeCodeActions(s *Server, params CodeActionParams) []CodeAction {
	if s.fixEngine == nil || len(params.Context.Diagnostics) == 0 {
		return []CodeAction{}
	}

	var actions []CodeAction
	for _, diag := range params.Context.Diagnostics {
		if diag.Code == "" {
			continue
		}
		// Build a violation from the diagnostic for FixEngine
		v := violationFromDiagnostic(diag, params.TextDocument.URI)
		fix := s.fixEngine.SuggestFix(v)
		if fix == nil {
			continue
		}

		args := []json.RawMessage{
			json.RawMessage(`"` + diag.Code + `"`),
		}

		action := CodeAction{
			Title: "Apply arx fix: " + fix.Description,
			Kind:  CodeActionQuickFix,
			Diagnostics: []Diagnostic{diag},
			Command: &Command{
				Title:     "Apply arx fix",
				Command:   "arx.fix",
				Arguments: args,
			},
		}
		actions = append(actions, action)
	}

	if actions == nil {
		return []CodeAction{}
	}
	return actions
}

// violationFromDiagnostic builds a minimal domain.Violation from an LSP diagnostic.
func violationFromDiagnostic(d Diagnostic, uri string) domain.Violation {
	return domain.Violation{
		RuleID:      d.Code,
		File:        uriToPath(uri),
		Line:        d.Range.Start.Line + 1, // LSP is 0-based, arx is 1-based
		Message:     d.Message,
		Severity:    domain.Severity(diagnosticSeverityToDomain(d.Severity)),
	}
}

// diagnosticSeverityToDomain maps LSP diagnostic severity to domain severity.
func diagnosticSeverityToDomain(sev DiagnosticSeverity) string {
	switch sev {
	case DSError:
		return "error"
	case DSWarning:
		return "warning"
	case DSInfo:
		return "info"
	case DSHint:
		return "info"
	default:
		return "error"
	}
}

// uriToPath converts a file:// URI to a filesystem path.
func uriToPath(uri string) string {
	if len(uri) > 7 && uri[:7] == "file://" {
		return uri[7:]
	}
	return uri
}


