package wasm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cache provides thread-safe caching for WASM compiled modules.
type Cache struct {
	mu       sync.RWMutex
	entries  map[string]interface{}
	cacheDir string
}

// NewCache creates a new WASM module cache with the given disk cache directory.
// If cacheDir is empty, disk caching is disabled.
func NewCache(cacheDir string) *Cache {
	c := &Cache{
		entries:  make(map[string]interface{}),
		cacheDir: cacheDir,
	}
	// Load existing cache entries from disk
	if cacheDir != "" {
		c.loadFromDisk()
	}
	return c
}

// cacheKey computes the SHA-256 hex digest of the given data.
func cacheKey(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// Get returns a cached entry by its data's SHA-256 hash.
func (c *Cache) Get(data []byte) (interface{}, bool) {
	key := cacheKey(data)
	c.mu.RLock()
	val, ok := c.entries[key]
	c.mu.RUnlock()
	return val, ok
}

// Set stores an entry in the cache, indexed by the SHA-256 hash of the data.
// If cacheDir is configured, it also persists the raw data to disk.
func (c *Cache) Set(data []byte, val interface{}) {
	key := cacheKey(data)
	c.mu.Lock()
	c.entries[key] = val
	c.mu.Unlock()

	if c.cacheDir != "" {
		c.persistToDisk(key, data)
	}
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(data []byte) {
	key := cacheKey(data)
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// persistToDisk saves the raw WASM data to the disk cache.
func (c *Cache) persistToDisk(key string, data []byte) {
	dir := filepath.Join(c.cacheDir, key)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return // silently fail on disk cache errors
	}
	// Save raw bytes
	if err := os.WriteFile(filepath.Join(dir, "wasm.bin"), data, 0644); err != nil {
		return
	}
}

// loadFromDisk loads cache entries from the disk cache directory.
func (c *Cache) loadFromDisk() {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return // directory doesn't exist yet
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		hash := entry.Name()
		// Read the raw wasm file to confirm it's valid
		wasmPath := filepath.Join(c.cacheDir, hash, "wasm.bin")
		data, err := os.ReadFile(wasmPath)
		if err != nil {
			continue
		}
		// Verify hash matches
		expectedHash := cacheKey(data)
		if expectedHash != hash {
			continue // corrupted or tampered
		}
		// Just mark as cached — the compiled module will be created on demand
		c.entries[hash] = struct{}{}
	}
}
