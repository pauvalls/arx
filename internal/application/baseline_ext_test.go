package application

import (
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// mockSnapshotStore implements ports.SnapshotStorage for testing.
type mockSnapshotStore struct {
	snapshots []domain.BaselineSnapshot
}

func (m *mockSnapshotStore) SaveSnapshot(dir string, snapshot domain.BaselineSnapshot) error {
	m.snapshots = append(m.snapshots, snapshot)
	return nil
}

func (m *mockSnapshotStore) ListSnapshots(dir string) ([]domain.BaselineSnapshot, error) {
	// Return a copy sorted newest first (matching real implementation behavior)
	sorted := make([]domain.BaselineSnapshot, len(m.snapshots))
	copy(sorted, m.snapshots)
	// The real storage sorts newest first; we store in append order so reverse
	for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
		sorted[i], sorted[j] = sorted[j], sorted[i]
	}
	return sorted, nil
}

func (m *mockSnapshotStore) LatestSnapshot(dir string) (*domain.BaselineSnapshot, error) {
	if len(m.snapshots) == 0 {
		return nil, nil
	}
	return &m.snapshots[len(m.snapshots)-1], nil
}

// mockTrackStore implements ports.TrackStorage for testing.
type mockTrackStore struct {
	data map[string]domain.BaselineTrack
}

func (m *mockTrackStore) SaveTrack(path string, track domain.BaselineTrack) error {
	if m.data == nil {
		m.data = make(map[string]domain.BaselineTrack)
	}
	m.data[path] = track
	return nil
}

func (m *mockTrackStore) LoadTrack(path string) (*domain.BaselineTrack, error) {
	if m.data == nil {
		return nil, nil
	}
	track, ok := m.data[path]
	if !ok {
		return nil, nil
	}
	return &track, nil
}

func newTestBaselineServiceExt() *BaselineService {
	return NewBaselineServiceFull(
		&mockBaselineStore{},
		&mockSnapshotStore{},
		&mockTrackStore{},
	)
}

func TestBaselineService_Snapshot(t *testing.T) {
	svc := newTestBaselineServiceExt()
	baselinePath := t.TempDir()

	// Need a baseline with violations to snapshot
	baseline := &domain.Baseline{
		Version:    "1.0",
		ConfigHash: "hash-123",
		Violations: []domain.BaselineViolation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x", File: "a.go"},
			{RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y", File: "b.go"},
		},
	}

	// First save a baseline so Snapshot has something to load
	svc.storage.(*mockBaselineStore).data = map[string]*domain.Baseline{
		svc.DefaultPath(baselinePath): baseline,
	}

	snapshot, err := svc.Snapshot(baselinePath)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if snapshot.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", snapshot.TotalCount)
	}
	if len(snapshot.Violations) != 2 {
		t.Errorf("len(Violations) = %d, want 2", len(snapshot.Violations))
	}
	if snapshot.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestBaselineService_SnapshotNoBaseline(t *testing.T) {
	svc := newTestBaselineServiceExt()
	baselinePath := t.TempDir()

	// No baseline exists
	snapshot, err := svc.Snapshot(baselinePath)
	if err != nil {
		t.Fatalf("Snapshot without baseline: %v", err)
	}

	if snapshot.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", snapshot.TotalCount)
	}
	if len(snapshot.Violations) != 0 {
		t.Errorf("len(Violations) = %d, want 0", len(snapshot.Violations))
	}
}

func TestBaselineService_DiffFromSnapshot(t *testing.T) {
	svc := newTestBaselineServiceExt()

	snapshot := domain.BaselineSnapshot{
		Violations: []domain.Violation{
			{ID: "V-001", RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x"},
			{ID: "V-002", RuleID: "R002", SourceLayer: "application", TargetLayer: "domain", Import: "y"},
		},
		CreatedAt:  time.Now(),
		TotalCount: 2,
	}

	current := []domain.Violation{
		{ID: "V-001", RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x"},
		{ID: "V-003", RuleID: "R003", SourceLayer: "domain", TargetLayer: "presentation", Import: "z"},
	}

	added, resolved, err := svc.DiffFromSnapshot(snapshot, current)
	if err != nil {
		t.Fatalf("DiffFromSnapshot: %v", err)
	}

	if len(added) != 1 {
		t.Errorf("added = %d, want 1", len(added))
	}
	if len(resolved) != 1 {
		t.Errorf("resolved = %d, want 1", len(resolved))
	}

	// V-003 is new, V-002 was resolved
	if len(added) > 0 && added[0].ID != "V-003" {
		t.Errorf("added[0].ID = %q, want %q", added[0].ID, "V-003")
	}
	if len(resolved) > 0 && resolved[0].ID != "V-002" {
		t.Errorf("resolved[0].ID = %q, want %q", resolved[0].ID, "V-002")
	}
}

func TestBaselineService_DiffFromSnapshotEmpty(t *testing.T) {
	svc := newTestBaselineServiceExt()

	snapshot := domain.BaselineSnapshot{
		Violations: nil,
		CreatedAt:  time.Now(),
	}

	current := []domain.Violation{
		{ID: "V-001", RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x"},
	}

	added, resolved, err := svc.DiffFromSnapshot(snapshot, current)
	if err != nil {
		t.Fatalf("DiffFromSnapshot: %v", err)
	}

	if len(added) != 1 {
		t.Errorf("added = %d, want 1", len(added))
	}
	if len(resolved) != 0 {
		t.Errorf("resolved = %d, want 0", len(resolved))
	}
}

func TestBaselineService_Trend(t *testing.T) {
	svc := newTestBaselineServiceExt()

	// Need to get a snapshot store with data
	snapStore := svc.snapshotStore.(*mockSnapshotStore)
	now := time.Now()

	snapStore.snapshots = []domain.BaselineSnapshot{
		{
			Violations: []domain.Violation{
				{ID: "V-001", Severity: domain.SeverityError},
				{ID: "V-002", Severity: domain.SeverityWarning},
			},
			CreatedAt:  now.Add(-48 * time.Hour),
			TotalCount: 2,
			SeverityBreakdown: map[domain.Severity]int{
				domain.SeverityError:   1,
				domain.SeverityWarning: 1,
			},
		},
		{
			Violations: []domain.Violation{
				{ID: "V-001", Severity: domain.SeverityError},
			},
			CreatedAt:  now.Add(-24 * time.Hour),
			TotalCount: 1,
			SeverityBreakdown: map[domain.Severity]int{
				domain.SeverityError: 1,
			},
		},
	}

	trend, err := svc.Trend(t.TempDir())
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}

	if len(trend) != 2 {
		t.Fatalf("Trend points = %d, want 2", len(trend))
	}

	if trend[0].Total != 1 {
		t.Errorf("first trend Total = %d, want 1", trend[0].Total)
	}
	if trend[0].Errors != 1 {
		t.Errorf("first trend Errors = %d, want 1", trend[0].Errors)
	}
	if trend[1].Total != 2 {
		t.Errorf("second trend Total = %d, want 2", trend[1].Total)
	}
	if trend[1].Warnings != 1 {
		t.Errorf("second trend Warnings = %d, want 1", trend[1].Warnings)
	}
}

func TestBaselineService_TrendEmpty(t *testing.T) {
	svc := newTestBaselineServiceExt()

	trend, err := svc.Trend(t.TempDir())
	if err != nil {
		t.Fatalf("Trend empty: %v", err)
	}
	if len(trend) != 0 {
		t.Errorf("Trend points = %d, want 0", len(trend))
	}
}

func TestBaselineService_AutoRefresh(t *testing.T) {
	t.Run("increments counter below threshold", func(t *testing.T) {
		svc := newTestBaselineServiceExt()
		track := domain.BaselineTrack{ConsecutiveClean: 1}

		updated, triggered, err := svc.AutoRefresh(track, 3)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if triggered {
			t.Error("AutoRefresh should not trigger below threshold")
		}
		if updated.ConsecutiveClean != 2 {
			t.Errorf("ConsecutiveClean = %d, want 2", updated.ConsecutiveClean)
		}
	})

	t.Run("triggers at threshold and resets counter", func(t *testing.T) {
		svc := newTestBaselineServiceExt()
		track := domain.BaselineTrack{ConsecutiveClean: 2}

		updated, triggered, err := svc.AutoRefresh(track, 3)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if !triggered {
			t.Error("AutoRefresh should trigger at threshold")
		}
		if updated.ConsecutiveClean != 0 {
			t.Errorf("After trigger ConsecutiveClean = %d, want 0", updated.ConsecutiveClean)
		}
		if updated.SnapshotCount != 1 {
			t.Errorf("SnapshotCount = %d, want 1", updated.SnapshotCount)
		}
	})

	t.Run("custom threshold value", func(t *testing.T) {
		svc := newTestBaselineServiceExt()
		// AutoRefresh increments BEFORE checking: with threshold=5,
		// ConsecutiveClean=4 triggers (4+1 >= 5), ConsecutiveClean=3 does not (3+1 < 5).
		track := domain.BaselineTrack{ConsecutiveClean: 2}

		updated, triggered, err := svc.AutoRefresh(track, 5)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if triggered {
			t.Error("AutoRefresh should not trigger at 2/5")
		}
		if updated.ConsecutiveClean != 3 {
			t.Errorf("ConsecutiveClean = %d, want 3", updated.ConsecutiveClean)
		}

		// Now at 3/5 — still not triggered
		updated, triggered, err = svc.AutoRefresh(updated, 5)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if triggered {
			t.Error("AutoRefresh should not trigger at 3/5")
		}
		if updated.ConsecutiveClean != 4 {
			t.Errorf("ConsecutiveClean = %d, want 4", updated.ConsecutiveClean)
		}

		// Now at 4/5 — next check triggers
		updated, triggered, err = svc.AutoRefresh(updated, 5)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if !triggered {
			t.Error("AutoRefresh should trigger at 4/5 (4+1 >= 5)")
		}
		if updated.ConsecutiveClean != 0 {
			t.Errorf("After trigger ConsecutiveClean = %d, want 0", updated.ConsecutiveClean)
		}
	})

	t.Run("zero threshold never triggers", func(t *testing.T) {
		svc := newTestBaselineServiceExt()
		track := domain.BaselineTrack{ConsecutiveClean: 100}

		updated, triggered, err := svc.AutoRefresh(track, 0)
		if err != nil {
			t.Fatalf("AutoRefresh: %v", err)
		}
		if triggered {
			t.Error("AutoRefresh should not trigger with zero threshold")
		}
		if updated.ConsecutiveClean != 101 {
			t.Errorf("ConsecutiveClean = %d, want 101", updated.ConsecutiveClean)
		}
	})
}

func TestBaselineService_AutoRefreshWithViolations(t *testing.T) {
	svc := newTestBaselineServiceExt()
	track := domain.BaselineTrack{ConsecutiveClean: 2}

	// Simulate a check with violations: counter resets
	track.ConsecutiveClean = 0

	updated, triggered, err := svc.AutoRefresh(track, 3)
	if err != nil {
		t.Fatalf("AutoRefresh: %v", err)
	}
	if triggered {
		t.Error("AutoRefresh should not trigger after violations")
	}
	if updated.ConsecutiveClean != 1 {
		t.Errorf("ConsecutiveClean = %d, want 1", updated.ConsecutiveClean)
	}
}
