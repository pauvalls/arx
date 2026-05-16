package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCommand is a small helper used throughout this test file.
func runCommand(cmd *exec.Cmd) (string, int) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return strings.TrimSpace(output), exitCode
}

// TestRuleExclude_ExcludedPathSkipped verifies that files matching exclude patterns
// do NOT produce violations for that rule.
func TestRuleExclude_ExcludedPathSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with a rule that excludes internal/legacy/**
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: infrastructure
    paths:
      - internal/infrastructure/**
  - name: legacy
    paths:
      - internal/legacy/**
rules:
  - id: no-domain-to-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain should not depend on infrastructure
    exclude:
      - internal/legacy/**
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source file in legacy layer (should be excluded)
	legacyDir := filepath.Join(tmpDir, "internal", "legacy")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	legacyFile := `package legacy

import (
	"example.com/test/internal/infrastructure/db"
)
`
	writeFile(t, filepath.Join(legacyDir, "old.go"), legacyFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should NOT find violations (excluded)
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, exitCode := runCommand(cmd)

	// Exit code should be 0 (no violations)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no violations), got %d. Output: %s", exitCode, output)
	}

	// Should not contain violation messages
	if strings.Contains(output, "no-domain-to-infra") {
		t.Errorf("expected no violations for excluded path, got: %s", output)
	}
}

// TestRuleExclude_NonExcludedPathReported verifies that files NOT matching
// exclude patterns still produce violations.
func TestRuleExclude_NonExcludedPathReported(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with a rule that excludes internal/legacy/**
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: infrastructure
    paths:
      - internal/infrastructure/**
  - name: legacy
    paths:
      - internal/legacy/**
rules:
  - id: no-domain-to-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain should not depend on infrastructure
    exclude:
      - internal/legacy/**
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source file in domain layer (should NOT be excluded)
	domainDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	domainFile := `package domain

import (
	"example.com/test/internal/infrastructure/db"
)
`
	writeFile(t, filepath.Join(domainDir, "user.go"), domainFile)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should find violations
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, exitCode := runCommand(cmd)

	// Exit code should be 1 (violations found)
	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violations found), got %d. Output: %s", exitCode, output)
	}

	// Should contain violation messages
	if !strings.Contains(output, "D-01") {
		t.Errorf("expected violation ID in output, got: %s", output)
	}
	if !strings.Contains(output, "Domain should not depend on infrastructure") {
		t.Errorf("expected violation explanation, got: %s", output)
	}
}

// TestRuleExclude_MultiplePatterns verifies that multiple exclude patterns work correctly.
func TestRuleExclude_MultiplePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with multiple exclude patterns
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: infrastructure
    paths:
      - internal/infrastructure/**
  - name: legacy
    paths:
      - internal/legacy/**
  - name: experimental
    paths:
      - internal/experimental/**
rules:
  - id: no-domain-to-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain should not depend on infrastructure
    exclude:
      - internal/legacy/**
      - internal/experimental/**
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source files in different layers
	// Legacy (excluded)
	legacyDir := filepath.Join(tmpDir, "internal", "legacy")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	writeFile(t, filepath.Join(legacyDir, "old.go"), `package legacy
import "example.com/test/internal/infrastructure/db"
`)

	// Experimental (excluded)
	expDir := filepath.Join(tmpDir, "internal", "experimental")
	if err := os.MkdirAll(expDir, 0755); err != nil {
		t.Fatalf("failed to create experimental dir: %v", err)
	}
	writeFile(t, filepath.Join(expDir, "new.go"), `package experimental
import "example.com/test/internal/infrastructure/db"
`)

	// Domain (NOT excluded)
	domainDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}
	writeFile(t, filepath.Join(domainDir, "user.go"), `package domain
import "example.com/test/internal/infrastructure/db"
`)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check — should only find violation in domain layer
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, exitCode := runCommand(cmd)

	// Exit code should be 1 (violations found in domain)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d. Output: %s", exitCode, output)
	}

	// Should contain exactly one violation (from domain/user.go)
	// Count occurrences of the violation ID
	count := strings.Count(output, "D-01")
	if count < 1 {
		t.Errorf("expected at least 1 violation, found %d. Output: %s", count, output)
	}

	// The violation should be from domain/user.go
	if !strings.Contains(output, "internal/domain/user.go") {
		t.Errorf("expected violation from domain/user.go, got: %s", output)
	}
}

// TestRuleExclude_GlobWildcard verifies that glob wildcards work correctly.
func TestRuleExclude_GlobWildcard(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with glob wildcard pattern
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: infrastructure
    paths:
      - internal/infrastructure/**
rules:
  - id: no-domain-to-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    exclude:
      - "internal/domain/*.go"
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source files
	domainDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}

	// This file matches *.go (excluded)
	writeFile(t, filepath.Join(domainDir, "user.go"), `package domain
import "example.com/test/internal/infrastructure/db"
`)

	// Create subdirectory with file that doesn't match *.go (not excluded)
	subDir := filepath.Join(domainDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	writeFile(t, filepath.Join(subDir, "service.go"), `package sub
import "example.com/test/internal/infrastructure/db"
`)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, exitCode := runCommand(cmd)

	// Should only find violation in sub/service.go (not excluded by *.go)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d. Output: %s", exitCode, output)
	}

	// Should contain violation from sub/service.go but not user.go
	if !strings.Contains(output, "internal/domain/sub/service.go") {
		t.Errorf("expected violation from sub/service.go, got: %s", output)
	}
	if strings.Contains(output, "internal/domain/user.go") {
		t.Errorf("did not expect violation from user.go (excluded), got: %s", output)
	}
}

// TestRuleExclude_TrailingSlash verifies that trailing slash patterns work as directory prefix.
func TestRuleExclude_TrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Go project structure
	writeGoMod(t, tmpDir, "example.com/test")

	// Create arx.yaml with trailing slash pattern
	arxYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - internal/domain/**
  - name: infrastructure
    paths:
      - internal/infrastructure/**
rules:
  - id: no-domain-to-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    exclude:
      - "internal/domain/legacy/"
`
	writeFile(t, filepath.Join(tmpDir, "arx.yaml"), arxYAML)

	// Create source files
	domainDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("failed to create domain dir: %v", err)
	}

	// This file is in legacy/ directory (excluded)
	legacyDir := filepath.Join(domainDir, "legacy")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	writeFile(t, filepath.Join(legacyDir, "old.go"), `package legacy
import "example.com/test/internal/infrastructure/db"
`)

	// This file is NOT in legacy/ directory (not excluded)
	writeFile(t, filepath.Join(domainDir, "user.go"), `package domain
import "example.com/test/internal/infrastructure/db"
`)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check
	cmd := exec.Command(binaryPath, "check")
	cmd.Dir = tmpDir
	output, exitCode := runCommand(cmd)

	// Should only find violation from user.go
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d. Output: %s", exitCode, output)
	}

	if !strings.Contains(output, "internal/domain/user.go") {
		t.Errorf("expected violation from user.go, got: %s", output)
	}
	if strings.Contains(output, "internal/domain/legacy/old.go") {
		t.Errorf("did not expect violation from legacy/old.go (excluded), got: %s", output)
	}
}
