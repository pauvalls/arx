package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/pauvalls/arx/internal/domain"
)

// TrackStorage implements ports.TrackStorage using a JSON file.
// Thread-safe with mutex.
type TrackStorage struct {
	mu sync.Mutex
}

// NewTrackStorage creates a new TrackStorage.
func NewTrackStorage() *TrackStorage {
	return &TrackStorage{}
}

// SaveTrack writes the track to the specified path.
func (s *TrackStorage) SaveTrack(path string, track domain.BaselineTrack) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(track, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling track: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing track file: %w", err)
	}

	return nil
}

// LoadTrack reads the track from the specified path.
// Returns nil, nil if the file does not exist.
func (s *TrackStorage) LoadTrack(path string) (*domain.BaselineTrack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading track file: %w", err)
	}

	var track domain.BaselineTrack
	if err := json.Unmarshal(data, &track); err != nil {
		return nil, fmt.Errorf("parsing track file: %w", err)
	}

	return &track, nil
}
