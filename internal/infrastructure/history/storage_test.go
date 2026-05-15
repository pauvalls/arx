package history

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func setupTestHistory(t *testing.T) (*FileSystemHistory, string) {
	t.Helper()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, ".arx-history")

	return NewFileSystemHistory(historyPath), historyPath
}

func createTestReport(t *testing.T, date time.Time) *domain.AuditReport {
	t.Helper()

	return &domain.AuditReport{
		Timestamp:   date,
		ProjectRoot: "/test/project",
		ConfigHash:  "abc123",
		Violations: []domain.Violation{
			{
				ID:          "V-001",
				RuleID:      "no-inbound-deps",
				File:        "domain/order.go",
				Line:        10,
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/infrastructure",
				Message:     "domain cannot depend on infrastructure",
			},
		},
		CouplingMatrix: domain.NewCouplingMatrix(),
		DebtScore:      domain.NewDebtScore(),
	}
}

func TestHistoryStorage_Save(t *testing.T) {
	t.Run("saves audit report to JSON file", func(t *testing.T) {
		storage, historyPath := setupTestHistory(t)
		ctx := context.Background()

		reportDate := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
		report := createTestReport(t, reportDate)

		savedPath, err := storage.Save(ctx, report)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file was created
		expectedFilename := "audit-2026-05-14.json"
		expectedPath := filepath.Join(historyPath, expectedFilename)
		if savedPath != expectedPath {
			t.Errorf("Save() path = %v, want %v", savedPath, expectedPath)
		}

		// Verify file exists
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", expectedPath)
		}

		// Verify content can be loaded
		loaded, err := storage.Load(ctx, reportDate)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if loaded == nil {
			t.Fatal("Load() returned nil")
		}

		if loaded.ProjectRoot != report.ProjectRoot {
			t.Errorf("ProjectRoot = %v, want %v", loaded.ProjectRoot, report.ProjectRoot)
		}
	})

	t.Run("creates history directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		historyPath := filepath.Join(tmpDir, ".arx-history", "nested")
		storage := NewFileSystemHistory(historyPath)
		ctx := context.Background()

		report := createTestReport(t, time.Now())
		_, err := storage.Save(ctx, report)

		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(historyPath); os.IsNotExist(err) {
			t.Error("Expected history directory to be created")
		}
	})

	t.Run("enforces retention policy when DeleteOld is called", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		// Create 12 audits (exceeds default limit of 10)
		for i := 0; i < 12; i++ {
			date := time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		// Manually enforce retention
		deleted, err := storage.DeleteOld(ctx, 10)
		if err != nil {
			t.Fatalf("DeleteOld() error = %v", err)
		}
		if deleted != 2 {
			t.Errorf("DeleteOld() deleted = %d, want 2", deleted)
		}

		// Verify only 10 audits remain
		dates, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(dates) != 10 {
			t.Errorf("List() count = %d, want 10", len(dates))
		}
	})
}

func TestHistoryStorage_Load(t *testing.T) {
	t.Run("loads existing audit report", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		reportDate := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
		report := createTestReport(t, reportDate)

		_, err := storage.Save(ctx, report)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		loaded, err := storage.Load(ctx, reportDate)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded == nil {
			t.Fatal("Load() returned nil")
		}

		if loaded.Timestamp != report.Timestamp {
			t.Errorf("Timestamp = %v, want %v", loaded.Timestamp, report.Timestamp)
		}
	})

	t.Run("returns nil for non-existent date", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		nonExistentDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		loaded, err := storage.Load(ctx, nonExistentDate)

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded != nil {
			t.Error("Expected Load() to return nil for non-existent date")
		}
	})

	t.Run("returns nil when history directory doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		historyPath := filepath.Join(tmpDir, "nonexistent")
		storage := NewFileSystemHistory(historyPath)
		ctx := context.Background()

		loaded, err := storage.Load(ctx, time.Now())

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if loaded != nil {
			t.Error("Expected Load() to return nil when directory doesn't exist")
		}
	})
}

func TestHistoryStorage_LoadLatest(t *testing.T) {
	t.Run("loads most recent audit", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		// Create audits on different dates
		dates := []time.Time{
			time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
		}

		for _, date := range dates {
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		latest, err := storage.LoadLatest(ctx)
		if err != nil {
			t.Fatalf("LoadLatest() error = %v", err)
		}

		if latest == nil {
			t.Fatal("LoadLatest() returned nil")
		}

		expectedDate := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
		if latest.Timestamp != expectedDate {
			t.Errorf("LoadLatest() date = %v, want %v", latest.Timestamp, expectedDate)
		}
	})

	t.Run("returns nil when no history exists", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		latest, err := storage.LoadLatest(ctx)
		if err != nil {
			t.Fatalf("LoadLatest() error = %v", err)
		}

		if latest != nil {
			t.Error("Expected LoadLatest() to return nil when no history exists")
		}
	})
}

func TestHistoryStorage_List(t *testing.T) {
	t.Run("lists all audit dates sorted newest first", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		// Create audits on different dates
		dates := []time.Time{
			time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
		}

		for _, date := range dates {
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		listed, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 3 {
			t.Errorf("List() count = %d, want 3", len(listed))
		}

		// Verify sorted newest first
		expectedOrder := []time.Time{
			time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		}

		for i, date := range listed {
			if date != expectedOrder[i] {
				t.Errorf("List()[%d] = %v, want %v", i, date, expectedOrder[i])
			}
		}
	})

	t.Run("returns empty list when no history exists", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		listed, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 0 {
			t.Errorf("List() count = %d, want 0", len(listed))
		}
	})

	t.Run("returns all dates without retention limit", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		// Create 15 audits
		for i := 0; i < 15; i++ {
			date := time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		listed, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		// List returns ALL dates (retention is enforced by DeleteOld, not List)
		if len(listed) != 15 {
			t.Errorf("List() count = %d, want 15 (List doesn't enforce retention)", len(listed))
		}
	})

	t.Run("ignores non-audit files", func(t *testing.T) {
		storage, historyPath := setupTestHistory(t)
		ctx := context.Background()

		// Create an audit file
		report := createTestReport(t, time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC))
		_, err := storage.Save(ctx, report)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Create a non-audit file
		otherFile := filepath.Join(historyPath, "other-file.json")
		if err := os.WriteFile(otherFile, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		listed, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 1 {
			t.Errorf("List() count = %d, want 1 (should ignore non-audit files)", len(listed))
		}
	})
}

func TestHistoryStorage_DeleteOld(t *testing.T) {
	t.Run("deletes audits beyond retention limit", func(t *testing.T) {
		storage, historyPath := setupTestHistory(t)
		ctx := context.Background()

		// Create 15 audits
		for i := 0; i < 15; i++ {
			date := time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		// Delete old audits (keep only 10)
		deleted, err := storage.DeleteOld(ctx, 10)
		if err != nil {
			t.Fatalf("DeleteOld() error = %v", err)
		}

		if deleted != 5 {
			t.Errorf("DeleteOld() deleted = %d, want 5", deleted)
		}

		// Verify only 10 audits remain
		listed, err := storage.List(ctx)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 10 {
			t.Errorf("List() count = %d, want 10", len(listed))
		}

		// Verify oldest files were deleted
		oldestFile := filepath.Join(historyPath, "audit-2026-01-01.json")
		if _, err := os.Stat(oldestFile); !os.IsNotExist(err) {
			t.Error("Expected oldest audit file to be deleted")
		}

		// Verify newest files remain
		newestFile := filepath.Join(historyPath, "audit-2026-01-15.json")
		if _, err := os.Stat(newestFile); os.IsNotExist(err) {
			t.Error("Expected newest audit file to remain")
		}
	})

	t.Run("returns 0 when within retention limit", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		// Create 5 audits (within limit)
		for i := 0; i < 5; i++ {
			date := time.Date(2026, 1, i+1, 0, 0, 0, 0, time.UTC)
			report := createTestReport(t, date)
			_, err := storage.Save(ctx, report)
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}
		}

		deleted, err := storage.DeleteOld(ctx, 10)
		if err != nil {
			t.Fatalf("DeleteOld() error = %v", err)
		}

		if deleted != 0 {
			t.Errorf("DeleteOld() deleted = %d, want 0", deleted)
		}
	})

	t.Run("handles empty history gracefully", func(t *testing.T) {
		storage, _ := setupTestHistory(t)
		ctx := context.Background()

		deleted, err := storage.DeleteOld(ctx, 10)
		if err != nil {
			t.Fatalf("DeleteOld() error = %v", err)
		}

		if deleted != 0 {
			t.Errorf("DeleteOld() deleted = %d, want 0", deleted)
		}
	})
}
