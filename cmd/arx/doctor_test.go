package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// createValidTestProject creates a temp project with valid config + go.mod + Go file
func createValidTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "arx.yaml")
	config := []byte(`version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test
    from: domain
    to: [domain]
    type: Cannot
`)
	if err := os.WriteFile(configPath, config, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Create go.mod for Go detector
	goMod := []byte(`module github.com/test/project

go 1.23
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file
	goDir := filepath.Join(tmpDir, "domain")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	goFile := []byte(`package domain`)
	if err := os.WriteFile(filepath.Join(goDir, "hello.go"), goFile, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	return tmpDir
}

// TestDoctorCmdSuccess tests doctor command on valid project
func TestDoctorCmdSuccess(t *testing.T) {
	tmpDir := createValidTestProject(t)

	cmd := &cobra.Command{}
	cmd.AddCommand(doctorCmd)

	args := []string{"doctor", tmpDir}
	cmd.SetArgs(args)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("doctor command should succeed, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "✅") {
		t.Errorf("expected success icon in output, got: %s", output)
	}
	if !strings.Contains(output, "Project root") {
		t.Error("expected 'Project root' check in output")
	}
}

// TestDoctorCmdNoConfig tests doctor on directory without config
func TestDoctorCmdNoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	cmd.AddCommand(doctorCmd)

	args := []string{"doctor", tmpDir}
	cmd.SetArgs(args)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected doctor to fail without config")
	}

	output := stdout.String()
	if !strings.Contains(output, "❌") {
		t.Errorf("expected error icon in output, got: %s", output)
	}
}

// TestDoctorCmdInvalidPath tests doctor on non-existent directory
func TestDoctorCmdInvalidPath(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(doctorCmd)

	args := []string{"doctor", "/nonexistent/path"}
	cmd.SetArgs(args)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected doctor to fail for nonexistent path")
	}

	output := stdout.String()
	if !strings.Contains(output, "❌") {
		t.Errorf("expected error icon, got: %s", output)
	}
}
