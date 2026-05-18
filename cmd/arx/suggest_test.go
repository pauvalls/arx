package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/infrastructure/output"
)

func TestSuggestCommand_IsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"suggest"})
	if err != nil {
		t.Fatal("suggest command not found on rootCmd")
	}
	if cmd.Use != "suggest [violation-id]" {
		t.Errorf("expected use 'suggest [violation-id]', got %q", cmd.Use)
	}
}

func TestSuggestCommand_HasApplyFlag(t *testing.T) {
	flag := suggestCmd.Flags().Lookup("apply")
	if flag == nil {
		t.Fatal("--apply flag not found on suggest command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--apply default should be false, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--apply should be bool type, got %q", flag.Value.Type())
	}
}

func TestSuggestCommand_HasForceFlag(t *testing.T) {
	flag := suggestCmd.Flags().Lookup("force")
	if flag == nil {
		t.Fatal("--force flag not found on suggest command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--force default should be false, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--force should be bool type, got %q", flag.Value.Type())
	}
}

func TestSuggestCommand_HasOutputFlag(t *testing.T) {
	flag := suggestCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("--output flag not found on suggest command")
	}
	if flag.DefValue != "" {
		t.Errorf("--output default should be empty, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("--output should be string type, got %q", flag.Value.Type())
	}
}

func TestSuggestCommand_HasShortOutputFlag(t *testing.T) {
	flag := suggestCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("--output flag not found")
	}
	if flag.Shorthand != "o" {
		t.Errorf("--output shorthand should be 'o', got %q", flag.Shorthand)
	}
}

func TestCachedToDomain_ConvertsFields(t *testing.T) {
	cached := []output.CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-imports-infrastructure",
			File:        "internal/domain/order.go",
			Line:        5,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "Domain layer should not import infrastructure",
		},
	}

	result := cachedToDomain(cached)

	if len(result) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result))
	}

	v := result[0]
	if v.ID != "D-01" {
		t.Errorf("expected ID 'D-01', got %q", v.ID)
	}
	if v.RuleID != "domain-imports-infrastructure" {
		t.Errorf("expected RuleID 'domain-imports-infrastructure', got %q", v.RuleID)
	}
	if v.File != "internal/domain/order.go" {
		t.Errorf("expected File 'internal/domain/order.go', got %q", v.File)
	}
	if v.Line != 5 {
		t.Errorf("expected Line 5, got %d", v.Line)
	}
	if v.SourceLayer != "domain" {
		t.Errorf("expected SourceLayer 'domain', got %q", v.SourceLayer)
	}
	if v.TargetLayer != "infrastructure" {
		t.Errorf("expected TargetLayer 'infrastructure', got %q", v.TargetLayer)
	}
}

func TestCachedToDomain_PreservesAllViolations(t *testing.T) {
	cached := []output.CachedViolation{
		{ID: "D-01", RuleID: "rule-a", File: "a.go"},
		{ID: "D-02", RuleID: "rule-b", File: "b.go"},
		{ID: "D-03", RuleID: "rule-c", File: "c.go"},
	}

	result := cachedToDomain(cached)

	if len(result) != 3 {
		t.Fatalf("expected 3 violations, got %d", len(result))
	}
	for i, v := range result {
		if v.ID != cached[i].ID {
			t.Errorf("violation %d: expected ID %q, got %q", i, cached[i].ID, v.ID)
		}
	}
}

func TestBackupDirFor_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	dir, err := backupDirFor(tmpDir)
	if err != nil {
		t.Fatalf("backupDirFor failed: %v", err)
	}

	expected := filepath.Join(tmpDir, ".arx-backup")
	if dir != expected {
		t.Errorf("expected dir %q, got %q", expected, dir)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("backup directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected backup path to be a directory")
	}
}

func TestBackupDirFor_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	first, err := backupDirFor(tmpDir)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	second, err := backupDirFor(tmpDir)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if first != second {
		t.Errorf("expected same path, got %q and %q", first, second)
	}
}

func TestSuggestCommand_RunWithoutCache_ReturnsError(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	os.RemoveAll(".arx-cache")

	// Reset flags
	suggestApply = false
	suggestForce = false
	suggestOutput = ""

	err = runSuggest(suggestCmd, nil)
	if err == nil {
		t.Fatal("expected error when no cache exists, got nil")
	}
	if !strings.Contains(err.Error(), "no violations found") {
		t.Errorf("expected 'no violations found' error, got: %v", err)
	}
}

func TestSuggestCommand_UnknownViolationID_ReturnsError(t *testing.T) {
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
	cacheContent := `{"violations":[{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"test.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"test"}],"timestamp":"` + timeNow().Format(time.RFC3339) + `","project_root":"."}`
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	suggestApply = false
	suggestForce = false
	suggestOutput = ""

	err = runSuggest(suggestCmd, []string{"D-99"})
	if err == nil {
		t.Fatal("expected error for unknown violation ID, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestSuggestCommand_OutputFlag_WritesToFile(t *testing.T) {
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
	cacheContent := `{"violations":[{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"test.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"test"}],"timestamp":"` + timeNow().Format(time.RFC3339) + `","project_root":"."}`
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("test.go", []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(tmpDir, "diff.patch")
	suggestOutput = outputFile
	suggestApply = false
	suggestForce = false
	defer func() { suggestOutput = "" }()

	var buf bytes.Buffer
	suggestCmd.SetOut(&buf)
	suggestCmd.SetErr(io.Discard)

	err = runSuggest(suggestCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "D-01") {
		t.Errorf("expected output to contain 'D-01', got: %s", content)
	}
	if !strings.Contains(content, "fix suggestion(s) generated") {
		t.Errorf("expected summary in output file, got: %s", content)
	}
}

func TestSuggestCommand_OutputWithSpecificViolation(t *testing.T) {
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
	cacheContent := `{"violations":[
		{"id":"D-01","rule_id":"domain-imports-infrastructure","severity":"error","file":"a.go","line":1,"source_layer":"domain","target_layer":"infrastructure","import":"pkg/infra","message":"test"},
		{"id":"D-02","rule_id":"application-imports-infrastructure","severity":"error","file":"b.go","line":2,"source_layer":"application","target_layer":"infrastructure","import":"pkg/db","message":"test2"}
	],"timestamp":"` + timeNow().Format(time.RFC3339) + `","project_root":"."}`
	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("a.go", []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("b.go", []byte("package b\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputFile := filepath.Join(tmpDir, "diff.patch")
	suggestOutput = outputFile
	suggestApply = false
	suggestForce = false
	defer func() { suggestOutput = "" }()

	err = runSuggest(suggestCmd, []string{"D-02"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "D-02") {
		t.Errorf("expected output to contain 'D-02', got: %s", content)
	}
	if strings.Contains(content, "D-01") {
		t.Errorf("expected output to NOT contain 'D-01' (specific violation requested), got: %s", content)
	}
}

// timeNow returns the current time — extracted for test mocking.
func timeNow() time.Time {
	return time.Now()
}
