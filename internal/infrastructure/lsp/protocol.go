package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// JSON-RPC types
// ---------------------------------------------------------------------------

// JSONRPCRequest represents a JSON-RPC 2.0 request or notification.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"` // nil for notifications
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Well-known JSON-RPC error codes.
var (
	ErrParse               = &RPCError{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest      = &RPCError{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound      = &RPCError{Code: -32601, Message: "Method not found"}
	ErrServerNotInitialized = &RPCError{Code: -32002, Message: "Server not initialized"}
)

// ---------------------------------------------------------------------------
// LSP types
// ---------------------------------------------------------------------------

// Position represents a zero-based line/character position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range between two positions.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a source location.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity int

const (
	DSError   DiagnosticSeverity = 1
	DSWarning DiagnosticSeverity = 2
	DSInfo    DiagnosticSeverity = 3
	DSHint    DiagnosticSeverity = 4
)

// Diagnostic represents a diagnostic (e.g. a lint error or warning).
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Code     string             `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// ---------------------------------------------------------------------------
// Initialize
// ---------------------------------------------------------------------------

// ClientCapabilities represents capabilities provided by the client.
type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

// TextDocumentClientCapabilities represents text document capabilities.
type TextDocumentClientCapabilities struct {
	Synchronization *SynchronizationCapabilities `json:"synchronization,omitempty"`
}

// SynchronizationCapabilities represents sync capabilities.
type SynchronizationCapabilities struct {
	DidSave bool `json:"didSave,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities.
type WorkspaceClientCapabilities struct {
	DidChangeWatchedFiles *DidChangeWatchedFilesCapabilities `json:"didChangeWatchedFiles,omitempty"`
}

// DidChangeWatchedFilesCapabilities represents file watching capabilities.
type DidChangeWatchedFilesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// InitializeParams represents the parameters for the initialize request.
type InitializeParams struct {
	ProcessID    *int                `json:"processId,omitempty"`
	RootURI      string              `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities  `json:"capabilities,omitempty"`
}

// TextDocumentSyncKind defines how text document changes are synced.
type TextDocumentSyncKind int

const (
	TDSyncKindNone        TextDocumentSyncKind = 0
	TDSyncKindFull        TextDocumentSyncKind = 1
	TDSyncKindIncremental TextDocumentSyncKind = 2
)

// DiagnosticRegistrationOptions describes diagnostic provider capabilities.
type DiagnosticRegistrationOptions struct {
	InterFileDependencies bool `json:"interFileDependencies"`
	WorkspaceDiagnostics  bool `json:"workspaceDiagnostics"`
}

// FileOperationPattern describes file operation filter patterns.
type FileOperationPattern struct {
	Pattern string `json:"pattern"`
}

// FileOperationRegistrationOptions describes file operation capabilities.
type FileOperationRegistrationOptions struct {
	Filters []FileOperationFilter `json:"filters"`
}

// FileOperationFilter describes a filter for file operations.
type FileOperationFilter struct {
	Scheme  string              `json:"scheme,omitempty"`
	Pattern FileOperationPattern `json:"pattern"`
}

// ServerCapabilities describes capabilities a server may advertise.
type ServerCapabilities struct {
	TextDocumentSync           TextDocumentSyncKind              `json:"textDocumentSync,omitempty"`
	CodeActionProvider         bool                              `json:"codeActionProvider,omitempty"`
	HoverProvider              bool                              `json:"hoverProvider,omitempty"`
	DiagnosticProvider         *DiagnosticRegistrationOptions    `json:"diagnosticProvider,omitempty"`
	Workspace                  *WorkspaceCapabilities            `json:"workspace,omitempty"`
}

// WorkspaceCapabilities describes workspace-level server capabilities.
type WorkspaceCapabilities struct {
	FileOperations *FileOperationCapabilities `json:"fileOperations,omitempty"`
}

// FileOperationCapabilities describes file operation capabilities.
type FileOperationCapabilities struct {
	DidCreate *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
	DidDelete *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
}

// InitializeResult represents the result of an initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// ---------------------------------------------------------------------------
// TextDocument Synchronization
// ---------------------------------------------------------------------------

// TextDocumentItem represents an open text document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentIdentifier identifies a text document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// DidOpenTextDocumentParams are the params for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentContentChangeEvent represents a content change in a text document.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidChangeTextDocumentParams are the params for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier     `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent    `json:"contentChanges"`
}

// DidCloseTextDocumentParams are the params for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams are the params for textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

// PublishDiagnosticsParams are the params for textDocument/publishDiagnostics.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// ---------------------------------------------------------------------------
// Code Actions
// ---------------------------------------------------------------------------

// CodeActionKind is a string identifying the kind of a code action.
type CodeActionKind string

const (
	CodeActionQuickFix          CodeActionKind = "quickfix"
	CodeActionRefactor          CodeActionKind = "refactor"
	CodeActionSourceOrganizeImports CodeActionKind = "source.organizeImports"
)

// CodeActionContext contains diagnostics and other context for code actions.
type CodeActionContext struct {
	Diagnostics []Diagnostic    `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// CodeActionParams are the params for textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// Command represents a command that can be executed.
type Command struct {
	Title     string            `json:"title"`
	Command   string            `json:"command"`
	Arguments []json.RawMessage `json:"arguments,omitempty"`
}

// CodeAction represents a code action that can be applied.
type CodeAction struct {
	Title       string         `json:"title"`
	Kind        CodeActionKind `json:"kind,omitempty"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	Command     *Command       `json:"command,omitempty"`
}

// ---------------------------------------------------------------------------
// Hover
// ---------------------------------------------------------------------------

// MarkupContent represents a markup content (e.g. markdown).
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Hover represents the result of a hover request.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// HoverParams are the params for textDocument/hover.
type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// ---------------------------------------------------------------------------
// File Events / Watchers
// ---------------------------------------------------------------------------

// FileChangeType represents the type of a file change.
type FileChangeType int

const (
	FileChangeTypeCreated FileChangeType = 1
	FileChangeTypeChanged FileChangeType = 2
	FileChangeTypeDeleted FileChangeType = 3
)

// FileEvent represents a file change event.
type FileEvent struct {
	URI  string         `json:"uri"`
	Type FileChangeType `json:"type"`
}

// DidChangeWatchedFilesParams are the params for workspace/didChangeWatchedFiles.
type DidChangeWatchedFilesParams struct {
	Changes []FileEvent `json:"changes"`
}

// ---------------------------------------------------------------------------
// Content-Length header parsing (LSP transport)
// ---------------------------------------------------------------------------

// ReadMessage reads a single JSON-RPC 2.0 message from a buffered reader.
// It parses the Content-Length header and reads the exact number of body bytes.
// The caller is responsible for creating and reusing the *bufio.Reader for the
// connection lifetime, ensuring buffered data is not lost between calls.
func ReadMessage(br *bufio.Reader) (*JSONRPCRequest, error) {
	// Read headers until empty line
	var contentLength int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			val := strings.TrimPrefix(line, "Content-Length: ")
			n, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length value %q: %w", val, err)
			}
			contentLength = n
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read exact body
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(br, body); err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON-RPC request: %w", err)
	}

	return &req, nil
}

// WriteMessage writes a JSON-RPC 2.0 response to the writer with Content-Length headers.
func WriteMessage(writer io.Writer, msg JSONRPCResponse) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling JSON-RPC response: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(writer, header); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("writing body: %w", err)
	}
	return nil
}
