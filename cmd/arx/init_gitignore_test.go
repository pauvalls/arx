package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitignoreEntries_ContainsBaselineHistory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .git directory to simulate a git repo
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Run ensureGitignoreEntries
	if err := ensureGitignoreEntries(tmpDir); err != nil {
		t.Fatalf("ensureGitignoreEntries: %v", err)
	}

	// Read .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, ".arx-baseline-history/") {
		t.Errorf(".gitignore should contain '.arx-baseline-history/', got:\n%s", content)
	}
	if !strings.Contains(content, ".arx-cache/") {
		t.Errorf(".gitignore should contain '.arx-cache/'")
	}
	if !strings.Contains(content, ".arx-history/") {
		t.Errorf(".gitignore should contain '.arx-history/'")
	}
}

func TestEnsureGitignoreEntries_NoDuplicateEntries(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Run twice to check no duplicates
	if err := ensureGitignoreEntries(tmpDir); err != nil {
		t.Fatalf("First call: %v", err)
	}
	if err := ensureGitignoreEntries(tmpDir); err != nil {
		t.Fatalf("Second call: %v", err)
	}

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	data, _ := os.ReadFile(gitignorePath)
	content := string(data)

	// Count occurrences of the baseline history entry
	count := strings.Count(content, ".arx-baseline-history/")
	if count != 1 {
		t.Errorf("Expected exactly 1 occurrence of '.arx-baseline-history/', got %d", count)
	}
}

func TestEnsureGitignoreEntries_NoGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// No .git directory — should be a no-op
	err := ensureGitignoreEntries(tmpDir)
	if err != nil {
		t.Fatalf("ensureGitignoreEntries outside git repo: %v", err)
	}

	// .gitignore should not be created
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); !os.IsNotExist(err) {
		t.Error(".gitignore should not be created outside git repo")
	}
}
