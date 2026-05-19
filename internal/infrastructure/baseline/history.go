package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

const (
	// historyDirName is the directory name for baseline snapshot history.
	historyDirName = ".arx-baseline-history"
	// snapshotTimeFormat is the ISO 8601 compact format for filenames.
	snapshotTimeFormat = "20060102T150405"
)

// HistoryStorage implements ports.SnapshotStorage using JSON files in a directory.
type HistoryStorage struct{}

// NewHistoryStorage creates a new HistoryStorage.
func NewHistoryStorage() *HistoryStorage {
	return &HistoryStorage{}
}

// historyDir returns the path to the history directory within the given dir.
func historyDir(dir string) string {
	return filepath.Join(dir, historyDirName)
}

// SaveSnapshot writes a snapshot as a timestamped JSON file.
func (s *HistoryStorage) SaveSnapshot(dir string, snapshot domain.BaselineSnapshot) error {
	saveDir := historyDir(dir)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	filename := snapshot.CreatedAt.Format(snapshotTimeFormat) + ".json"
	filePath := filepath.Join(saveDir, filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("writing snapshot file: %w", err)
	}

	return nil
}

// ListSnapshots returns all snapshots sorted by time, newest first.
func (s *HistoryStorage) ListSnapshots(dir string) ([]domain.BaselineSnapshot, error) {
	saveDir := historyDir(dir)

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(saveDir)
	if err != nil {
		return nil, fmt.Errorf("reading history directory: %w", err)
	}

	var snapshots []domain.BaselineSnapshot
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(saveDir, entry.Name()))
		if err != nil {
			continue // skip unreadable files
		}

		var snapshot domain.BaselineSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue // skip malformed files
		}

		snapshots = append(snapshots, snapshot)
	}

	// Sort by CreatedAt, newest first
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// LatestSnapshot returns the most recent snapshot, or nil if none exist.
func (s *HistoryStorage) LatestSnapshot(dir string) (*domain.BaselineSnapshot, error) {
	snapshots, err := s.ListSnapshots(dir)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, nil
	}
	return &snapshots[0], nil
}
