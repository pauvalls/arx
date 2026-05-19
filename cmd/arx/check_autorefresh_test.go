package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestAutoRefresh_TriggersAfterThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a baseline file
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	baseline := &domain.Baseline{
		Version:     "1.0",
		ConfigHash:  "hash",
		GeneratedAt: time.Now(),
		Violations:  []domain.BaselineViolation{},
	}
	svc := newBaselineService()
	if err := svc.Save(baseline, baselinePath); err != nil {
		t.Fatalf("Save baseline: %v", err)
	}

	// Create track file at threshold-1 (so next check triggers)
	cacheDir := filepath.Join(tmpDir, ".arx-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	trackPath := filepath.Join(cacheDir, "baseline-track.json")

	extSvc := newExtendedBaselineService()
	track := domain.BaselineTrack{
		ConsecutiveClean: 2,
		LastCheck:        time.Now(),
	}
	if err := extSvc.SaveTrack(trackPath, track); err != nil {
		t.Fatalf("SaveTrack: %v", err)
	}

	// Set the global threshold for the test
	oldThreshold := baselineRefreshThreshold
	baselineRefreshThreshold = 3
	defer func() { baselineRefreshThreshold = oldThreshold }()

	// Override projectRoot used by tryAutoRefresh by saving the baseline at the right path
	tryAutoRefresh(tmpDir)

	// Verify track was updated
	loaded, err := extSvc.LoadTrack(trackPath)
	if err != nil {
		t.Fatalf("LoadTrack: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadTrack returned nil")
	}
	if loaded.ConsecutiveClean != 0 {
		t.Errorf("ConsecutiveClean = %d, want 0 (reset after refresh)", loaded.ConsecutiveClean)
	}
	if loaded.SnapshotCount != 1 {
		t.Errorf("SnapshotCount = %d, want 1", loaded.SnapshotCount)
	}

	// Verify a baseline history snapshot was created
	snapshot, err := extSvc.LatestSnapshot(tmpDir)
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if snapshot == nil {
		t.Error("Expected a snapshot to be created after auto-refresh")
	}
}

func TestAutoRefresh_NoBaseline(t *testing.T) {
	tmpDir := t.TempDir()

	// No baseline file — auto-refresh should do nothing
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	if _, err := os.Stat(baselinePath); !os.IsNotExist(err) {
		t.Fatal("baseline should not exist")
	}

	tryAutoRefresh(tmpDir)

	// No crash, no snapshot directory created
	extSvc := newExtendedBaselineService()
	snapshot, err := extSvc.LatestSnapshot(tmpDir)
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if snapshot != nil {
		t.Error("No snapshot should be created without baseline")
	}
}

func TestAutoRefresh_CounterIncrements(t *testing.T) {
	tmpDir := t.TempDir()

	// Baseline exists
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	svc := newBaselineService()
	svc.Save(&domain.Baseline{
		Version: "1.0", ConfigHash: "hash", GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{},
	}, baselinePath)

	// Track at 0 (first clean check)
	cacheDir := filepath.Join(tmpDir, ".arx-cache")
	os.MkdirAll(cacheDir, 0755)
	trackPath := filepath.Join(cacheDir, "baseline-track.json")

	extSvc := newExtendedBaselineService()
	extSvc.SaveTrack(trackPath, domain.BaselineTrack{ConsecutiveClean: 0, LastCheck: time.Now()})

	oldThreshold := baselineRefreshThreshold
	baselineRefreshThreshold = 3
	defer func() { baselineRefreshThreshold = oldThreshold }()

	tryAutoRefresh(tmpDir)

	// Should be 1 now (incremented but not triggered)
	loaded, _ := extSvc.LoadTrack(trackPath)
	if loaded.ConsecutiveClean != 1 {
		t.Errorf("ConsecutiveClean = %d, want 1", loaded.ConsecutiveClean)
	}

	// Second clean check
	tryAutoRefresh(tmpDir)
	loaded, _ = extSvc.LoadTrack(trackPath)
	if loaded.ConsecutiveClean != 2 {
		t.Errorf("ConsecutiveClean = %d, want 2", loaded.ConsecutiveClean)
	}

	// Third clean check — triggers refresh
	tryAutoRefresh(tmpDir)
	loaded, _ = extSvc.LoadTrack(trackPath)
	if loaded.ConsecutiveClean != 0 {
		t.Errorf("After trigger ConsecutiveClean = %d, want 0", loaded.ConsecutiveClean)
	}
	if loaded.SnapshotCount < 1 {
		t.Error("SnapshotCount should be >= 1 after refresh")
	}
}

func TestAutoRefresh_TrackFileCreated(t *testing.T) {
	tmpDir := t.TempDir()

	// Baseline exists but no track file
	baselinePath := filepath.Join(tmpDir, ".arx-baseline.json")
	svc := newBaselineService()
	svc.Save(&domain.Baseline{
		Version: "1.0", ConfigHash: "hash", GeneratedAt: time.Now(),
		Violations: []domain.BaselineViolation{},
	}, baselinePath)

	oldThreshold := baselineRefreshThreshold
	baselineRefreshThreshold = 3
	defer func() { baselineRefreshThreshold = oldThreshold }()

	// Should create track file automatically
	tryAutoRefresh(tmpDir)

	// Track file should exist now
	trackPath := filepath.Join(tmpDir, ".arx-cache", "baseline-track.json")
	if _, err := os.Stat(trackPath); os.IsNotExist(err) {
		t.Error("Track file should be created")
	}
}
