package baseline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestStorage_SaveLoad_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	now := time.Now()
	baseline := &domain.Baseline{
		Version:     "1.0",
		ConfigHash:  "test-hash-123",
		GeneratedAt: now,
		Violations: []domain.BaselineViolation{
			{
				RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
				Import: "github.com/example/db", File: "user.go",
			},
		},
	}

	// Save
	if err := storage.Save(baseline, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Save() did not create the file")
	}

	// Load
	loaded, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil after Save()")
	}

	if loaded.Version != baseline.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, baseline.Version)
	}
	if loaded.ConfigHash != baseline.ConfigHash {
		t.Errorf("ConfigHash = %q, want %q", loaded.ConfigHash, baseline.ConfigHash)
	}
	if len(loaded.Violations) != 1 {
		t.Fatalf("Violations count = %d, want 1", len(loaded.Violations))
	}
	if loaded.Violations[0].RuleID != "R001" {
		t.Errorf("Violation RuleID = %q, want %q", loaded.Violations[0].RuleID, "R001")
	}
}

func TestStorage_Load_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	baseline, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for non-existent file", err)
	}
	if baseline != nil {
		t.Error("Load() should return nil for non-existent file")
	}
}

func TestStorage_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	if storage.Exists(path) {
		t.Error("Exists() should return false for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if !storage.Exists(path) {
		t.Error("Exists() should return true for existing file")
	}
}

func TestStorage_Save_NilBaseline(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	err := storage.Save(nil, path)
	if err == nil {
		t.Fatal("Save(nil) should return an error")
	}
}

func TestStorage_Load_CorruptedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not valid json{{{"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	storage := NewStorage()

	_, err := storage.Load(path)
	if err == nil {
		t.Fatal("Load() with corrupted JSON should return an error")
	}
}

func TestStorage_SaveLoad_EmptyViolations(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	baseline := &domain.Baseline{
		Version:     "1.0",
		ConfigHash:  "hash-123",
		GeneratedAt: time.Now(),
		Violations:  []domain.BaselineViolation{},
	}

	if err := storage.Save(baseline, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil")
	}

	if len(loaded.Violations) != 0 {
		t.Errorf("Violations count = %d, want 0", len(loaded.Violations))
	}
}

func TestStorage_Load_ReadError(t *testing.T) {
	// Create a directory instead of a file to cause a read error
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "is-a-dir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	storage := NewStorage()

	_, err := storage.Load(path)
	if err == nil {
		t.Fatal("Load() on directory should return an error")
	}
}

func TestStorage_LargeBaseline(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	storage := NewStorage()

	// Create a baseline with many violations
	violations := make([]domain.BaselineViolation, 1000)
	for i := 0; i < 1000; i++ {
		violations[i] = domain.BaselineViolation{
			RuleID:      fmt.Sprintf("R%04d", i),
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      fmt.Sprintf("github.com/example/pkg%d", i),
			File:        fmt.Sprintf("file%d.go", i),
		}
	}

	baseline := &domain.Baseline{
		Version:     "1.0",
		ConfigHash:  "large-hash",
		GeneratedAt: time.Now(),
		Violations:  violations,
	}

	if err := storage.Save(baseline, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := storage.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil")
	}

	if len(loaded.Violations) != 1000 {
		t.Errorf("Violations count = %d, want 1000", len(loaded.Violations))
	}

	// Verify some entries
	if loaded.Violations[0].RuleID != "R0000" {
		t.Errorf("First violation RuleID = %q, want %q", loaded.Violations[0].RuleID, "R0000")
	}
	if loaded.Violations[999].RuleID != "R0999" {
		t.Errorf("Last violation RuleID = %q, want %q", loaded.Violations[999].RuleID, "R0999")
	}
}
