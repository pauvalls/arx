package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/domain"
)

// Storage handles reading and writing baseline files to disk.
type Storage struct{}

// NewStorage creates a new Storage instance.
func NewStorage() *Storage {
	return &Storage{}
}

// Save writes a baseline to the specified path using atomic writes.
func (s *Storage) Save(baseline *domain.Baseline, path string) error {
	if baseline == nil {
		return fmt.Errorf("baseline is nil")
	}

	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Atomic write: write to temp file, then rename
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // clean up temp file on failure
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// Load reads a baseline from the specified path.
// Returns nil, nil if the file does not exist.
func (s *Storage) Load(path string) (*domain.Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading baseline file: %w", err)
	}

	var baseline domain.Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("parsing baseline file: %w", err)
	}

	return &baseline, nil
}

// Exists checks if a baseline file exists at the specified path.
func (s *Storage) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
