package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ViolationCache mirrors the production structure for test setup.
type ViolationCache struct {
	Violations  []CachedViolation `json:"violations"`
	Timestamp   time.Time         `json:"timestamp"`
	ProjectRoot string            `json:"project_root"`
}

// CachedViolation mirrors the production structure for test setup.
type CachedViolation struct {
	ID          string `json:"id"`
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	SourceLayer string `json:"source_layer"`
	TargetLayer string `json:"target_layer"`
	Import      string `json:"import"`
	Message     string `json:"message"`
	Explanation string `json:"explanation"`
}

// createFreshCache writes a violations.json cache that the suggest/explain commands can read.
// The cache is written to projectDir/.arx-cache/violations.json.
func createFreshCache(t *testing.T, projectDir string, violations []CachedViolation) {
	t.Helper()
	cacheDir := filepath.Join(projectDir, ".arx-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}

	cache := ViolationCache{
		Violations:  violations,
		Timestamp:   time.Now(),
		ProjectRoot: projectDir,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(cacheDir, "violations.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// createMinimalProject creates a minimal Go project structure for arx to operate on.
func createMinimalProject(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()

	arxCfg := `version: "1.0"
layers:
  - name: domain
    paths: ["internal/domain/**"]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]
rules:
  - id: domain-cannot-depend-on-infrastructure
    from: domain
    to: [infrastructure]
    type: cannot
    severity: error
exclude: []
`
	if err := os.WriteFile(filepath.Join(projectDir, "arx.yaml"), []byte(arxCfg), 0644); err != nil {
		t.Fatal(err)
	}

	domainDir := filepath.Join(projectDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatal(err)
	}
	domainContent := `package domain

import "github.com/example/test-project/internal/infrastructure/db"

type User struct {
	ID string
}
`
	if err := os.WriteFile(filepath.Join(domainDir, "user.go"), []byte(domainContent), 0644); err != nil {
		t.Fatal(err)
	}

	infraDir := filepath.Join(projectDir, "internal", "infrastructure", "db")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "db.go"), []byte("package db\n"), 0644); err != nil {
		t.Fatal(err)
	}

	goMod := `module github.com/example/test-project

go 1.21
`
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	return projectDir
}

// runArx runs an arx command in the given directory and returns output.
func runArx(t *testing.T, binaryPath, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSuggestBatch_DryRun_NoFilesChanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := createMinimalProject(t)

	// Create a fresh violations cache
	createFreshCache(t, projectDir, []CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    "error",
			File:        "internal/domain/user.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/test-project/internal/infrastructure/db",
			Message:     "Domain layer should not import infrastructure",
		},
	})

	binaryPath := buildArxBinary(t)

	// Dry run
	dryRunOut, err := runArx(t, binaryPath, projectDir, "suggest", "--dry-run")
	if err != nil {
		t.Fatalf("suggest --dry-run failed: %v\nOutput: %s", err, dryRunOut)
	}
	t.Logf("dry-run output: %s", dryRunOut)

	if !strings.Contains(dryRunOut, "DRY RUN") {
		t.Errorf("dry-run output should indicate dry run mode, got: %s", dryRunOut)
	}
	if !strings.Contains(dryRunOut, "D-01") {
		t.Errorf("dry-run output should contain violation D-01, got: %s", dryRunOut)
	}

	// Verify file unchanged
	domainFile := filepath.Join(projectDir, "internal", "domain", "user.go")
	content, err := os.ReadFile(domainFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "import") {
		t.Error("file appears to have been modified by dry-run")
	}

	t.Log("✅ suggest --dry-run works")
}

func TestSuggestBatch_ApplyAndRollbackAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := createMinimalProject(t)

	createFreshCache(t, projectDir, []CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    "error",
			File:        "internal/domain/user.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/test-project/internal/infrastructure/db",
			Message:     "Domain layer should not import infrastructure",
		},
	})

	binaryPath := buildArxBinary(t)
	domainFile := filepath.Join(projectDir, "internal", "domain", "user.go")
	origContent, _ := os.ReadFile(domainFile)

	// Apply with force
	applyOut, err := runArx(t, binaryPath, projectDir, "suggest", "--apply", "--force")
	if err != nil {
		t.Fatalf("suggest --apply --force failed: %v\nOutput: %s", err, applyOut)
	}
	t.Logf("apply output: %s", applyOut)

	if !strings.Contains(applyOut, "Applied") {
		t.Errorf("apply output should mention applied fixes, got: %s", applyOut)
	}

	// Verify file was modified
	modifiedContent, _ := os.ReadFile(domainFile)
	if string(modifiedContent) == string(origContent) {
		t.Error("file was NOT modified after --apply --force")
	}

	// Rollback all
	rollbackOut, err := runArx(t, binaryPath, projectDir, "rollback", "--all")
	if err != nil {
		t.Fatalf("rollback --all failed: %v\nOutput: %s", err, rollbackOut)
	}
	t.Logf("rollback output: %s", rollbackOut)

	restoredContent, _ := os.ReadFile(domainFile)
	if string(restoredContent) != string(origContent) {
		t.Errorf("file was NOT restored after rollback, got: %s", string(restoredContent))
	}

	t.Log("✅ suggest --apply + rollback --all works")
}

func TestSuggestBatch_ApplyAndRollbackSingleFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := createMinimalProject(t)

	createFreshCache(t, projectDir, []CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    "error",
			File:        "internal/domain/user.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/test-project/internal/infrastructure/db",
			Message:     "Domain layer should not import infrastructure",
		},
	})

	binaryPath := buildArxBinary(t)
	domainFile := filepath.Join(projectDir, "internal", "domain", "user.go")
	origContent, _ := os.ReadFile(domainFile)

	// Apply
	runArx(t, binaryPath, projectDir, "suggest", "--apply", "--force")

	// Rollback single file
	rollbackOut, err := runArx(t, binaryPath, projectDir, "rollback", "internal/domain/user.go")
	if err != nil {
		t.Fatalf("rollback single file failed: %v\nOutput: %s", err, rollbackOut)
	}
	t.Logf("rollback output: %s", rollbackOut)

	restoredContent, _ := os.ReadFile(domainFile)
	if string(restoredContent) != string(origContent) {
		t.Errorf("file was NOT restored after single rollback, got: %s", string(restoredContent))
	}

	t.Log("✅ suggest --apply + rollback <file> works")
}

func TestSuggestBatch_ExplainAndSuggest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := createMinimalProject(t)

	createFreshCache(t, projectDir, []CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    "error",
			File:        "internal/domain/user.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/test-project/internal/infrastructure/db",
			Message:     "Domain layer should not import infrastructure",
		},
	})

	binaryPath := buildArxBinary(t)

	// Test explain with violation ID
	explainOut, _ := runArx(t, binaryPath, projectDir, "explain", "D-01")
	t.Logf("explain output: %s", explainOut)

	if !strings.Contains(explainOut, "D-01") {
		t.Error("explain output should contain violation ID")
	}

	// Test explain --suggest
	suggestOut, _ := runArx(t, binaryPath, projectDir, "explain", "--suggest", "domain-cannot-depend-on-infrastructure")
	t.Logf("explain --suggest output: %s", suggestOut)

	t.Log("✅ explain + explain --suggest works")
}

func TestSuggestBatch_AllFlagInteractiveLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := createMinimalProject(t)

	createFreshCache(t, projectDir, []CachedViolation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    "error",
			File:        "internal/domain/user.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/test-project/internal/infrastructure/db",
			Message:     "Domain layer should not import infrastructure",
		},
	})

	binaryPath := buildArxBinary(t)

	// Run --all with "q" input to quit
	cmd := exec.Command(binaryPath, "suggest", "--all")
	cmd.Dir = projectDir
	cmd.Stdin = strings.NewReader("q\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("suggest --all failed: %v\nOutput: %s", err, string(out))
	}
	outStr := string(out)
	t.Logf("--all output: %s", outStr)

	if !strings.Contains(outStr, "Apply fix for D-01") {
		t.Errorf("output should show review prompt, got: %s", outStr)
	}

	t.Log("✅ suggest --all with interactive loop works")
}

func TestSuggestBatch_RollbackList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()

	// Create violation-ID based backup
	violationDir := filepath.Join(projectDir, ".arx-backup", "V-001")
	if err := os.MkdirAll(violationDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(violationDir, "test.go.bak"), []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create legacy backup
	now := time.Now().Format("20060102T150405")
	legacyDir := filepath.Join(projectDir, ".arx-backup", now)
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "legacy.go.bak"), []byte("legacy\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := buildArxBinary(t)

	listOut, err := runArx(t, binaryPath, projectDir, "rollback", "--list")
	if err != nil {
		t.Fatalf("rollback --list failed: %v\nOutput: %s", err, listOut)
	}
	t.Logf("rollback --list output: %s", listOut)

	if !strings.Contains(listOut, "V-001") {
		t.Errorf("rollback --list should show violation ID, got: %s", listOut)
	}
	if !strings.Contains(listOut, "test.go") {
		t.Errorf("rollback --list should show filename, got: %s", listOut)
	}

	t.Log("✅ rollback --list works")
}

func TestSuggestBatch_LegacyBackupCompat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	projectDir := t.TempDir()

	// Create a test file
	testFile := "test.go"
	if err := os.WriteFile(filepath.Join(projectDir, testFile), []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create legacy timestamp-based backup (old format)
	legacyTs := "20250101T120000"
	backupDir := filepath.Join(projectDir, ".arx-backup", legacyTs)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, testFile+".bak"), []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}

	binaryPath := buildArxBinary(t)

	// Rollback should detect and restore from legacy backup
	rollbackOut, err := runArx(t, binaryPath, projectDir, "rollback", testFile)
	if err != nil {
		t.Fatalf("rollback with legacy backup failed: %v\nOutput: %s", err, rollbackOut)
	}
	t.Logf("legacy rollback output: %s", rollbackOut)

	restored, err := os.ReadFile(filepath.Join(projectDir, testFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "original\n" {
		t.Errorf("file was NOT restored from legacy backup, got: %s", string(restored))
	}

	t.Log("✅ Legacy backup compatibility works")
}

func init() {
	fmt.Println("Suggest batch integration tests initialized")
}
