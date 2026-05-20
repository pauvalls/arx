package lsp

import (
	"testing"
)

func TestComputeHover_OnImportLine(t *testing.T) {
	cfg := testConfig()
	s := NewServer(nil, nil, cfg)
	s.initialized = true
	s.documents["file:///project/main.go"] = "package main\n\nimport \"internal/domain/user\"\n"

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///project/main.go"},
		Position:     Position{Line: 2, Character: 10},
	}

	hover := ComputeHover(s, params)
	if hover == nil {
		t.Fatal("expected hover, got nil")
	}
	if hover.Contents.Kind != "markdown" {
		t.Errorf("Kind = %q, want %q", hover.Contents.Kind, "markdown")
	}
	if hover.Contents.Value == "" {
		t.Error("expected non-empty hover content")
	}
}

func TestComputeHover_OnNonImportLine(t *testing.T) {
	cfg := testConfig()
	s := NewServer(nil, nil, cfg)
	s.initialized = true
	s.documents["file:///project/main.go"] = "package main\n\nfunc main() {}\n"

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///project/main.go"},
		Position:     Position{Line: 2, Character: 5},
	}

	hover := ComputeHover(s, params)
	if hover != nil {
		t.Errorf("expected nil hover for non-import line, got %+v", hover)
	}
}

func TestComputeHover_UnknownURI(t *testing.T) {
	cfg := testConfig()
	s := NewServer(nil, nil, cfg)
	s.initialized = true

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///unknown.go"},
		Position:     Position{Line: 0, Character: 0},
	}

	hover := ComputeHover(s, params)
	if hover != nil {
		t.Errorf("expected nil hover for unknown URI, got %+v", hover)
	}
}

func TestComputeHover_OutOfBoundsLine(t *testing.T) {
	cfg := testConfig()
	s := NewServer(nil, nil, cfg)
	s.initialized = true
	s.documents["file:///project/main.go"] = "package main\n"

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///project/main.go"},
		Position:     Position{Line: 100, Character: 0},
	}

	hover := ComputeHover(s, params)
	if hover != nil {
		t.Errorf("expected nil hover for out-of-bounds line, got %+v", hover)
	}
}

func TestComputeHover_TypeScriptImport(t *testing.T) {
	cfg := testConfig()
	s := NewServer(nil, nil, cfg)
	s.initialized = true
	s.documents["file:///project/app.ts"] = "import { Component } from \"react\"\n"

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///project/app.ts"},
		Position:     Position{Line: 0, Character: 10},
	}

	hover := ComputeHover(s, params)
	if hover != nil {
		t.Logf("hover content: %s", hover.Contents.Value)
	}
	// May or may not find a matching layer — that's fine
}

func TestComputeHover_NilConfig(t *testing.T) {
	s := NewServer(nil, nil, nil)
	s.initialized = true
	s.documents["file:///project/main.go"] = "package main\n\nimport \"fmt\"\n"

	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///project/main.go"},
		Position:     Position{Line: 2, Character: 5},
	}

	hover := ComputeHover(s, params)
	if hover != nil {
		t.Errorf("expected nil hover with nil config, got %+v", hover)
	}
}

func TestExtractImportPath(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{`import "fmt"`, "fmt"},
		{`import "github.com/pauvalls/arx/internal/domain"`, "github.com/pauvalls/arx/internal/domain"},
		{`import (`, ""}, // multi-line import start
		{`import { Component } from "react"`, "react"},
		{`import "side-effect"`, "side-effect"},
		{`package main`, ""},
		{`func main() {}`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		got := extractImportPath(tt.line)
		if got != tt.want {
			t.Errorf("extractImportPath(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}
