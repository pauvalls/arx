package ports

import "github.com/pauvalls/arx/internal/domain"

// FileCacheKey identifies a single file in the per-file cache.
type FileCacheKey struct {
	DetectorName string `json:"detector"`
	RelativePath string `json:"path"`
	ContentHash  string `json:"sha256"`
}

// Cache defines the interface for caching detector results.
// Cache entries are keyed by file hash and detector name, with config hash
// used for bulk invalidation when arx.yaml changes.
type Cache interface {
	// Get returns cached dependencies for a given file hash and detector.
	// Returns the dependencies and true on hit, nil and false on miss.
	Get(fileHash string, detectorName string) ([]domain.Dependency, bool)

	// Put stores dependencies in the cache for a file hash and detector.
	Put(fileHash string, detectorName string, deps []domain.Dependency) error

	// GetFile returns cached dependencies for a single file identified by key.
	// Returns the dependencies and true on hit, nil and false on miss.
	GetFile(key FileCacheKey) ([]domain.Dependency, bool)

	// PutFile stores dependencies for a single file identified by key.
	PutFile(key FileCacheKey, deps []domain.Dependency) error

	// SetConfigHash stores the current config hash for invalidation checks.
	SetConfigHash(hash string) error

	// ConfigHash returns the stored config hash, or empty string if not set.
	ConfigHash() (string, error)

	// Clear removes all cached entries.
	Clear() error
}
