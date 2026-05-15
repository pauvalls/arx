package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRulesPattern_MatchingImport verifies that a pattern-only rule
// detects violations when an import matches the configured pattern.
func TestRulesPattern_MatchingImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with a pattern-only rule.
	// Source file in "application" layer, import resolves to "domain" layer
	// to avoid false circular dependency from sourceLayer==targetLayer.
	arxYAML := `version: "1.0"
layers:
  - name: application
    paths:
      - internal/application/**
  - name: domain
    paths:
      - internal/domain/**
rules:
  - id: no-legacy
    pattern: ".*legacy.*"
    type: Cannot
    severity: error
    explanation: Legacy packages must not be imported
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source file in application layer importing from domain/legacy
	sourceDir := filepath.Join(tmpDir, "internal", "application")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	sourceFile := `package application

import (
	"example.com/test/internal/domain/legacy/helper"
)
`
	writeFile(t, filepath.Join(sourceDir, "service.go"), sourceFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should find violations and exit 1
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	// The rule's explanation appears in the violation output
	if !strings.Contains(outputStr, "Legacy packages must not be imported") {
		t.Errorf("expected violation explanation for legacy import, got: %s", outputStr)
	}
	// The matching import path should be in the output
	if !strings.Contains(outputStr, "internal/domain/legacy/helper") {
		t.Errorf("expected matching import path in output, got: %s", outputStr)
	}
}

// TestRulesPattern_NonMatchingImport verifies that a pattern-only rule
// does NOT report violations when the import doesn't match the pattern.
func TestRulesPattern_NonMatchingImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Use two layers to avoid false circular dependency
	arxYAML := `version: "1.0"
layers:
  - name: application
    paths:
      - internal/application/**
  - name: domain
    paths:
      - internal/domain/**
rules:
  - id: no-legacy
    pattern: ".*legacy.*"
    type: Cannot
    severity: error
    explanation: Legacy packages must not be imported
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source file with a non-matching import to domain
	sourceDir := filepath.Join(tmpDir, "internal", "application")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	sourceFile := `package application

import (
	"example.com/test/internal/domain/service"
)
`
	writeFile(t, filepath.Join(sourceDir, "service.go"), sourceFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should exit 0 (pattern doesn't match)
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// If there is an error, it should NOT be from our pattern rule
		if strings.Contains(outputStr, "Legacy packages must not be imported") {
			t.Errorf("expected no 'Legacy packages' violation for non-matching import, got: %s", outputStr)
		}
		// Allow failure only if it's not related to our pattern rule
		t.Logf("arx check exited with error (possibly unrelated): %v\nOutput: %s", err, outputStr)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Legacy packages must not be imported") {
		t.Errorf("expected no 'Legacy packages' violation for non-matching import, got: %s", outputStr)
	}
}

// TestRulesPattern_CombinedRule verifies that a combined (pattern + from/to) rule
// reports violations only when both conditions are met.
func TestRulesPattern_CombinedRule(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with a combined pattern+from/to rule
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: application
    paths:
      - internal/application/**
rules:
  - id: domain-no-db-imports
    from: domain
    to:
      - application
    pattern: ".*[Dd][Bb].*"
    type: Cannot
    severity: error
    explanation: Domain must not import database-related packages from application
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create domain source file importing a DB package from application
	sourceDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	sourceFile := `package domain

import (
	"example.com/test/internal/application/db/repository"
)
`
	writeFile(t, filepath.Join(sourceDir, "entity.go"), sourceFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should find violations and exit 1
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	// The rule's explanation should appear in the violation output
	if !strings.Contains(outputStr, "Domain must not import database-related packages from application") {
		t.Errorf("expected violation explanation for combined rule, got: %s", outputStr)
	}
	// The matching import path should be in the output
	if !strings.Contains(outputStr, "internal/application/db/repository") {
		t.Errorf("expected matching import path in output, got: %s", outputStr)
	}
}

// TestRulesPattern_InvalidRegexAtStartup verifies that an invalid regex pattern
// causes arx check to fail at startup with a clear error.
func TestRulesPattern_InvalidRegexAtStartup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with an invalid regex pattern
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
rules:
  - id: bad-pattern
    pattern: "[invalid"
    type: Cannot
    severity: error
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check -- should fail with invalid regex error
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("arx check should have failed with invalid regex")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "invalid pattern") {
		t.Errorf("expected 'invalid pattern' error, got: %s", outputStr)
	}
}

// TestRulesPattern_EdgeCases verifies edge case handling for patterns:
// unicode patterns, special regex characters, etc.
func TestRulesPattern_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Use two layers to avoid circular dependency false positives
	arxYAML := `version: "1.0"
layers:
  - name: application
    paths:
      - internal/application/**
  - name: domain
    paths:
      - internal/domain/**
rules:
  - id: unicode-pattern
    pattern: ".*café.*"
    type: Cannot
    severity: error
  - id: special-chars
    pattern: ".*[Uu]til/.*"
    type: Cannot
    severity: error
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Source file in application importing domain packages that match patterns
	sourceDir := filepath.Join(tmpDir, "internal", "application")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	sourceFile := `package application

import (
	"example.com/test/internal/domain/util/helper"
	"example.com/test/internal/domain/café/bar"
)
`
	writeFile(t, filepath.Join(sourceDir, "service.go"), sourceFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should find violations and exit 1
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, _ := cmd.CombinedOutput()

	outputStr := string(output)
	// Check for the matching import paths in the output
	if !strings.Contains(outputStr, "internal/domain/util/helper") {
		t.Errorf("expected violation for util import (special-chars pattern), got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "internal/domain/café/bar") {
		t.Errorf("expected violation for café import (unicode-pattern), got: %s", outputStr)
	}
	// Should be exactly 2 violations (no more, no less)
	if strings.Count(outputStr, "❌") != 2 {
		t.Errorf("expected exactly 2 violations, got output: %s", outputStr)
	}
}

// writeFile is a helper to write a file in a test.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// writeGoMod writes a basic go.mod file in the given directory.
func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	goModContent := "module " + module + "\n\ngo 1.23\n"
	writeFile(t, filepath.Join(dir, "go.mod"), goModContent)
}
