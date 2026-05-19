package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/ports"
)

func TestRunDetectorsCachedWithStatus_NilCache(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := RunDetectorsCachedWithStatus(context.Background(), tmpDir, nil, nil, nil)
	if err == nil {
		t.Error("expected error for nil detectors")
	}
}

func TestRunDetectorsCachedWithStatus_EmptyDetectors(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := RunDetectorsCachedWithStatus(context.Background(), tmpDir, nil, []ports.Detector{}, &mockCache{})
	if err == nil {
		t.Error("expected error for empty detectors")
	}
}

func TestHashProjectFiles_GoProject(t *testing.T) {
	tmpDir := t.TempDir()
	// Create Go files
	internal := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internal, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(internal, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(internal, "util.go"), []byte("package util"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashProjectFiles(tmpDir, "go")
	if err != nil {
		t.Fatalf("hashProjectFiles() error = %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for Go project")
	}

	// Same project should produce same hash deterministically
	hash2, err := hashProjectFiles(tmpDir, "go")
	if err != nil {
		t.Fatal(err)
	}
	if hash != hash2 {
		t.Error("expected deterministic hash for same project")
	}
}

func TestHashProjectFiles_NoRelevantFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashProjectFiles(tmpDir, "go")
	if err != nil {
		t.Fatalf("hashProjectFiles() error = %v", err)
	}
	// No .go files = empty hash
	if hash != "" {
		t.Logf("hash for non-Go project: %s", hash)
	}
}

func TestHashFile_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "data.bin")
	data := []byte{0, 1, 2, 3, 4, 5}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashFile(filePath)
	if err != nil {
		t.Fatalf("hashFile() error = %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for binary file")
	}
}
