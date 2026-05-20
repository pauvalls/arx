package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lspResponse represents a JSON-RPC message from the LSP server.
type lspResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// writeLSP sends a JSON-RPC message to the writer.
func writeLSP(w io.Writer, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

// readLSP reads a single JSON-RPC message from a buffered reader.
func readLSP(br *bufio.Reader) (*lspResponse, error) {
	var contentLength int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading header: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			val := strings.TrimPrefix(line, "Content-Length: ")
			n := 0
			for _, c := range val {
				if c >= '0' && c <= '9' {
					n = n*10 + int(c-'0')
				}
			}
			contentLength = n
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("no Content-Length header")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(br, body); err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	var resp lspResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &resp, nil
}

func TestLSP_FullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping LSP integration test in short mode")
	}

	// Find a fixture directory with arx.yaml
	fixtureDir, err := findFixtureDir()
	if err != nil {
		t.Skipf("no fixture found: %v", err)
	}
	t.Logf("Using fixture: %s", fixtureDir)

	// Find arx binary
	arxBin := findArxBinary(t)
	t.Logf("Using arx binary: %s", arxBin)

	// Start arx lsp subprocess
	cmd := exec.Command(arxBin, "lsp")
	cmd.Dir = fixtureDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}

	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		cmd.Wait()
	}()

	// Use a single bufio.Reader for ALL reads (no concurrent read goroutines)
	br := bufio.NewReader(stdout)

	// 1. Initialize
	t.Log("Step 1: Initialize")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"rootUri": "file://" + fixtureDir,
		},
	}); err != nil {
		t.Fatalf("write initialize: %v", err)
	}

	resp, err := readLSP(br)
	if err != nil {
		t.Fatalf("read initialize response: %v\nstderr: %s", err, stderr.String())
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %+v", resp.Error)
	}
	if resp.ID == nil || *resp.ID != 1 {
		t.Errorf("initialize response ID = %v, want 1", resp.ID)
	}
	if resp.Result == nil {
		t.Fatal("initialize result should not be nil")
	}

	// Parse capabilities from result
	var result struct {
		Capabilities struct {
			TextDocumentSync   int  `json:"textDocumentSync"`
			CodeActionProvider bool `json:"codeActionProvider"`
			HoverProvider      bool `json:"hoverProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal capabilities: %v", err)
	}
	if result.Capabilities.TextDocumentSync != 1 {
		t.Errorf("TextDocumentSync = %d, want 1 (Full)", result.Capabilities.TextDocumentSync)
	}
	if !result.Capabilities.CodeActionProvider {
		t.Error("CodeActionProvider should be true")
	}
	if !result.Capabilities.HoverProvider {
		t.Error("HoverProvider should be true")
	}
	t.Log("✓ Initialize OK")

	// 2. DidOpen — notification, no response expected
	t.Log("Step 2: DidOpen")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":  "file://" + filepath.Join(fixtureDir, "test_open.go"),
				"text": "package main\n\nfunc main() {}\n",
			},
		},
	}); err != nil {
		t.Fatalf("write didOpen: %v", err)
	}

	// Give the server time to process the notification
	time.Sleep(200 * time.Millisecond)

	// Drain any notifications (diagnostics) from the pipe without goroutine race
	// We use a non-blocking check: set a short read deadline isn't possible on pipes
	// Instead, we just accept any pending non-response messages
	// For now, the server doesn't push diagnostics, so there shouldn't be any
	t.Log("✓ DidOpen OK")

	// 3. CodeAction
	t.Log("Step 3: CodeAction")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "textDocument/codeAction",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file://" + filepath.Join(fixtureDir, "test_open.go"),
			},
			"range": map[string]interface{}{
				"start": map[string]interface{}{"line": 0, "character": 0},
				"end":   map[string]interface{}{"line": 0, "character": 10},
			},
			"context": map[string]interface{}{
				"diagnostics": []interface{}{},
			},
		},
	}); err != nil {
		t.Fatalf("write codeAction: %v", err)
	}

	resp, err = readLSP(br)
	if err != nil {
		t.Fatalf("read codeAction response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("codeAction error: %+v", resp.Error)
	}
	if resp.ID == nil || *resp.ID != 3 {
		t.Errorf("codeAction response ID = %v, want 3", resp.ID)
	}
	t.Log("✓ CodeAction OK")

	// 4. Hover
	t.Log("Step 4: Hover")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "textDocument/hover",
		"params": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": "file://" + filepath.Join(fixtureDir, "test_open.go"),
			},
			"position": map[string]interface{}{
				"line":      0,
				"character": 0,
			},
		},
	}); err != nil {
		t.Fatalf("write hover: %v", err)
	}

	resp, err = readLSP(br)
	if err != nil {
		t.Fatalf("read hover response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("hover error: %+v", resp.Error)
	}
	if resp.ID == nil || *resp.ID != 4 {
		t.Errorf("hover response ID = %v, want 4", resp.ID)
	}
	t.Log("✓ Hover OK")

	// 5. Shutdown
	t.Log("Step 5: Shutdown")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "shutdown",
	}); err != nil {
		t.Fatalf("write shutdown: %v", err)
	}

	resp, err = readLSP(br)
	if err != nil {
		t.Fatalf("read shutdown response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("shutdown error: %+v", resp.Error)
	}
	if resp.ID == nil || *resp.ID != 5 {
		t.Errorf("shutdown response ID = %v, want 5", resp.ID)
	}
	t.Log("✓ Shutdown OK")

	// 6. Exit
	t.Log("Step 6: Exit")
	if err := writeLSP(stdin, map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "exit",
	}); err != nil {
		t.Fatalf("write exit: %v", err)
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Process exited: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit after 5s")
	}

	t.Log("✓ LSP integration test PASSED")
}

// findArxBinary locates the arx binary, building if needed.
func findArxBinary(t *testing.T) string {
	t.Helper()

	projectRoot := findProjectRoot()
	candidates := []string{
		filepath.Join(projectRoot, "arx"),
		filepath.Join(projectRoot, "bin", "arx"),
		filepath.Join(os.Getenv("HOME"), "go", "bin", "arx"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Build it
	t.Log("Building arx binary...")
	binPath := filepath.Join(projectRoot, "arx")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/arx")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build arx: %v\n%s", err, out)
	}
	return binPath
}

// findProjectRoot finds the arx project root by looking for go.mod.
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
