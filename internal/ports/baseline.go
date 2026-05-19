package ports

import "github.com/pauvalls/arx/internal/domain"

// BaselineStorage defines the interface for baseline persistence.
// Application layer depends on this interface, not on concrete implementations.
type BaselineStorage interface {
	Load(path string) (*domain.Baseline, error)
	Save(b *domain.Baseline, path string) error
	Exists(path string) bool
}

// SnapshotStorage defines the interface for baseline snapshot history persistence.
// Application layer depends on this interface, not on concrete implementations.
type SnapshotStorage interface {
	// SaveSnapshot writes a timestamped snapshot to the specified directory.
	SaveSnapshot(dir string, snapshot domain.BaselineSnapshot) error
	// ListSnapshots returns all snapshots sorted by time, newest first.
	ListSnapshots(dir string) ([]domain.BaselineSnapshot, error)
	// LatestSnapshot returns the most recent snapshot, or nil if none exist.
	LatestSnapshot(dir string) (*domain.BaselineSnapshot, error)
}

// TrackStorage defines the interface for baseline track persistence.
type TrackStorage interface {
	// SaveTrack writes the track to the specified path.
	SaveTrack(path string, track domain.BaselineTrack) error
	// LoadTrack reads the track from the specified path. Returns nil if file doesn't exist.
	LoadTrack(path string) (*domain.BaselineTrack, error)
}
