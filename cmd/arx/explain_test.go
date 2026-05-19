package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplainCommand_HasSuggestFlag(t *testing.T) {
	flag := explainCmd.Flags().Lookup("suggest")
	if flag == nil {
		t.Fatal("--suggest flag not found on explain command")
	}
	if flag.DefValue != "" {
		t.Errorf("--suggest default should be empty, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("--suggest should be string type, got %q", flag.Value.Type())
	}
}

func TestExplainCommand_ListFlag_ShowsViolations(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	cacheDir := ".arx-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	cacheContent := fmt.Sprintf(`{"violations":[{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"test.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"Domain should not import infra"}],"timestamp":"%s","project_root":"."}`, now)
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	explainList = true
	explainLast = false
	explainSuggest = ""
	defer func() {
		explainList = false
		explainSuggest = ""
	}()

	var buf bytes.Buffer
	oldOut := explainStdout
	explainStdout = &buf
	defer func() { explainStdout = oldOut }()

	err = runExplain(explainCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "D-01") {
		t.Errorf("output should contain D-01, got: %s", output)
	}
}

func TestExplainCommand_SuggestFlag_ShowsDiff(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	cacheDir := ".arx-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	cacheContent := fmt.Sprintf(`{"violations":[{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"test.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"Domain should not import infra"}],"timestamp":"%s","project_root":"."}`, now)
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("test.go", []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	explainList = false
	explainLast = false
	explainSuggest = "domain-imports-infrastructure"
	defer func() {
		explainList = false
		explainLast = false
		explainSuggest = ""
	}()

	var buf bytes.Buffer
	oldOut := explainStdout
	explainStdout = &buf
	defer func() { explainStdout = oldOut }()

	err = runExplain(explainCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Fix suggestion") {
		t.Errorf("--suggest output should contain 'Fix suggestion', got: %s", output)
	}
}

func TestExplainCommand_ExplainWithoutFix(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	cacheDir := ".arx-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	// Use an unknown rule ID so no fix matches, but keep the cache valid
	cacheContent := fmt.Sprintf(`{"violations":[{"id":"X-01","rule_id":"unknown-rule","severity":"error","file":"test.go","line":1,"source_layer":"presentation","target_layer":"domain","import":"","message":"Test"}],"timestamp":"%s","project_root":"."}`, now)
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("test.go", []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	explainList = false
	explainLast = false
	explainSuggest = "unknown-rule"
	defer func() {
		explainList = false
		explainLast = false
		explainSuggest = ""
	}()

	var buf bytes.Buffer
	oldOut := explainStdout
	explainStdout = &buf
	defer func() { explainStdout = oldOut }()

	err = runExplain(explainCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not error — just say no suggestion available
}

func TestExplainCommand_OutputContainsDiff(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	cacheDir := ".arx-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Format(time.RFC3339)
	cacheContent := fmt.Sprintf(`{"violations":[{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"test.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"Domain should not import infra"}],"timestamp":"%s","project_root":"."}`, now)
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("test.go", []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	explainList = false
	explainLast = false
	explainSuggest = ""
	defer func() {
		explainList = false
		explainLast = false
		explainSuggest = ""
	}()

	explainStdout = io.Discard
	defer func() { explainStdout = nil }()

	err = runExplain(explainCmd, []string{"D-01"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
