package baseline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestHistoryStorage_SaveAndList(t *testing.T) {
	t.Run("saves snapshot and lists it sorted newest first", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewHistoryStorage()

		s1 := domain.BaselineSnapshot{
			CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
			TotalCount: 10,
		}
		s2 := domain.BaselineSnapshot{
			CreatedAt:  time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
			TotalCount: 5,
		}

		if err := storage.SaveSnapshot(dir, s1); err != nil {
			t.Fatalf("SaveSnapshot s1: %v", err)
		}
		if err := storage.SaveSnapshot(dir, s2); err != nil {
			t.Fatalf("SaveSnapshot s2: %v", err)
		}

		snapshots, err := storage.ListSnapshots(dir)
		if err != nil {
			t.Fatalf("ListSnapshots: %v", err)
		}

		if len(snapshots) != 2 {
			t.Fatalf("got %d snapshots, want 2", len(snapshots))
		}

		// Newest first
		if snapshots[0].TotalCount != 5 {
			t.Errorf("first snapshot TotalCount = %d, want 5", snapshots[0].TotalCount)
		}
		if snapshots[1].TotalCount != 10 {
			t.Errorf("second snapshot TotalCount = %d, want 10", snapshots[1].TotalCount)
		}
	})

	t.Run("empty directory returns empty list", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewHistoryStorage()

		snapshots, err := storage.ListSnapshots(dir)
		if err != nil {
			t.Fatalf("ListSnapshots: %v", err)
		}
		if len(snapshots) != 0 {
			t.Errorf("got %d snapshots, want 0", len(snapshots))
		}
	})

	t.Run("missing directory returns empty list", func(t *testing.T) {
		storage := NewHistoryStorage()

		snapshots, err := storage.ListSnapshots("/nonexistent/path")
		if err != nil {
			t.Fatalf("ListSnapshots on missing dir: %v", err)
		}
		if len(snapshots) != 0 {
			t.Errorf("got %d snapshots, want 0", len(snapshots))
		}
	})
}

func TestHistoryStorage_LatestSnapshot(t *testing.T) {
	t.Run("returns most recent snapshot", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewHistoryStorage()

		s1 := domain.BaselineSnapshot{
			CreatedAt:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
			TotalCount: 10,
		}
		s2 := domain.BaselineSnapshot{
			CreatedAt:  time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC),
			TotalCount: 3,
		}

		if err := storage.SaveSnapshot(dir, s1); err != nil {
			t.Fatalf("SaveSnapshot s1: %v", err)
		}
		if err := storage.SaveSnapshot(dir, s2); err != nil {
			t.Fatalf("SaveSnapshot s2: %v", err)
		}

		latest, err := storage.LatestSnapshot(dir)
		if err != nil {
			t.Fatalf("LatestSnapshot: %v", err)
		}
		if latest == nil {
			t.Fatal("LatestSnapshot returned nil")
		}
		if latest.TotalCount != 3 {
			t.Errorf("TotalCount = %d, want 3", latest.TotalCount)
		}
	})

	t.Run("returns nil when no snapshots", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewHistoryStorage()

		latest, err := storage.LatestSnapshot(dir)
		if err != nil {
			t.Fatalf("LatestSnapshot: %v", err)
		}
		if latest != nil {
			t.Error("LatestSnapshot should be nil when no snapshots exist")
		}
	})
}

func TestHistoryStorage_RoundtripPreservesData(t *testing.T) {
	dir := t.TempDir()
	storage := NewHistoryStorage()

	now := time.Date(2026, 5, 19, 14, 30, 0, 0, time.UTC)
	snapshot := domain.BaselineSnapshot{
		Violations: []domain.Violation{
			{ID: "V-001", RuleID: "R001", File: "a.go", Line: 10, SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", Severity: domain.SeverityError},
			{ID: "V-002", RuleID: "R002", File: "b.go", Line: 20, SourceLayer: "application", TargetLayer: "domain", Import: "y", Severity: domain.SeverityWarning},
		},
		CreatedAt:  now,
		TotalCount: 2,
		SeverityBreakdown: map[domain.Severity]int{
			domain.SeverityError:   1,
			domain.SeverityWarning: 1,
		},
	}

	if err := storage.SaveSnapshot(dir, snapshot); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	loaded, err := storage.LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if loaded == nil {
		t.Fatal("LatestSnapshot returned nil")
	}

	if loaded.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", loaded.TotalCount)
	}
	if len(loaded.Violations) != 2 {
		t.Errorf("len(Violations) = %d, want 2", len(loaded.Violations))
	}
	if loaded.Violations[0].ID != "V-001" {
		t.Errorf("Violation[0].ID = %q, want %q", loaded.Violations[0].ID, "V-001")
	}
	if loaded.SeverityBreakdown[domain.SeverityError] != 1 {
		t.Errorf("Errors = %d, want 1", loaded.SeverityBreakdown[domain.SeverityError])
	}
}

func TestHistoryStorage_MalformedFile(t *testing.T) {
	dir := t.TempDir()
	storage := NewHistoryStorage()

	// Write a file with a valid-looking name but invalid JSON
	badFile := filepath.Join(dir, "20260519T143000.json")
	if err := os.WriteFile(badFile, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// ListSnapshots should skip malformed files
	snapshots, err := storage.ListSnapshots(dir)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("got %d snapshots, want 0 (malformed file should be skipped)", len(snapshots))
	}

	// LatestSnapshot should skip malformed files
	latest, err := storage.LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if latest != nil {
		t.Error("LatestSnapshot should be nil when only malformed file exists")
	}
}
