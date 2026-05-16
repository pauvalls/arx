package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestThreshold_ExitCodeUnderThreshold verifies exit code 0 when violations are under threshold
func TestThreshold_ExitCodeUnderThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with max_violations threshold
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: domain-no-import-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
max_violations: 5
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	writeGoMod(t, tmpDir, "example.com/test")
	// Create Go source files with violations
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)

	// Create 3 files in domain that import infrastructure (3 violations)
	for i := 0; i < 3; i++ {
		filename := filepath.Join(tmpDir, "domain", "service"+string(rune('A'+i))+".go")
		content := `package domain

import "example.com/test/infrastructure"

type Service` + string(rune('A'+i)) + ` struct {
	repo *infrastructure.Repository
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
	}

	// Create infrastructure file
	os.WriteFile(filepath.Join(tmpDir, "infrastructure", "repository.go"), []byte(`package infrastructure

type Repository struct{}
`), 0644)

	// Build arx binary
	binaryPath := buildArxBinary(t)

	// Run arx check
	cmd := exec.Command(binaryPath, "check", tmpDir)
	output, err := cmd.CombinedOutput()

	// Exit code should be 0 (under threshold of 5)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Errorf("Expected exit code 0 (under threshold), got %d\nOutput: %s", exitErr.ExitCode(), string(output))
		} else {
			t.Errorf("Command failed: %v\nOutput: %s", err, string(output))
		}
	}

	t.Logf("Exit code 0 as expected (3 violations under threshold of 5)\nOutput: %s", string(output))
}

// TestThreshold_ExitCodeOverThreshold verifies exit code 1 when violations exceed threshold
func TestThreshold_ExitCodeOverThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with max_violations threshold
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: domain-no-import-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
max_violations: 2
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	writeGoMod(t, tmpDir, "example.com/test")
	// Create Go source files with violations
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)

	// Create 5 files in domain that import infrastructure (5 violations, over threshold of 2)
	for i := 0; i < 5; i++ {
		filename := filepath.Join(tmpDir, "domain", "service"+string(rune('A'+i))+".go")
		content := `package domain

import "example.com/test/infrastructure"

type Service` + string(rune('A'+i)) + ` struct {
	repo *infrastructure.Repository
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
	}

	// Create infrastructure file
	os.WriteFile(filepath.Join(tmpDir, "infrastructure", "repository.go"), []byte(`package infrastructure

type Repository struct{}
`), 0644)

	// Run arx check
	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", tmpDir)
	
	output, err := cmd.CombinedOutput()

	// Exit code should be 1 (over threshold of 2)
	if err == nil {
		t.Errorf("Expected exit code 1 (over threshold), got 0\nOutput: %s", string(output))
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d\nOutput: %s", exitErr.ExitCode(), string(output))
			}
		}
	}

	t.Logf("Exit code 1 as expected (5 violations over threshold of 2)\nOutput: %s", string(output))
}

// TestThreshold_BackwardCompatibility verifies backward compatibility (no threshold = original behavior)
func TestThreshold_BackwardCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml WITHOUT max_violations (backward compatible)
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: domain-no-import-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	writeGoMod(t, tmpDir, "example.com/test")
	// Create Go source files with violations
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)

	// Create 1 file in domain that imports infrastructure (1 violation)
	filename := filepath.Join(tmpDir, "domain", "service.go")
	content := `package domain

import "example.com/test/infrastructure"

type Service struct {
	repo *infrastructure.Repository
}
`
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Create infrastructure file
	os.WriteFile(filepath.Join(tmpDir, "infrastructure", "repository.go"), []byte(`package infrastructure

type Repository struct{}
`), 0644)

	// Run arx check
	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", tmpDir)
	
	output, err := cmd.CombinedOutput()

	// Exit code should be 1 (any violation fails when no threshold set)
	if err == nil {
		t.Errorf("Expected exit code 1 (no threshold, any violation fails), got 0\nOutput: %s", string(output))
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d\nOutput: %s", exitErr.ExitCode(), string(output))
			}
		}
	}

	t.Logf("Exit code 1 as expected (backward compatibility: no threshold)\nOutput: %s", string(output))
}

// TestThreshold_ZeroThreshold verifies zero threshold means unlimited (backward compatible)
func TestThreshold_ZeroThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with max_violations: 0 (unlimited)
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: domain-no-import-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
max_violations: 0
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	writeGoMod(t, tmpDir, "example.com/test")
	// Create Go source files with violations
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)

	// Create 10 files in domain that import infrastructure (10 violations)
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tmpDir, "domain", "service"+string(rune('A'+i))+".go")
		content := `package domain

import "example.com/test/infrastructure"

type Service` + string(rune('A'+i)) + ` struct {
	repo *infrastructure.Repository
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
	}

	// Create infrastructure file
	os.WriteFile(filepath.Join(tmpDir, "infrastructure", "repository.go"), []byte(`package infrastructure

type Repository struct{}
`), 0644)

	// Run arx check
	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", tmpDir)
	
	output, err := cmd.CombinedOutput()

	// Exit code should be 1 (zero threshold means unlimited, but violations still fail)
	// Actually, with max_violations=0, the logic is: if maxViolations > 0, use threshold; else use backward-compatible behavior
	// So with 0, it should fail on any non-overridden violation
	if err == nil {
		t.Errorf("Expected exit code 1 (zero threshold = backward compatible), got 0\nOutput: %s", string(output))
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d\nOutput: %s", exitErr.ExitCode(), string(output))
			}
		}
	}

	t.Logf("Exit code 1 as expected (zero threshold = backward compatible)\nOutput: %s", string(output))
}

// TestThreshold_JSONOutput verifies JSON output includes max_violations field
func TestThreshold_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with max_violations threshold
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infrastructure/**
rules:
  - id: domain-no-import-infra
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    explanation: Domain must not depend on infrastructure
max_violations: 3
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	writeGoMod(t, tmpDir, "example.com/test")
	// Create Go source files with violations
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)

	// Create 2 files in domain that import infrastructure (2 violations)
	for i := 0; i < 2; i++ {
		filename := filepath.Join(tmpDir, "domain", "service"+string(rune('A'+i))+".go")
		content := `package domain

import "example.com/test/infrastructure"

type Service` + string(rune('A'+i)) + ` struct {
	repo *infrastructure.Repository
}
`
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %d: %v", i, err)
		}
	}

	// Create infrastructure file
	os.WriteFile(filepath.Join(tmpDir, "infrastructure", "repository.go"), []byte(`package infrastructure

type Repository struct{}
`), 0644)

	// Run arx check with JSON output
	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", "--ci", tmpDir)
	
	output, _ := cmd.CombinedOutput()

	// Check that JSON output contains max_violations field
	outputStr := string(output)
	if !contains(outputStr, `"max_violations"`) {
		t.Errorf("Expected JSON output to contain 'max_violations' field\nOutput: %s", outputStr)
	}

	// Check that max_violations value is 3
	if !contains(outputStr, `"max_violations": 3`) {
		t.Errorf("Expected JSON output to contain 'max_violations: 3'\nOutput: %s", outputStr)
	}

	t.Logf("JSON output correctly includes max_violations field\nOutput: %s", outputStr)
}

// TestThreshold_NegativeValueRejected verifies negative max_violations is rejected
func TestThreshold_NegativeValueRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with negative max_violations
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
rules: []
max_violations: -5
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(arxConfig), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Run arx check
	binaryPath := buildArxBinary(t)
	cmd := exec.Command(binaryPath, "check", tmpDir)
	
	output, err := cmd.CombinedOutput()

	// Should fail with validation error
	if err == nil {
		t.Errorf("Expected error for negative max_violations, got nil\nOutput: %s", string(output))
	}

	outputStr := string(output)
	if !contains(outputStr, "max_violations cannot be negative") {
		t.Errorf("Expected error message about negative max_violations\nOutput: %s", outputStr)
	}

	t.Logf("Negative max_violations correctly rejected\nOutput: %s", outputStr)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
