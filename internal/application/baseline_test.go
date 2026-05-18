package application

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/baseline"
)

// mockBaselineStore is a test-only implementation of ports.BaselineStorage.
// For Load/Exists it falls back to real filesystem when path is not in the map.
type mockBaselineStore struct {
	data map[string]*domain.Baseline
	saved map[string]bool
}

func (m *mockBaselineStore) Load(path string) (*domain.Baseline, error) {
	// Check if we saved it
	if m.saved != nil && m.saved[path] {
		// Read from filesystem (real storage behavior)
		return nil, nil // Don't implement actual JSON read
	}
	// Check in-memory
	if m.data != nil {
		if b, ok := m.data[path]; ok {
			return b, nil
		}
	}
	return nil, nil
}

func (m *mockBaselineStore) Save(b *domain.Baseline, path string) error {
	if b == nil {
		return fmt.Errorf("cannot save nil baseline")
	}
	if m.saved == nil {
		m.saved = make(map[string]bool)
	}
	m.saved[path] = true
	return nil
}

func (m *mockBaselineStore) Exists(path string) bool {
	// Check filesystem for real files (used by Exists test that creates a temp file)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	if m.saved != nil {
		return m.saved[path]
	}
	if m.data != nil {
		_, ok := m.data[path]
		return ok
	}
	return false
}

func newTestBaselineService() *BaselineService {
	return NewBaselineService(&mockBaselineStore{})
}

func TestBaselineService_Generate(t *testing.T) {
	svc := newTestBaselineService()

	violations := []domain.Violation{
		{
			RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure",
			Import: "github.com/example/db", File: "user.go", Line: 10,
		},
		{
			RuleID: "R002", SourceLayer: "application", TargetLayer: "domain",
			Import: "github.com/example/entity", File: "service.go", Line: 20,
		},
	}

	baseline := svc.Generate(violations, "test-hash")

	if baseline == nil {
		t.Fatal("Generate() returned nil")
	}
	if baseline.Version != "1.0" {
		t.Errorf("Version = %q, want %q", baseline.Version, "1.0")
	}
	if baseline.ConfigHash != "test-hash" {
		t.Errorf("ConfigHash = %q, want %q", baseline.ConfigHash, "test-hash")
	}
	if len(baseline.Violations) != 2 {
		t.Errorf("Violations count = %d, want 2", len(baseline.Violations))
	}
}

func TestBaselineService_SaveLoad_Roundtrip(t *testing.T) {
	realStore := &baseline.Storage{}
	svc := NewBaselineService(realStore)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	baseline := &domain.Baseline{
		Version:     "1.0",
		ConfigHash:  "hash-123",
		GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
		},
	}

	if err := svc.Save(baseline, path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := svc.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil after Save()")
	}
	if loaded.ConfigHash != baseline.ConfigHash {
		t.Errorf("ConfigHash = %q, want %q", loaded.ConfigHash, baseline.ConfigHash)
	}
}

func TestBaselineService_Load_NotExists(t *testing.T) {
	svc := newTestBaselineService()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	baseline, err := svc.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for non-existent file", err)
	}
	if baseline != nil {
		t.Error("Load() should return nil for non-existent file")
	}
}

func TestBaselineService_FilterViolations(t *testing.T) {
	svc := newTestBaselineService()

	baseline := &domain.Baseline{
		Version: "1.0", ConfigHash: "hash", GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "old.go"},
		},
	}

	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "github.com/example/db", File: "user.go", Line: 10},
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "github.com/example/entity", File: "service.go", Line: 20},
	}

	filtered := svc.FilterViolations(violations, baseline)

	if len(filtered) != 1 {
		t.Errorf("FilterViolations() returned %d violations, want 1", len(filtered))
	}
	if filtered[0].RuleID != "R002" {
		t.Errorf("FilterViolations() kept %q, want %q", filtered[0].RuleID, "R002")
	}
}

func TestBaselineService_FilterViolations_NilBaseline(t *testing.T) {
	svc := newTestBaselineService()

	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R002", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "y", File: "b.go", Line: 2},
	}

	filtered := svc.FilterViolations(violations, nil)

	if len(filtered) != 2 {
		t.Errorf("FilterViolations(nil) returned %d violations, want 2", len(filtered))
	}
}

func TestBaselineService_DefaultPath(t *testing.T) {
	svc := newTestBaselineService()

	path := svc.DefaultPath("/some/project")
	want := "/some/project/.arx-baseline.json"

	if path != want {
		t.Errorf("DefaultPath() = %q, want %q", path, want)
	}
}

func TestBaselineService_Exists(t *testing.T) {
	svc := newTestBaselineService()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	if svc.Exists(path) {
		t.Error("Exists() should return false for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	if !svc.Exists(path) {
		t.Error("Exists() should return true for existing file")
	}
}

func TestBaselineService_FilterViolations_Empty(t *testing.T) {
	svc := newTestBaselineService()

	baseline := &domain.Baseline{
		Version: "1.0", ConfigHash: "hash", GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
		},
	}

	filtered := svc.FilterViolations([]domain.Violation{}, baseline)

	if len(filtered) != 0 {
		t.Errorf("FilterViolations(empty) returned %d violations, want 0", len(filtered))
	}
}

func TestBaselineService_FilterViolations_AllSuppressed(t *testing.T) {
	svc := newTestBaselineService()

	baseline := &domain.Baseline{
		Version: "1.0", ConfigHash: "hash", GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
			{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go"},
		},
	}

	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
		{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go", Line: 2},
	}

	filtered := svc.FilterViolations(violations, baseline)

	if len(filtered) != 0 {
		t.Errorf("FilterViolations(all suppressed) returned %d violations, want 0", len(filtered))
	}
}

func TestBaselineService_Load_CorruptedJSON(t *testing.T) {
	realStore := &baseline.Storage{}
	svc := NewBaselineService(realStore)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not valid json{{{"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := svc.Load(path)
	if err == nil {
		t.Fatal("Load() with corrupted JSON should return an error")
	}
}

func TestBaselineService_Save_NilBaseline(t *testing.T) {
	realStore := &baseline.Storage{}
	svc := NewBaselineService(realStore)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".arx-baseline.json")

	err := svc.Save(nil, path)
	if err == nil {
		t.Fatal("Save(nil) should return an error")
	}
}

func TestBaselineService_Generate_EmptyConfigHash(t *testing.T) {
	svc := newTestBaselineService()

	violations := []domain.Violation{
		{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go", Line: 1},
	}

	baseline := svc.Generate(violations, "")

	if baseline == nil {
		t.Fatal("Generate() returned nil")
	}

	if baseline.ConfigHash != "" {
		t.Errorf("ConfigHash = %q, want empty string", baseline.ConfigHash)
	}
}
