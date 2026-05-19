package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRollbackCommand_IsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"rollback"})
	if err != nil {
		t.Fatal("rollback command not found on rootCmd")
	}
	if cmd.Use != "rollback [file]" {
		t.Errorf("expected use 'rollback [file]', got %q", cmd.Use)
	}
}

func TestRollbackCommand_HasListFlag(t *testing.T) {
	flag := rollbackCmd.Flags().Lookup("list")
	if flag == nil {
		t.Fatal("--list flag not found on rollback command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--list default should be false, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--list should be bool type, got %q", flag.Value.Type())
	}
}

func TestRollbackCommand_HasAllFlag(t *testing.T) {
	flag := rollbackCmd.Flags().Lookup("all")
	if flag == nil {
		t.Fatal("--all flag not found on rollback command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--all default should be false, got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--all should be bool type, got %q", flag.Value.Type())
	}
}

func TestRollbackCommand_ListFlag_ShowsBackups(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	// Create a backup manually
	backupDir := filepath.Join(".arx-backup", "V-001")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "test.go.bak"), []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rollbackList = true
	rollbackAll = false
	defer func() {
		rollbackList = false
		rollbackAll = false
	}()

	var buf bytes.Buffer
	oldOut := rollbackStdout
	rollbackStdout = &buf
	defer func() { rollbackStdout = oldOut }()

	err = runRollback(rollbackCmd, nil)
	if err != nil {
		t.Fatalf("runRollback failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "V-001") {
		t.Errorf("output should contain violation ID, got: %s", output)
	}
	if !strings.Contains(output, "test.go") {
		t.Errorf("output should contain filename, got: %s", output)
	}
}

func TestRollbackCommand_AllFlag_RestoresAll(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	// Create backup and modified file
	backupDir := filepath.Join(".arx-backup", "V-001")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "test.go.bak"), []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("test.go", []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rollbackList = false
	rollbackAll = true
	defer func() {
		rollbackList = false
		rollbackAll = false
	}()

	var buf bytes.Buffer
	oldOut := rollbackStdout
	rollbackStdout = &buf
	defer func() { rollbackStdout = oldOut }()

	err = runRollback(rollbackCmd, nil)
	if err != nil {
		t.Fatalf("runRollback failed: %v", err)
	}

	data, err := os.ReadFile("test.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Errorf("file content = %q, want %q", string(data), "original\n")
	}
}

func TestRollbackCommand_RestoreSingleFile(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	// Create backup and modified file
	backupDir := filepath.Join(".arx-backup", "V-001")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "test.go.bak"), []byte("original\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("test.go", []byte("modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	rollbackList = false
	rollbackAll = false
	defer func() {
		rollbackList = false
		rollbackAll = false
	}()

	var buf bytes.Buffer
	oldOut := rollbackStdout
	rollbackStdout = &buf
	defer func() { rollbackStdout = oldOut }()

	err = runRollback(rollbackCmd, []string{"test.go"})
	if err != nil {
		t.Fatalf("runRollback failed: %v", err)
	}

	data, err := os.ReadFile("test.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Errorf("file content = %q, want %q", string(data), "original\n")
	}
}

func TestRollbackCommand_RollbackNonexistentFile(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	rollbackList = false
	rollbackAll = false
	defer func() {
		rollbackList = false
		rollbackAll = false
	}()

	oldOut := rollbackStdout
	rollbackStdout = io.Discard
	defer func() { rollbackStdout = oldOut }()

	err = runRollback(rollbackCmd, []string{"nonexistent.go"})
	if err == nil {
		t.Fatal("expected error when restoring nonexistent file, got nil")
	}
}

func TestRollbackCommand_NoBackups(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	rollbackList = true
	rollbackAll = false
	defer func() {
		rollbackList = false
		rollbackAll = false
	}()

	var buf bytes.Buffer
	oldOut := rollbackStdout
	rollbackStdout = &buf
	defer func() { rollbackStdout = oldOut }()

	err = runRollback(rollbackCmd, nil)
	if err != nil {
		t.Fatalf("runRollback failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No backups") {
		t.Errorf("output should say 'No backups', got: %s", output)
	}
}
