package wasm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "small data",
			data: []byte("hello wasm"),
		},
		{
			name: "wasm binary data",
			data: []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cacheKey(tt.data)
			expected := fmt.Sprintf("%x", sha256.Sum256(tt.data))
			if key != expected {
				t.Errorf("cacheKey() = %q, want %q", key, expected)
			}
		})
	}
}

func TestNewCache(t *testing.T) {
	c := NewCache("")
	if c == nil {
		t.Fatal("NewCache() returned nil")
	}
	// Default cache dir should be under home/.arx-cache
	if c.cacheDir != "" {
		t.Errorf("NewCache('') should use empty cacheDir, got %q", c.cacheDir)
	}
}

func TestCacheGetSet(t *testing.T) {
	c := NewCache("")
	data := []byte("test wasm data")

	// Initially, entry should not exist
	_, ok := c.Get(data)
	if ok {
		t.Error("Get() should return false for uncached entry")
	}

	// Set the entry
	c.Set(data, struct{}{})

	// Now it should exist
	val, ok := c.Get(data)
	if !ok {
		t.Error("Get() should return true for cached entry")
	}
	if val == nil {
		t.Error("Get() should return non-nil value")
	}
}

func TestCacheDelete(t *testing.T) {
	c := NewCache("")
	data := []byte("test wasm data")
	c.Set(data, struct{}{})

	_, ok := c.Get(data)
	if !ok {
		t.Fatal("Get() should return true after Set()")
	}

	c.Delete(data)
	_, ok = c.Get(data)
	if ok {
		t.Error("Get() should return false after Delete()")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := NewCache("")
	var wg sync.WaitGroup

	// Concurrently set and get
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf("data-%d", i))
			c.Set(data, struct{}{})
			_, ok := c.Get(data)
			if !ok {
				t.Errorf("concurrent Get() failed for data-%d", i)
			}
		}(i)
	}
	wg.Wait()

	// Verify all entries are accessible
	for i := 0; i < 20; i++ {
		data := []byte(fmt.Sprintf("data-%d", i))
		_, ok := c.Get(data)
		if !ok {
			t.Errorf("Get() should return true for data-%d after concurrent access", i)
		}
	}
}

func TestCacheDiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".arx-cache", "policies")
	c := NewCache(cacheDir)

	data := []byte("persistent wasm data")

	// Set value
	c.Set(data, struct{}{})

	// Verify in-memory
	_, ok := c.Get(data)
	if !ok {
		t.Fatal("Get() should return true after Set()")
	}

	// Verify disk storage exists
	hash := cacheKey(data)
	diskPath := filepath.Join(cacheDir, hash)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Errorf("disk cache file should exist at %s", diskPath)
	}
}

func TestCacheDiskPersists(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".arx-cache", "policies")

	// Create first cache instance and set data
	data := []byte("persistent data")
	hash := cacheKey(data)

	c1 := NewCache(cacheDir)
	c1.Set(data, struct{}{})

	// Create a NEW cache instance (simulates restart)
	c2 := NewCache(cacheDir)

	// Data should be loaded from disk
	val, ok := c2.Get(data)
	if !ok {
		t.Errorf("Get() should return true after cache reload (disk persistence)")
	}
	if val == nil {
		t.Error("Get() should return non-nil value after cache reload")
	}

	// Verify disk file exists
	diskPath := filepath.Join(cacheDir, hash)
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Errorf("disk cache file should exist at %s", diskPath)
	}
}

func TestCacheDiskInvalidHashDir(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".arx-cache", "policies")

	c := NewCache(cacheDir)
	data := []byte("test data")

	// Write a corrupt entry to disk
	hash := cacheKey(data)
	badDir := filepath.Join(cacheDir, hash)
	if err := os.MkdirAll(badDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "wasm.bin"), []byte("corrupted"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load from disk - should not panic
	c.loadFromDisk()

	// Should still work
	c.Set(data, struct{}{})
	_, ok := c.Get(data)
	if !ok {
		t.Error("Get() should return true after disk corruption recovery")
	}
}
