package lsp

import (
	"context"
	"fmt"
	"sync"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
)

// Server holds the LSP server state and implements handler methods.
type Server struct {
	initialized   bool
	shutdown      bool
	documents     map[string]string // uri → text content
	workspaceRoot string
	checkService  *application.CheckService
	fixEngine     *application.FixEngine
	config        *domain.Config
	mu            sync.Mutex
}

// NewServer creates a new LSP server with the given dependencies.
func NewServer(cs *application.CheckService, fe *application.FixEngine, cfg *domain.Config) *Server {
	return &Server{
		documents:    make(map[string]string),
		checkService: cs,
		fixEngine:    fe,
		config:       cfg,
	}
}

// requireInit checks that the server is initialized and not shutdown.
// Returns an error if the server is not in the correct state.
func (s *Server) requireInit() error {
	if s.shutdown {
		return fmt.Errorf("Invalid Request")
	}
	if !s.initialized {
		return fmt.Errorf("Server not initialized")
	}
	return nil
}

// Initialize handles the initialize request. Returns capabilities.
// Idempotent — safe to call multiple times.
func (s *Server) Initialize(params InitializeParams) (InitializeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = true
	if params.RootURI != "" {
		s.workspaceRoot = params.RootURI
	}

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:   TDSyncKindFull,
			CodeActionProvider: true,
			HoverProvider:      true,
			DiagnosticProvider: &DiagnosticRegistrationOptions{
				InterFileDependencies: true,
				WorkspaceDiagnostics:  false,
			},
			Workspace: &WorkspaceCapabilities{
				FileOperations: &FileOperationCapabilities{
					DidCreate: &FileOperationRegistrationOptions{
						Filters: []FileOperationFilter{
							{
								Scheme: "file",
								Pattern: FileOperationPattern{
									Pattern: "**/*.{go,yaml,yml}",
								},
							},
						},
					},
					DidDelete: &FileOperationRegistrationOptions{
						Filters: []FileOperationFilter{
							{
								Scheme: "file",
								Pattern: FileOperationPattern{
									Pattern: "**/*.{go,yaml,yml}",
								},
							},
						},
					},
				},
			},
		},
	}

	return result, nil
}

// DidOpen handles textDocument/didOpen. Stores the document content and
// returns diagnostics for the file.
func (s *Server) DidOpen(params DidOpenTextDocumentParams) ([]Diagnostic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.requireInit(); err != nil {
		return nil, err
	}

	s.documents[params.TextDocument.URI] = params.TextDocument.Text
	// Diagnostics are computed by the caller via the diagnostics service
	// Return nil — diagnostics will be pushed separately
	return nil, nil
}

// DidChange handles textDocument/didChange. Updates the stored document content
// and returns diagnostics for the file.
func (s *Server) DidChange(params DidChangeTextDocumentParams) ([]Diagnostic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.requireInit(); err != nil {
		return nil, err
	}

	if _, ok := s.documents[params.TextDocument.URI]; !ok {
		return nil, fmt.Errorf("document not open: %s", params.TextDocument.URI)
	}

	if len(params.ContentChanges) > 0 {
		// Full sync: replace with last content change
		s.documents[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
	}

	// Diagnostics are computed by the caller
	return nil, nil
}

// DidClose handles textDocument/didClose. Removes the document from the cache.
func (s *Server) DidClose(params DidCloseTextDocumentParams) ([]Diagnostic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.requireInit(); err != nil {
		return nil, err
	}

	if _, ok := s.documents[params.TextDocument.URI]; !ok {
		return nil, fmt.Errorf("document not open: %s", params.TextDocument.URI)
	}

	delete(s.documents, params.TextDocument.URI)
	return nil, nil
}

// Shutdown handles the shutdown request. After shutdown, all method requests
// return InvalidRequest.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return fmt.Errorf("Server not initialized")
	}
	if s.shutdown {
		return fmt.Errorf("already shutdown")
	}

	s.shutdown = true
	return nil
}

// Exit handles the exit notification. Returns 0 if shutdown was called, 1 otherwise.
func (s *Server) Exit() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shutdown {
		return 0
	}
	return 1
}

// IsInitialized returns whether the server has been initialized.
func (s *Server) IsInitialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

// IsShutdown returns whether the server has been shut down.
func (s *Server) IsShutdown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdown
}

// GetAllDocuments returns a copy of all open document URIs.
func (s *Server) GetAllDocuments() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	docs := make(map[string]string, len(s.documents))
	for k, v := range s.documents {
		docs[k] = v
	}
	return docs
}

// GetDocument returns the cached content for a URI, or empty string if not found.
func (s *Server) GetDocument(uri string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.documents[uri]
}

// GetWorkspaceRoot returns the workspace root URI.
func (s *Server) GetWorkspaceRoot() string {
	return s.workspaceRoot
}

// GetConfig returns the server's config.
func (s *Server) GetConfig() *domain.Config {
	return s.config
}

// GetCheckService returns the check service.
func (s *Server) GetCheckService() *application.CheckService {
	return s.checkService
}

// GetFixEngine returns the fix engine.
func (s *Server) GetFixEngine() *application.FixEngine {
	return s.fixEngine
}

// ComputeDiagnostics runs the check service and returns LSP diagnostics for all violations.
func (s *Server) ComputeDiagnostics(ctx context.Context) []Diagnostic {
	if s.checkService == nil || s.config == nil {
		return []Diagnostic{}
	}

	// Run full detection
	projectRoot := s.workspaceRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	deps, err := s.checkService.DetectCached(ctx, projectRoot, s.config.Layers)
	if err != nil {
		return []Diagnostic{}
	}

	// Evaluate rules
	violations := s.checkService.Evaluate(deps, s.config.Rules, s.config.Layers)
	if len(violations) == 0 {
		return []Diagnostic{}
	}

	return ViolationsToDiagnostics(violations)
}
