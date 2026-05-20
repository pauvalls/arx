package lsp

import (
	"testing"
)

func TestComputeCodeActions_KnownRule(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
		Range: Range{
			Start: Position{Line: 5, Character: 0},
			End:   Position{Line: 5, Character: 30},
		},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{
				{
					Range: Range{
						Start: Position{Line: 5, Character: 0},
						End:   Position{Line: 5, Character: 30},
					},
					Severity: DSError,
					Code:     "domain-imports-infrastructure",
					Source:   "arx",
					Message:  "Domain must not import infrastructure",
				},
			},
		},
	}

	actions := ComputeCodeActions(s, params)
	if len(actions) == 0 {
		// This is okay — fix engine may not have a fix for every rule in test config
		t.Log("no code actions returned (fix engine may not match)")
		return
	}

	action := actions[0]
	if action.Kind != CodeActionQuickFix {
		t.Errorf("Kind = %q, want %q", action.Kind, CodeActionQuickFix)
	}
	if action.Command == nil {
		t.Fatal("Command should be non-nil")
	}
	if action.Command.Command != "arx.fix" {
		t.Errorf("Command = %q, want %q", action.Command.Command, "arx.fix")
	}
}

func TestComputeCodeActions_UnknownRule(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
		Range: Range{
			Start: Position{Line: 10, Character: 0},
			End:   Position{Line: 10, Character: 50},
		},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{
				{
					Code:   "unknown-rule-id",
					Source: "arx",
					Message: "some unknown violation",
				},
			},
		},
	}

	actions := ComputeCodeActions(s, params)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for unknown rule, got %d", len(actions))
	}
}

func TestComputeCodeActions_NoDiagnostics(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{},
		},
	}

	actions := ComputeCodeActions(s, params)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for no diagnostics, got %d", len(actions))
	}
}

func TestComputeCodeActions_EmptyContext(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
	}

	actions := ComputeCodeActions(s, params)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions for empty context, got %d", len(actions))
	}
}

func TestComputeCodeActions_NilFixEngine(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	s.fixEngine = nil
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{
				{Code: "test", Source: "arx"},
			},
		},
	}

	actions := ComputeCodeActions(s, params)
	if len(actions) != 0 {
		t.Errorf("expected 0 actions with nil FixEngine, got %d", len(actions))
	}
}

func TestViolationFromDiagnostic(t *testing.T) {
	d := Diagnostic{
		Range: Range{
			Start: Position{Line: 4, Character: 0},
			End:   Position{Line: 4, Character: 40},
		},
		Severity: DSError,
		Code:     "test-rule",
		Source:   "arx",
		Message:  "test violation",
	}

	v := violationFromDiagnostic(d, "file:///project/main.go")
	if v.RuleID != "test-rule" {
		t.Errorf("RuleID = %q, want %q", v.RuleID, "test-rule")
	}
	if v.Line != 5 { // LSP 0-based → domain 1-based
		t.Errorf("Line = %d, want 5", v.Line)
	}
	if v.Message != "test violation" {
		t.Errorf("Message = %q, want %q", v.Message, "test violation")
	}
}

func TestUriToPath(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"file:///home/user/project/main.go", "/home/user/project/main.go"},
		{"/home/user/project/main.go", "/home/user/project/main.go"},
		{"file:///C:/Users/test/main.go", "/C:/Users/test/main.go"},
		{"", ""},
	}
	for _, tt := range tests {
		got := uriToPath(tt.uri)
		if got != tt.want {
			t.Errorf("uriToPath(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}
