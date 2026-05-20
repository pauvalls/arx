package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPosition_JSONRoundTrip(t *testing.T) {
	p := Position{Line: 5, Character: 12}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal(Position) error = %v", err)
	}
	var got Position
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Position) error = %v", err)
	}
	if got.Line != 5 || got.Character != 12 {
		t.Errorf("Position round-trip = %+v, want {Line:5 Character:12}", got)
	}
}

func TestRange_JSONRoundTrip(t *testing.T) {
	r := Range{
		Start: Position{Line: 0, Character: 1},
		End:   Position{Line: 0, Character: 20},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal(Range) error = %v", err)
	}
	var got Range
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Range) error = %v", err)
	}
	if got.Start.Line != 0 || got.Start.Character != 1 {
		t.Errorf("Range.Start round-trip = %+v, want {Line:0 Character:1}", got.Start)
	}
	if got.End.Line != 0 || got.End.Character != 20 {
		t.Errorf("Range.End round-trip = %+v, want {Line:0 Character:20}", got.End)
	}
}

func TestDiagnostic_JSONRoundTrip(t *testing.T) {
	d := Diagnostic{
		Range: Range{
			Start: Position{Line: 10, Character: 0},
			End:   Position{Line: 10, Character: 40},
		},
		Severity: DSError,
		Code:     "domain-imports-infrastructure",
		Source:   "arx",
		Message:  "Domain layer must not import infrastructure packages",
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal(Diagnostic) error = %v", err)
	}
	var got Diagnostic
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Diagnostic) error = %v", err)
	}
	if got.Severity != DSError {
		t.Errorf("Diagnostic.Severity = %d, want %d", got.Severity, DSError)
	}
	if got.Code != "domain-imports-infrastructure" {
		t.Errorf("Diagnostic.Code = %q, want %q", got.Code, "domain-imports-infrastructure")
	}
	if got.Source != "arx" {
		t.Errorf("Diagnostic.Source = %q, want %q", got.Source, "arx")
	}
	if got.Message != "Domain layer must not import infrastructure packages" {
		t.Errorf("Diagnostic.Message = %q, want %q", got.Message, "Domain layer must not import infrastructure packages")
	}
}

func TestDiagnosticSeverity_Constants(t *testing.T) {
	tests := []struct {
		sev  DiagnosticSeverity
		want int
	}{
		{DSError, 1},
		{DSWarning, 2},
		{DSInfo, 3},
		{DSHint, 4},
	}
	for _, tt := range tests {
		if int(tt.sev) != tt.want {
			t.Errorf("DiagnosticSeverity(%d) = %d, want %d", tt.sev, tt.sev, tt.want)
		}
	}
}

func TestJSONRPCRequest_Notification(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  json.RawMessage(`{"textDocument":{"uri":"file:///test.go","languageId":"go","version":1,"text":"package main"}}`),
	}
	if req.ID != nil {
		t.Error("notification request should have nil ID")
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(notification) error = %v", err)
	}
	var got JSONRPCRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(notification) error = %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.Method != "textDocument/didOpen" {
		t.Errorf("Method = %q, want %q", got.Method, "textDocument/didOpen")
	}
	if got.ID != nil {
		t.Error("notification should have nil ID after round-trip")
	}
}

func TestJSONRPCRequest_Request(t *testing.T) {
	id := 1
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(request) error = %v", err)
	}
	var got JSONRPCRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(request) error = %v", err)
	}
	if got.ID == nil {
		t.Fatal("request should have non-nil ID after round-trip")
	}
	if *got.ID != 1 {
		t.Errorf("ID = %d, want %d", *got.ID, 1)
	}
}

func TestJSONRPCResponse_Result(t *testing.T) {
	id := 1
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(`{"capabilities":{}}`),
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(response) error = %v", err)
	}
	var got JSONRPCResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, "2.0")
	}
	if got.ID == nil || *got.ID != 1 {
		t.Errorf("ID = %v, want 1", got.ID)
	}
	if got.Error != nil {
		t.Errorf("unexpected error: %+v", got.Error)
	}
}

func TestJSONRPCResponse_Error(t *testing.T) {
	id := 1
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Error: &RPCError{
			Code:    -32601,
			Message: "Method not found",
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal(error response) error = %v", err)
	}
	var got JSONRPCResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(error response) error = %v", err)
	}
	if got.Error == nil {
		t.Fatal("expected Error to be non-nil")
	}
	if got.Error.Code != -32601 {
		t.Errorf("Error.Code = %d, want %d", got.Error.Code, -32601)
	}
	if got.Error.Message != "Method not found" {
		t.Errorf("Error.Message = %q, want %q", got.Error.Message, "Method not found")
	}
}

func TestRPCError_Constants(t *testing.T) {
	if ErrParse.Code != -32700 || ErrParse.Message != "Parse error" {
		t.Errorf("ErrParse = %+v", ErrParse)
	}
	if ErrInvalidRequest.Code != -32600 || ErrInvalidRequest.Message != "Invalid Request" {
		t.Errorf("ErrInvalidRequest = %+v", ErrInvalidRequest)
	}
	if ErrMethodNotFound.Code != -32601 || ErrMethodNotFound.Message != "Method not found" {
		t.Errorf("ErrMethodNotFound = %+v", ErrMethodNotFound)
	}
	if ErrServerNotInitialized.Code != -32002 || ErrServerNotInitialized.Message != "Server not initialized" {
		t.Errorf("ErrServerNotInitialized = %+v", ErrServerNotInitialized)
	}
}

func TestReadMessage_SimpleRequest(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	raw := "Content-Length: " + intToStr(len(body)) + "\r\n\r\n" + body
	reader := bufio.NewReader(strings.NewReader(raw))

	req, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if req.Method != "initialize" {
		t.Errorf("Method = %q, want %q", req.Method, "initialize")
	}
	if req.ID == nil || *req.ID != 1 {
		t.Errorf("ID = %v, want 1", req.ID)
	}
}

func TestReadMessage_Notification(t *testing.T) {
	body := `{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{}}`
	raw := "Content-Length: " + intToStr(len(body)) + "\r\n\r\n" + body
	reader := bufio.NewReader(strings.NewReader(raw))

	req, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if req.Method != "textDocument/didOpen" {
		t.Errorf("Method = %q, want %q", req.Method, "textDocument/didOpen")
	}
	if req.ID != nil {
		t.Error("notification should have nil ID")
	}
}

func TestReadMessage_MissingContentLength(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("invalid header\r\n\r\n{}"))
	_, err := ReadMessage(reader)
	if err == nil {
		t.Fatal("expected error for missing Content-Length header")
	}
}

func TestReadMessage_InvalidContentLength(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("Content-Length: abc\r\n\r\n{}"))
	_, err := ReadMessage(reader)
	if err == nil {
		t.Fatal("expected error for invalid Content-Length value")
	}
}

func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	id := 1
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(`{"capabilities":{}}`),
	}

	err := WriteMessage(&buf, resp)
	if err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(output, "Content-Length: ") {
		t.Errorf("output should start with Content-Length header, got %q", output)
	}
	if !strings.Contains(output, "\r\n\r\n") {
		t.Errorf("output should contain header/body separator")
	}
	if !strings.Contains(output, `"jsonrpc":"2.0"`) {
		t.Errorf("output should contain JSON-RPC body")
	}
}

func TestWriteMessage_RoundTrip(t *testing.T) {
	id := 1
	original := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  json.RawMessage(`{"capabilities":{}}`),
	}

	var buf bytes.Buffer
	if err := WriteMessage(&buf, original); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}

	// Now read it back
	req, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage() after WriteMessage error = %v", err)
	}
	_ = req
}

func TestReadMessage_MultipleMessages(t *testing.T) {
	body1 := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	body2 := `{"jsonrpc":"2.0","id":2,"method":"shutdown"}`
	raw := "Content-Length: " + intToStr(len(body1)) + "\r\n\r\n" + body1 +
		"Content-Length: " + intToStr(len(body2)) + "\r\n\r\n" + body2
	// Use a single bufio.Reader for both messages to preserve buffered data
	reader := bufio.NewReader(strings.NewReader(raw))

	req1, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage() first msg error = %v", err)
	}
	if req1.Method != "initialize" {
		t.Errorf("First message Method = %q, want %q", req1.Method, "initialize")
	}

	req2, err := ReadMessage(reader)
	if err != nil {
		t.Fatalf("ReadMessage() second msg error = %v", err)
	}
	if req2.Method != "shutdown" {
		t.Errorf("Second message Method = %q, want %q", req2.Method, "shutdown")
	}
}

func TestInitializeParams_JSON(t *testing.T) {
	params := InitializeParams{
		ProcessID: intPtr(12345),
		RootURI:   "file:///home/user/project",
		Capabilities: ClientCapabilities{
			TextDocument: &TextDocumentClientCapabilities{
				Synchronization: &SynchronizationCapabilities{
					DidSave: true,
				},
			},
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(InitializeParams) error = %v", err)
	}
	var got InitializeParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(InitializeParams) error = %v", err)
	}
	if got.RootURI != "file:///home/user/project" {
		t.Errorf("RootURI = %q, want %q", got.RootURI, "file:///home/user/project")
	}
	if got.ProcessID == nil || *got.ProcessID != 12345 {
		t.Errorf("ProcessID = %v, want 12345", got.ProcessID)
	}
}

func TestInitializeResult_ServerCapabilities(t *testing.T) {
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:   TDSyncKindFull,
			CodeActionProvider: true,
			HoverProvider:      true,
			DiagnosticProvider: &DiagnosticRegistrationOptions{
				InterFileDependencies: true,
				WorkspaceDiagnostics:  false,
			},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(InitializeResult) error = %v", err)
	}
	var got InitializeResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(InitializeResult) error = %v", err)
	}
	if got.Capabilities.TextDocumentSync != TDSyncKindFull {
		t.Errorf("TextDocumentSync = %d, want %d", got.Capabilities.TextDocumentSync, TDSyncKindFull)
	}
	if !got.Capabilities.CodeActionProvider {
		t.Error("CodeActionProvider should be true")
	}
	if !got.Capabilities.HoverProvider {
		t.Error("HoverProvider should be true")
	}
	if got.Capabilities.DiagnosticProvider == nil {
		t.Fatal("DiagnosticProvider should be non-nil")
	}
	if !got.Capabilities.DiagnosticProvider.InterFileDependencies {
		t.Error("InterFileDependencies should be true")
	}
	if got.Capabilities.DiagnosticProvider.WorkspaceDiagnostics {
		t.Error("WorkspaceDiagnostics should be false")
	}
}

func TestDidOpenTextDocumentParams_JSON(t *testing.T) {
	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        "file:///test.go",
			LanguageID: "go",
			Version:    1,
			Text:       "package main\nfunc main() {}\n",
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(DidOpenTextDocumentParams) error = %v", err)
	}
	var got DidOpenTextDocumentParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(DidOpenTextDocumentParams) error = %v", err)
	}
	if got.TextDocument.URI != "file:///test.go" {
		t.Errorf("URI = %q, want %q", got.TextDocument.URI, "file:///test.go")
	}
	if got.TextDocument.LanguageID != "go" {
		t.Errorf("LanguageID = %q, want %q", got.TextDocument.LanguageID, "go")
	}
}

func TestPublishDiagnosticsParams_JSON(t *testing.T) {
	params := PublishDiagnosticsParams{
		URI: "file:///test.go",
		Diagnostics: []Diagnostic{
			{
				Range: Range{
					Start: Position{Line: 5, Character: 0},
					End:   Position{Line: 5, Character: 30},
				},
				Severity: DSError,
				Code:     "E001",
				Source:   "arx",
				Message:  "test violation",
			},
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(PublishDiagnosticsParams) error = %v", err)
	}
	var got PublishDiagnosticsParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(PublishDiagnosticsParams) error = %v", err)
	}
	if got.URI != "file:///test.go" {
		t.Errorf("URI = %q, want %q", got.URI, "file:///test.go")
	}
	if len(got.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(got.Diagnostics))
	}
	if got.Diagnostics[0].Code != "E001" {
		t.Errorf("Diagnostic.Code = %q, want %q", got.Diagnostics[0].Code, "E001")
	}
}

func TestCodeActionParams_JSON(t *testing.T) {
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
					Code:   "E001",
					Source: "arx",
				},
			},
			Only: []CodeActionKind{CodeActionQuickFix},
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(CodeActionParams) error = %v", err)
	}
	var got CodeActionParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(CodeActionParams) error = %v", err)
	}
	if got.TextDocument.URI != "file:///test.go" {
		t.Errorf("URI = %q, want %q", got.TextDocument.URI, "file:///test.go")
	}
	if len(got.Context.Diagnostics) != 1 {
		t.Fatalf("got %d context diagnostics, want 1", len(got.Context.Diagnostics))
	}
}

func TestCodeAction_JSON(t *testing.T) {
	action := CodeAction{
		Title: "Apply arx fix: test",
		Kind:  CodeActionQuickFix,
		Diagnostics: []Diagnostic{
			{
				Range: Range{
					Start: Position{Line: 5, Character: 0},
					End:   Position{Line: 5, Character: 30},
				},
				Code: "E001",
			},
		},
		Command: &Command{
			Title:     "Apply arx fix",
			Command:   "arx.fix",
			Arguments: []json.RawMessage{json.RawMessage(`"E001"`)},
		},
	}
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("json.Marshal(CodeAction) error = %v", err)
	}
	var got CodeAction
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(CodeAction) error = %v", err)
	}
	if got.Title != "Apply arx fix: test" {
		t.Errorf("Title = %q, want %q", got.Title, "Apply arx fix: test")
	}
	if got.Kind != CodeActionQuickFix {
		t.Errorf("Kind = %q, want %q", got.Kind, CodeActionQuickFix)
	}
	if got.Command == nil {
		t.Fatal("Command should be non-nil")
	}
	if got.Command.Command != "arx.fix" {
		t.Errorf("Command = %q, want %q", got.Command.Command, "arx.fix")
	}
}

func TestHover_JSON(t *testing.T) {
	h := Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: "**Layer**: domain\n\n**Rule**: D-01: Cannot import infrastructure",
		},
		Range: &Range{
			Start: Position{Line: 5, Character: 0},
			End:   Position{Line: 5, Character: 30},
		},
	}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal(Hover) error = %v", err)
	}
	var got Hover
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Hover) error = %v", err)
	}
	if got.Contents.Kind != "markdown" {
		t.Errorf("Contents.Kind = %q, want %q", got.Contents.Kind, "markdown")
	}
	if got.Contents.Value != "**Layer**: domain\n\n**Rule**: D-01: Cannot import infrastructure" {
		t.Errorf("Contents.Value = %q, want %q", got.Contents.Value, "**Layer**: domain\n\n**Rule**: D-01: Cannot import infrastructure")
	}
	if got.Range == nil {
		t.Fatal("Range should be non-nil")
	}
}

func TestHover_NullRange(t *testing.T) {
	h := Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: "**Layer**: domain",
		},
	}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal(Hover) error = %v", err)
	}
	var got Hover
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Hover) error = %v", err)
	}
	if got.Range != nil {
		t.Error("Range should be nil when not set")
	}
}

func TestDidChangeWatchedFilesParams_JSON(t *testing.T) {
	params := DidChangeWatchedFilesParams{
		Changes: []FileEvent{
			{URI: "file:///arx.yaml", Type: FileChangeTypeChanged},
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(DidChangeWatchedFilesParams) error = %v", err)
	}
	var got DidChangeWatchedFilesParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(DidChangeWatchedFilesParams) error = %v", err)
	}
	if len(got.Changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(got.Changes))
	}
	if got.Changes[0].URI != "file:///arx.yaml" {
		t.Errorf("URI = %q, want %q", got.Changes[0].URI, "file:///arx.yaml")
	}
	if got.Changes[0].Type != FileChangeTypeChanged {
		t.Errorf("Type = %d, want %d", got.Changes[0].Type, FileChangeTypeChanged)
	}
}

func TestDidChangeTextDocumentParams_JSON(t *testing.T) {
	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///test.go"},
			Version:                2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "package main\nfunc main() {}\n"},
		},
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(DidChangeTextDocumentParams) error = %v", err)
	}
	var got DidChangeTextDocumentParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(DidChangeTextDocumentParams) error = %v", err)
	}
	if got.TextDocument.URI != "file:///test.go" {
		t.Errorf("URI = %q, want %q", got.TextDocument.URI, "file:///test.go")
	}
	if got.TextDocument.Version != 2 {
		t.Errorf("Version = %d, want %d", got.TextDocument.Version, 2)
	}
	if len(got.ContentChanges) != 1 {
		t.Fatalf("got %d content changes, want 1", len(got.ContentChanges))
	}
	if got.ContentChanges[0].Text != "package main\nfunc main() {}\n" {
		t.Errorf("ContentChanges[0].Text = %q, want %q", got.ContentChanges[0].Text, "package main\nfunc main() {}\n")
	}
}

func TestServerCapabilities_TextDocumentSyncKind(t *testing.T) {
	tests := []struct {
		kind TextDocumentSyncKind
		want int
	}{
		{TDSyncKindNone, 0},
		{TDSyncKindFull, 1},
		{TDSyncKindIncremental, 2},
	}
	for _, tt := range tests {
		if int(tt.kind) != tt.want {
			t.Errorf("TextDocumentSyncKind(%d) = %d, want %d", tt.kind, tt.kind, tt.want)
		}
	}
}

func TestFileChangeType_Constants(t *testing.T) {
	tests := []struct {
		ft   FileChangeType
		want int
	}{
		{FileChangeTypeCreated, 1},
		{FileChangeTypeChanged, 2},
		{FileChangeTypeDeleted, 3},
	}
	for _, tt := range tests {
		if int(tt.ft) != tt.want {
			t.Errorf("FileChangeType(%d) = %d, want %d", tt.ft, tt.ft, tt.want)
		}
	}
}

// intToStr is a helper used in test setup to convert int to string without fmt import
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// intPtr returns a pointer to an int (used in test setup)
func intPtr(n int) *int {
	return &n
}
