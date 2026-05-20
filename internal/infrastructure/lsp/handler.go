package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Run starts the LSP message loop. It reads JSON-RPC 2.0 messages from stdin,
// dispatches them to the Server, and writes responses to stdout.
// The loop terminates on EOF, error, or exit notification.
func Run(ctx context.Context, s *Server, stdin io.Reader, stdout io.Writer) error {
	br := bufio.NewReader(stdin)

	for {
		req, err := ReadMessage(br)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		stop, err := dispatch(ctx, s, req, stdout)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
}

// dispatch routes a single JSON-RPC message to the appropriate handler.
// Returns stop=true if the server should stop (on exit), and any fatal error.
func dispatch(ctx context.Context, s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	switch req.Method {
	case "initialize":
		return handleInitialize(s, req, stdout)
	case "initialized":
		// Notification — ignore (idempotent)
		return false, nil
	case "textDocument/didOpen":
		return handleDidOpen(ctx, s, req, stdout)
	case "textDocument/didChange":
		return handleDidChange(ctx, s, req, stdout)
	case "textDocument/didClose":
		return handleDidClose(s, req, stdout)
	case "textDocument/didSave":
		return handleDidSave(ctx, s, req, stdout)
	case "textDocument/codeAction":
		return handleCodeAction(s, req, stdout)
	case "textDocument/hover":
		return handleHover(s, req, stdout)
	case "shutdown":
		return handleShutdown(s, req, stdout)
	case "exit":
		return handleExit(s, req)
	case "workspace/didChangeWatchedFiles":
		return handleDidChangeWatchedFiles(ctx, s, req, stdout)
	default:
		return handleUnknown(req, stdout)
	}
}

// sendResponse writes a success response to stdout.
func sendResponse(writer io.Writer, id *int, result interface{}) error {
	var resultRaw json.RawMessage
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		resultRaw = json.RawMessage(data)
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultRaw,
	}
	return WriteMessage(writer, resp)
}

// sendError writes an error response to stdout.
func sendError(writer io.Writer, id *int, rpcErr *RPCError) error {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcErr,
	}
	return WriteMessage(writer, resp)
}

// PushDiagnostics sends a textDocument/publishDiagnostics notification to the writer.
func PushDiagnostics(writer io.Writer, params PublishDiagnosticsParams) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal diagnostics params: %w", err)
	}

	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params":  json.RawMessage(data),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal diagnostics notification: %w", err)
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(writer, header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}

// handleInitialize handles the initialize request.
func handleInitialize(s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	var params InitializeParams
	if len(req.Params) > 0 {
		json.Unmarshal(req.Params, &params)
	}

	result, err := s.Initialize(params)
	if err != nil {
		return false, sendError(stdout, req.ID, &RPCError{Code: -32603, Message: err.Error()})
	}

	return false, sendResponse(stdout, req.ID, result)
}

// handleDidOpen handles textDocument/didOpen.
func handleDidOpen(ctx context.Context, s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	var params DidOpenTextDocumentParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			if req.ID != nil {
				return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
			}
			return false, nil
		}
	}

	_, err := s.DidOpen(params)
	if err != nil {
		if req.ID != nil {
			return false, sendError(stdout, req.ID, ErrServerNotInitialized)
		}
		return false, nil
	}

	// Push diagnostics after opening document
	diags := s.ComputeDiagnostics(ctx)
	pushErr := PushDiagnostics(stdout, PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: diags,
	})
	if pushErr != nil {
		return false, pushErr
	}

	if req.ID != nil {
		return false, sendResponse(stdout, req.ID, nil)
	}
	return false, nil
}

// handleDidChange handles textDocument/didChange.
func handleDidChange(ctx context.Context, s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	var params DidChangeTextDocumentParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			if req.ID != nil {
				return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
			}
			return false, nil
		}
	}

	_, err := s.DidChange(params)
	if err != nil {
		if req.ID != nil {
			return false, sendError(stdout, req.ID, ErrServerNotInitialized)
		}
		return false, nil
	}

	// Push diagnostics after content change
	diags := s.ComputeDiagnostics(ctx)
	pushErr := PushDiagnostics(stdout, PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: diags,
	})
	if pushErr != nil {
		return false, pushErr
	}

	if req.ID != nil {
		return false, sendResponse(stdout, req.ID, nil)
	}
	return false, nil
}

// handleDidClose handles textDocument/didClose.
func handleDidClose(s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	var params DidCloseTextDocumentParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			if req.ID != nil {
				return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
			}
			return false, nil
		}
	}

	_, err := s.DidClose(params)
	if err != nil {
		if req.ID != nil {
			return false, sendError(stdout, req.ID, ErrServerNotInitialized)
		}
		return false, nil
	}

	if req.ID != nil {
		return false, sendResponse(stdout, req.ID, nil)
	}
	return false, nil
}

// handleDidSave handles textDocument/didSave.
func handleDidSave(ctx context.Context, s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	// Re-push diagnostics on save
	var params DidSaveTextDocumentParams
	if len(req.Params) > 0 {
		json.Unmarshal(req.Params, &params)
	}

	diags := s.ComputeDiagnostics(ctx)
	PushDiagnostics(stdout, PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: diags,
	})

	if req.ID != nil {
		return false, sendResponse(stdout, req.ID, nil)
	}
	return false, nil
}

// handleDidChangeWatchedFiles handles workspace/didChangeWatchedFiles.
func handleDidChangeWatchedFiles(ctx context.Context, s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	var params DidChangeWatchedFilesParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			if req.ID != nil {
				return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
			}
			return false, nil
		}
	}

	// If arx.yaml changed, re-check all open documents
	configChanged := false
	for _, change := range params.Changes {
		if stringsSuffix(change.URI, "arx.yaml") || stringsSuffix(change.URI, "arx.yml") {
			configChanged = true
			break
		}
	}

	if configChanged {
		diags := s.ComputeDiagnostics(ctx)
		// Re-push diagnostics for all open documents
		for uri := range s.GetAllDocuments() {
			PushDiagnostics(stdout, PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: diags,
			})
		}
	}

	if req.ID != nil {
		return false, sendResponse(stdout, req.ID, nil)
	}
	return false, nil
}

// handleCodeAction handles textDocument/codeAction.
func handleCodeAction(s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	if req.ID == nil {
		return false, nil
	}

	if !s.IsInitialized() || s.IsShutdown() {
		return false, sendError(stdout, req.ID, ErrServerNotInitialized)
	}

	var params CodeActionParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
		}
	}

	actions := ComputeCodeActions(s, params)
	return false, sendResponse(stdout, req.ID, actions)
}

// handleHover handles textDocument/hover.
func handleHover(s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	if req.ID == nil {
		return false, nil
	}

	if !s.IsInitialized() || s.IsShutdown() {
		return false, sendError(stdout, req.ID, ErrServerNotInitialized)
	}

	var params HoverParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false, sendError(stdout, req.ID, &RPCError{Code: -32700, Message: "Parse error"})
		}
	}

	hover := ComputeHover(s, params)
	return false, sendResponse(stdout, req.ID, hover)
}

// handleShutdown handles the shutdown request.
func handleShutdown(s *Server, req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	if req.ID == nil {
		return false, nil
	}

	err := s.Shutdown()
	if err != nil {
		return false, sendError(stdout, req.ID, &RPCError{Code: -32603, Message: err.Error()})
	}

	return false, sendResponse(stdout, req.ID, nil)
}

// handleExit handles the exit notification. Returns stop=true so Run exits.
func handleExit(s *Server, req *JSONRPCRequest) (bool, error) {
	_ = s.Exit()
	return true, nil
}

// handleUnknown handles an unknown method.
func handleUnknown(req *JSONRPCRequest, stdout io.Writer) (bool, error) {
	if req.ID == nil {
		return false, nil // notification — ignore
	}
	return false, sendError(stdout, req.ID, ErrMethodNotFound)
}

// stringsSuffix is a simple suffix check without importing strings.
func stringsSuffix(s, suffix string) bool {
	if len(suffix) > len(s) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
