package lsp

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

// Helper: minimal config for tests
func testConfig() *domain.Config {
	return &domain.Config{
		Version: "1",
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "domain-no-infra",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     domain.RuleTypeCannot,
				Severity: domain.SeverityError,
			},
		},
	}
}

func TestNewServer_InitialState(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	if s == nil {
		t.Fatal("NewServer() returned nil")
	}
	if s.initialized {
		t.Error("server should not be initialized initially")
	}
	if s.shutdown {
		t.Error("server should not be shutdown initially")
	}
	if len(s.documents) != 0 {
		t.Errorf("documents map should be empty, got %d entries", len(s.documents))
	}
}

func TestServer_Initialize_SetsInitialized(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	params := InitializeParams{
		RootURI: "file:///home/user/project",
	}
	result, err := s.Initialize(params)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if !s.initialized {
		t.Error("server should be initialized after Initialize()")
	}
	if s.workspaceRoot != "file:///home/user/project" {
		t.Errorf("workspaceRoot = %q, want %q", s.workspaceRoot, "file:///home/user/project")
	}

	// Check capabilities are advertised
	if result.Capabilities.TextDocumentSync != TDSyncKindFull {
		t.Errorf("TextDocumentSync = %d, want %d", result.Capabilities.TextDocumentSync, TDSyncKindFull)
	}
	if !result.Capabilities.CodeActionProvider {
		t.Error("CodeActionProvider should be true")
	}
	if !result.Capabilities.HoverProvider {
		t.Error("HoverProvider should be true")
	}
	if result.Capabilities.DiagnosticProvider == nil {
		t.Fatal("DiagnosticProvider should be non-nil")
	}
	if !result.Capabilities.DiagnosticProvider.InterFileDependencies {
		t.Error("InterFileDependencies should be true")
	}
	if result.Capabilities.DiagnosticProvider.WorkspaceDiagnostics {
		t.Error("WorkspaceDiagnostics should be false")
	}
}

func TestServer_MethodBeforeInitialize_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	// Try to call DidOpen before Initialize
	_, err := s.DidOpen(DidOpenTextDocumentParams{})
	if err == nil {
		t.Fatal("expected error for method before initialize")
	}
	if err.Error() != "Server not initialized" {
		t.Errorf("error = %q, want %q", err.Error(), "Server not initialized")
	}
}

func TestServer_MethodAfterShutdown_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	// Initialize first
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	// Shutdown
	if err := s.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	// Now try to call DidOpen
	_, err = s.DidOpen(DidOpenTextDocumentParams{})
	if err == nil {
		t.Fatal("expected error for method after shutdown")
	}
}

func TestServer_Initialize_DoubleInitializeIdempotent(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{RootURI: "file:///project"})
	if err != nil {
		t.Fatalf("First Initialize() error = %v", err)
	}
	// Second initialize should succeed (idempotent)
	_, err = s.Initialize(InitializeParams{RootURI: "file:///project"})
	if err != nil {
		t.Fatalf("Second Initialize() error = %v", err)
	}
}

func TestServer_DidOpen_StoresDocument(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main\n",
		},
	}
	diags, err := s.DidOpen(params)
	if err != nil {
		t.Fatalf("DidOpen() error = %v", err)
	}

	// Document should be stored
	content, ok := s.documents["file:///test.go"]
	if !ok {
		t.Fatal("document not stored after DidOpen")
	}
	if content != "package main\n" {
		t.Errorf("stored content = %q, want %q", content, "package main\n")
	}

	_ = diags // diagnostics result checked in T-3.1
}

func TestServer_DidOpen_EmptyContent(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:  "file:///empty.go",
			Text: "",
		},
	}
	_, err = s.DidOpen(params)
	if err != nil {
		t.Fatalf("DidOpen() empty content error = %v", err)
	}

	content, ok := s.documents["file:///empty.go"]
	if !ok {
		t.Fatal("document not stored for empty content")
	}
	if content != "" {
		t.Errorf("stored content = %q, want empty", content)
	}
}

func TestServer_DidChange_UpdatesDocument(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// First open
	s.DidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main\n",
		},
	})

	// Then change
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///test.go"},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "package main\n\nimport \"fmt\"\n"},
		},
	}
	diags, err := s.DidChange(params)
	if err != nil {
		t.Fatalf("DidChange() error = %v", err)
	}

	content, ok := s.documents["file:///test.go"]
	if !ok {
		t.Fatal("document not found after DidChange")
	}
	if content != "package main\n\nimport \"fmt\"\n" {
		t.Errorf("updated content = %q, want %q", content, "package main\n\nimport \"fmt\"\n")
	}

	_ = diags
}

func TestServer_DidChange_UntrackedURI_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///unknown.go"},
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "new content"},
		},
	}
	_, err = s.DidChange(params)
	if err == nil {
		t.Fatal("expected error for untracked URI")
	}
}

func TestServer_DidClose_RemovesDocument(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Open then close
	s.DidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main\n",
		},
	})

	diags, err := s.DidClose(DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test.go"},
	})
	if err != nil {
		t.Fatalf("DidClose() error = %v", err)
	}

	if _, ok := s.documents["file:///test.go"]; ok {
		t.Error("document should be removed after DidClose")
	}

	_ = diags
}

func TestServer_DidClose_UntrackedURI_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	_, err = s.DidClose(DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///unknown.go"},
	})
	if err == nil {
		t.Fatal("expected error for untracked URI close")
	}
}

func TestServer_Shutdown_SetsShutdownFlag(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if err := s.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if !s.shutdown {
		t.Error("shutdown flag should be true after Shutdown()")
	}
}

func TestServer_Shutdown_BeforeInitialize_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	err := s.Shutdown()
	if err == nil {
		t.Fatal("expected error for Shutdown before Initialize")
	}
}

func TestServer_Shutdown_DoubleShutdown_ReturnsError(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	s.Shutdown()
	err = s.Shutdown()
	if err == nil {
		t.Fatal("expected error for double Shutdown")
	}
}

func TestServer_Exit_NoPriorShutdown(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	exitCode := s.Exit()
	if exitCode != 1 {
		t.Errorf("Exit() without prior Shutdown should return 1, got %d", exitCode)
	}
}

func TestServer_Exit_AfterShutdown(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	s.Shutdown()
	exitCode := s.Exit()
	if exitCode != 0 {
		t.Errorf("Exit() after Shutdown should return 0, got %d", exitCode)
	}
}

func TestServer_Lifecycle_FullSequence(t *testing.T) {
	s := NewServer(nil, nil, testConfig())

	// Not initialized → error
	_, err := s.DidOpen(DidOpenTextDocumentParams{})
	if err == nil {
		t.Fatal("expected error before initialize")
	}

	// Initialize
	_, err = s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// DidOpen works
	_, err = s.DidOpen(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:  "file:///test.go",
			Text: "package main\n",
		},
	})
	if err != nil {
		t.Fatalf("DidOpen() error = %v", err)
	}

	// Shutdown
	if err := s.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	// After shutdown → error
	_, err = s.DidOpen(DidOpenTextDocumentParams{})
	if err == nil {
		t.Fatal("expected error after shutdown")
	}

	// Exit with clean code
	code := s.Exit()
	if code != 0 {
		t.Errorf("Exit() code = %d, want 0", code)
	}
}

func TestServer_Initialize_EmptyRootURI(t *testing.T) {
	s := NewServer(nil, nil, testConfig())
	_, err := s.Initialize(InitializeParams{})
	if err != nil {
		t.Fatalf("Initialize() with empty RootURI error = %v", err)
	}
}
