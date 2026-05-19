package baseline

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestTrackStorage_SaveAndLoad(t *testing.T) {
	t.Run("saves and loads track", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "baseline-track.json")
		storage := NewTrackStorage()

		now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
		track := domain.BaselineTrack{
			ConsecutiveClean: 3,
			LastCheck:        now,
			LastSnapshot:     now,
			SnapshotCount:    2,
		}

		if err := storage.SaveTrack(path, track); err != nil {
			t.Fatalf("SaveTrack: %v", err)
		}

		loaded, err := storage.LoadTrack(path)
		if err != nil {
			t.Fatalf("LoadTrack: %v", err)
		}
		if loaded == nil {
			t.Fatal("LoadTrack returned nil")
		}
		if loaded.ConsecutiveClean != 3 {
			t.Errorf("ConsecutiveClean = %d, want 3", loaded.ConsecutiveClean)
		}
		if loaded.SnapshotCount != 2 {
			t.Errorf("SnapshotCount = %d, want 2", loaded.SnapshotCount)
		}
	})

	t.Run("returns nil when file does not exist", func(t *testing.T) {
		storage := NewTrackStorage()

		loaded, err := storage.LoadTrack("/nonexistent/path.json")
		if err != nil {
			t.Fatalf("LoadTrack: %v", err)
		}
		if loaded != nil {
			t.Error("LoadTrack should return nil for non-existent file")
		}
	})
}

func TestTrackStorage_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline-track.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	storage := NewTrackStorage()
	loaded, err := storage.LoadTrack(path)
	if err == nil {
		t.Fatal("LoadTrack should return error for invalid JSON")
	}
	if loaded != nil {
		t.Error("LoadTrack should return nil on error")
	}
}

func TestTrackStorage_ThreadSafe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline-track.json")
	storage := NewTrackStorage()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			track := domain.BaselineTrack{
				ConsecutiveClean: n,
				LastCheck:        time.Now(),
			}
			err := storage.SaveTrack(path, track)
			if err != nil {
				t.Errorf("SaveTrack: %v", err)
			}
			_, err = storage.LoadTrack(path)
			if err != nil {
				t.Errorf("LoadTrack: %v", err)
			}
		}(i)
	}
	wg.Wait()
}

func TestTrackStorage_EmptyTrack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline-track.json")
	storage := NewTrackStorage()

	track := domain.BaselineTrack{}
	if err := storage.SaveTrack(path, track); err != nil {
		t.Fatalf("SaveTrack: %v", err)
	}

	loaded, err := storage.LoadTrack(path)
	if err != nil {
		t.Fatalf("LoadTrack: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadTrack returned nil for empty track")
	}
	if loaded.ConsecutiveClean != 0 {
		t.Errorf("ConsecutiveClean = %d, want 0", loaded.ConsecutiveClean)
	}
}
