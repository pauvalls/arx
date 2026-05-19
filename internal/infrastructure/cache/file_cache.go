package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// cacheEntry represents a single cached dependency result.
type cacheEntry struct {
	FileHash     string             `json:"file_hash"`
	ConfigHash   string             `json:"config_hash"`
	DetectorName string             `json:"detector_name"`
	Dependencies []domain.Dependency `json:"dependencies"`
	Timestamp    time.Time          `json:"timestamp"`
}

// FileCache implements the ports.Cache interface using JSON files on disk.
// Cache entries are stored in {root}/{detector_name}/{file_hash}.json.
// The config hash is stored in {root}/config-hash for invalidation.
type FileCache struct {
	root string
}

// NewFileCache creates a new FileCache with the given root directory.
func NewFileCache(root string) *FileCache {
	return &FileCache{root: root}
}

// Get returns cached dependencies for a given file hash and detector.
// Returns the dependencies and true on hit, nil and false on miss.
// A miss occurs if the config hash has changed since the entry was stored.
func (c *FileCache) Get(fileHash string, detectorName string) ([]domain.Dependency, bool) {
	// Read current config hash
	currentConfigHash, err := c.ConfigHash()
	if err != nil {
		// If we can't read config hash, treat as miss
		return nil, false
	}

	entry, ok := c.readEntry(fileHash, detectorName)
	if !ok {
		return nil, false
	}

	// Config hash mismatch = stale cache
	if entry.ConfigHash != currentConfigHash {
		return nil, false
	}

	return entry.Dependencies, true
}

// Put stores dependencies in the cache for a file hash and detector.
func (c *FileCache) Put(fileHash string, detectorName string, deps []domain.Dependency) error {
	configHash, err := c.ConfigHash()
	if err != nil {
		configHash = ""
	}

	entry := cacheEntry{
		FileHash:     fileHash,
		ConfigHash:   configHash,
		DetectorName: detectorName,
		Dependencies: deps,
		Timestamp:    time.Now(),
	}

	return c.writeEntry(entry)
}

// SetConfigHash stores the current config hash for invalidation checks.
func (c *FileCache) SetConfigHash(hash string) error {
	if err := os.MkdirAll(c.root, 0o755); err != nil {
		return err
	}
	path := filepath.Join(c.root, "config-hash")
	return os.WriteFile(path, []byte(hash), 0o644)
}

// ConfigHash returns the stored config hash, or empty string if not set.
func (c *FileCache) ConfigHash() (string, error) {
	path := filepath.Join(c.root, "config-hash")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Clear removes all cached entries by deleting the cache directory.
func (c *FileCache) Clear() error {
	return os.RemoveAll(c.root)
}

// GetFile returns cached dependencies for a single file identified by key.
// Returns the dependencies and true on hit, nil and false on miss.
func (c *FileCache) GetFile(key ports.FileCacheKey) ([]domain.Dependency, bool) {
	// Read current config hash
	currentConfigHash, err := c.ConfigHash()
	if err != nil {
		return nil, false
	}

	entry, ok := c.readEntry(key.ContentHash, key.DetectorName)
	if !ok {
		return nil, false
	}

	// Config hash mismatch = stale cache
	if entry.ConfigHash != currentConfigHash {
		return nil, false
	}

	return entry.Dependencies, true
}

// PutFile stores dependencies for a single file identified by key.
func (c *FileCache) PutFile(key ports.FileCacheKey, deps []domain.Dependency) error {
	configHash, err := c.ConfigHash()
	if err != nil {
		configHash = ""
	}

	entry := cacheEntry{
		FileHash:     key.ContentHash,
		ConfigHash:   configHash,
		DetectorName: key.DetectorName,
		Dependencies: deps,
		Timestamp:    time.Now(),
	}

	return c.writeEntry(entry)
}

// encodePath encodes a relative path for use as a filesystem directory name.
// Replaces directory separators with underscores for safe filesystem paths.
func encodePath(path string) string {
	s := filepath.ToSlash(path)
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

func (c *FileCache) detectorDir(detectorName string) string {
	return filepath.Join(c.root, detectorName)
}

func (c *FileCache) entryPath(fileHash string, detectorName string) string {
	return filepath.Join(c.detectorDir(detectorName), fileHash+".json")
}

func (c *FileCache) readEntry(fileHash string, detectorName string) (cacheEntry, bool) {
	path := c.entryPath(fileHash, detectorName)
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}, false
	}

	return entry, true
}

func (c *FileCache) writeEntry(entry cacheEntry) error {
	dir := c.detectorDir(entry.DetectorName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := c.entryPath(entry.FileHash, entry.DetectorName)

	// Atomic write: write to temp file, then rename
	tmpPath := path + ".tmp"
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
