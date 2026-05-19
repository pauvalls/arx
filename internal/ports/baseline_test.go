package ports

import (
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// Ensure SnapshotStorage interface compiles with a mock.
type mockSnapshotStorage struct {
	snapshots map[string][]domain.BaselineSnapshot
}

func (m *mockSnapshotStorage) SaveSnapshot(dir string, snapshot domain.BaselineSnapshot) error {
	if m.snapshots == nil {
		m.snapshots = make(map[string][]domain.BaselineSnapshot)
	}
	m.snapshots[dir] = append(m.snapshots[dir], snapshot)
	return nil
}

func (m *mockSnapshotStorage) ListSnapshots(dir string) ([]domain.BaselineSnapshot, error) {
	if m.snapshots == nil {
		return nil, nil
	}
	return m.snapshots[dir], nil
}

func (m *mockSnapshotStorage) LatestSnapshot(dir string) (*domain.BaselineSnapshot, error) {
	snapshots, err := m.ListSnapshots(dir)
	if err != nil || len(snapshots) == 0 {
		return nil, err
	}
	return &snapshots[len(snapshots)-1], nil
}

func TestSnapshotStorage_InterfaceCompiles(t *testing.T) {
	// Verify the mock satisfies the interface at compile time
	var _ SnapshotStorage = (*mockSnapshotStorage)(nil)
}

func TestSnapshotStorage_MockBehavior(t *testing.T) {
	mock := &mockSnapshotStorage{}
	dir := t.TempDir()

	// ListSnapshots on empty storage
	snapshots, err := mock.ListSnapshots(dir)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("got %d, want 0", len(snapshots))
	}

	// Save a snapshot
	snapshot := domain.BaselineSnapshot{
		CreatedAt:  time.Now(),
		TotalCount: 5,
	}
	if err := mock.SaveSnapshot(dir, snapshot); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// LatestSnapshot returns it
	latest, err := mock.LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if latest == nil {
		t.Fatal("LatestSnapshot should not be nil")
	}
	if latest.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", latest.TotalCount)
	}

	// ListSnapshots returns saved
	snapshots, err = mock.ListSnapshots(dir)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("got %d snapshots, want 1", len(snapshots))
	}
}

func TestSnapshotStorage_MockLatestNil(t *testing.T) {
	mock := &mockSnapshotStorage{}

	latest, err := mock.LatestSnapshot("/nonexistent")
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if latest != nil {
		t.Error("LatestSnapshot should be nil for empty storage")
	}
}
