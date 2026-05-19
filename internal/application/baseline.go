package application

import (
	"path/filepath"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// DefaultBaselineFile is the default filename for the baseline.
const DefaultBaselineFile = ".arx-baseline.json"

// BaselineService handles baseline generation, loading, filtering, snapshot history, and auto-refresh.
type BaselineService struct {
	storage       ports.BaselineStorage
	snapshotStore ports.SnapshotStorage
	trackStore    ports.TrackStorage
}

// NewBaselineService creates a new BaselineService.
func NewBaselineService(storage ports.BaselineStorage) *BaselineService {
	return &BaselineService{
		storage: storage,
	}
}

// NewBaselineServiceFull creates a BaselineService with all dependencies.
func NewBaselineServiceFull(storage ports.BaselineStorage, snapshots ports.SnapshotStorage, tracks ports.TrackStorage) *BaselineService {
	return &BaselineService{
		storage:       storage,
		snapshotStore: snapshots,
		trackStore:    tracks,
	}
}

// Generate creates a baseline from violations and config hash.
func (s *BaselineService) Generate(violations []domain.Violation, configHash string) *domain.Baseline {
	return domain.GenerateBaseline(violations, configHash)
}

// Load reads a baseline from the specified path.
// Returns nil, nil if the file does not exist.
func (s *BaselineService) Load(path string) (*domain.Baseline, error) {
	return s.storage.Load(path)
}

// Save writes a baseline to the specified path.
func (s *BaselineService) Save(b *domain.Baseline, path string) error {
	return s.storage.Save(b, path)
}

// Exists checks if a baseline file exists at the specified path.
func (s *BaselineService) Exists(path string) bool {
	return s.storage.Exists(path)
}

// FilterViolations returns only violations NOT in the baseline.
// Returns all violations unchanged when baseline is nil.
func (s *BaselineService) FilterViolations(violations []domain.Violation, b *domain.Baseline) []domain.Violation {
	if b == nil {
		return violations
	}
	return b.Filter(violations)
}

// DefaultPath returns the default baseline path for a project root.
func (s *BaselineService) DefaultPath(projectRoot string) string {
	return projectRoot + "/" + DefaultBaselineFile
}

// historyDir returns the baseline history directory path for a project root.
func (s *BaselineService) historyDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".arx-baseline-history")
}

// trackPath returns the baseline track file path for a project root.
func (s *BaselineService) trackPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".arx-cache", "baseline-track.json")
}

// Snapshot creates a snapshot of the current baseline, saves it to history,
// and returns the snapshot.
func (s *BaselineService) Snapshot(projectRoot string) (*domain.BaselineSnapshot, error) {
	if s.snapshotStore == nil {
		return nil, nil
	}

	// Load the current baseline
	baseline, err := s.storage.Load(s.DefaultPath(projectRoot))
	if err != nil {
		return nil, err
	}

	var violations []domain.Violation
	if baseline != nil {
		for _, bv := range baseline.Violations {
			violations = append(violations, domain.Violation{
				ID:          bv.Fingerprint(),
				RuleID:      bv.RuleID,
				File:        bv.File,
				SourceLayer: bv.SourceLayer,
				TargetLayer: bv.TargetLayer,
				Import:      bv.Import,
			})
		}
	}

	snapshot := domain.NewBaselineSnapshot(violations)
	if err := s.snapshotStore.SaveSnapshot(s.historyDir(projectRoot), snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// DiffFromSnapshot compares current violations against a snapshot's violations
// using CompareViolations. Returns added and resolved violations.
func (s *BaselineService) DiffFromSnapshot(snapshot domain.BaselineSnapshot, currentViolations []domain.Violation) (added, resolved []domain.Violation, err error) {
	result := CompareViolations(snapshot.Violations, currentViolations)
	return result.Added, result.Resolved, nil
}

// LoadTrack reads the baseline track from the specified path.
func (s *BaselineService) LoadTrack(path string) (*domain.BaselineTrack, error) {
	if s.trackStore == nil {
		return nil, nil
	}
	return s.trackStore.LoadTrack(path)
}

// SaveTrack writes the baseline track to the specified path.
func (s *BaselineService) SaveTrack(path string, track domain.BaselineTrack) error {
	if s.trackStore == nil {
		return nil
	}
	return s.trackStore.SaveTrack(path, track)
}

// LatestSnapshot returns the most recent snapshot from history, or nil if none exist.
func (s *BaselineService) LatestSnapshot(projectRoot string) (*domain.BaselineSnapshot, error) {
	if s.snapshotStore == nil {
		return nil, nil
	}
	return s.snapshotStore.LatestSnapshot(s.historyDir(projectRoot))
}

// Trend reads all snapshots and returns TrendPoints computed from them.
func (s *BaselineService) Trend(projectRoot string) ([]domain.TrendPoint, error) {
	if s.snapshotStore == nil {
		return nil, nil
	}

	snapshots, err := s.snapshotStore.ListSnapshots(s.historyDir(projectRoot))
	if err != nil {
		return nil, err
	}

	trend := make([]domain.TrendPoint, len(snapshots))
	for i, snap := range snapshots {
		trend[i] = domain.TrendPointFromSnapshot(snap)
	}

	return trend, nil
}

// DefaultRefreshThreshold is the default number of consecutive clean checks before auto-refresh.
const DefaultRefreshThreshold = 3

// AutoRefresh increments or resets the consecutive clean counter based on violations.
// When threshold is met (consecutive clean >= threshold and threshold > 0), triggers refresh
// and resets the counter. Returns the updated track, whether refresh was triggered, and any error.
func (s *BaselineService) AutoRefresh(track domain.BaselineTrack, threshold int) (domain.BaselineTrack, bool, error) {
	track.ConsecutiveClean++

	if threshold > 0 && track.ConsecutiveClean >= threshold {
		track.ConsecutiveClean = 0
		track.SnapshotCount++
		track.LastSnapshot = track.LastCheck
		return track, true, nil
	}

	return track, false, nil
}
