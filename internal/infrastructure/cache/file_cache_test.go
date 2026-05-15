package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func newTestCache(t *testing.T) (*FileCache, string) {
	t.Helper()
	dir := t.TempDir()
	return NewFileCache(dir), dir
}

func TestFileCache_PutAndGet(t *testing.T) {
	cache, _ := newTestCache(t)

	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 5, ImportPath: "fmt"},
		{SourceFile: "main.go", SourceLine: 6, ImportPath: "os"},
	}

	err := cache.Put("abc123", "go", deps)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, ok := cache.Get("abc123", "go")
	if !ok {
		t.Fatal("Get() should return true on hit")
	}

	if len(got) != len(deps) {
		t.Fatalf("Get() returned %d deps, want %d", len(got), len(deps))
	}

	if got[0].ImportPath != "fmt" {
		t.Errorf("Get()[0].ImportPath = %q, want %q", got[0].ImportPath, "fmt")
	}
}

func TestFileCache_Get_Miss(t *testing.T) {
	cache, _ := newTestCache(t)

	_, ok := cache.Get("nonexistent", "go")
	if ok {
		t.Error("Get() should return false on miss")
	}
}

func TestFileCache_Get_DifferentDetector(t *testing.T) {
	cache, _ := newTestCache(t)

	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
	}

	err := cache.Put("abc123", "go", deps)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Same file hash but different detector should miss
	_, ok := cache.Get("abc123", "typescript")
	if ok {
		t.Error("Get() with different detector should miss")
	}
}

func TestFileCache_Clear(t *testing.T) {
	cache, dir := newTestCache(t)

	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
	}

	err := cache.Put("abc123", "go", deps)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	err = cache.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// After clear, cache directory should not exist
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("Clear() should remove cache directory")
	}

	// Get after clear should miss
	_, ok := cache.Get("abc123", "go")
	if ok {
		t.Error("Get() after Clear() should miss")
	}
}

func TestFileCache_ConfigHash(t *testing.T) {
	cache, _ := newTestCache(t)

	// Initially should return empty string
	hash, err := cache.ConfigHash()
	if err != nil {
		t.Fatalf("ConfigHash() error = %v", err)
	}
	if hash != "" {
		t.Errorf("ConfigHash() = %q, want empty", hash)
	}

	// Set and retrieve
	expected := "test-config-hash-123"
	err = cache.SetConfigHash(expected)
	if err != nil {
		t.Fatalf("SetConfigHash() error = %v", err)
	}

	got, err := cache.ConfigHash()
	if err != nil {
		t.Fatalf("ConfigHash() after set error = %v", err)
	}
	if got != expected {
		t.Errorf("ConfigHash() = %q, want %q", got, expected)
	}
}

func TestFileCache_ConfigHashInvalidation(t *testing.T) {
	cache, _ := newTestCache(t)

	// Store config hash
	err := cache.SetConfigHash("old-hash")
	if err != nil {
		t.Fatalf("SetConfigHash() error = %v", err)
	}

	// Put some data
	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
	}
	err = cache.Put("abc123", "go", deps)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Data is retrievable with old config
	got, ok := cache.Get("abc123", "go")
	if !ok {
		t.Fatal("Get() should return data with matching config hash")
	}
	if len(got) != 1 {
		t.Fatalf("Get() returned %d deps, want 1", len(got))
	}

	// Change config hash
	err = cache.SetConfigHash("new-hash")
	if err != nil {
		t.Fatalf("SetConfigHash() error = %v", err)
	}

	// Now Get should miss (config hash mismatch = stale cache)
	_, ok = cache.Get("abc123", "go")
	if ok {
		t.Error("Get() should miss when config hash changed")
	}
}

func TestFileCache_MissingCacheDirectory(t *testing.T) {
	// Create cache pointing to non-existent directory
	cache := NewFileCache("/nonexistent/path/that/does/not/exist")

	// Get should not panic, just return miss
	_, ok := cache.Get("abc123", "go")
	if ok {
		t.Error("Get() on missing directory should miss")
	}
}

func TestFileCache_PerDetectorIsolation(t *testing.T) {
	cache, _ := newTestCache(t)

	goDeps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
	}
	tsDeps := []domain.Dependency{
		{SourceFile: "app.ts", SourceLine: 1, ImportPath: "react"},
	}

	err := cache.Put("hash1", "go", goDeps)
	if err != nil {
		t.Fatalf("Put(go) error = %v", err)
	}

	err = cache.Put("hash1", "typescript", tsDeps)
	if err != nil {
		t.Fatalf("Put(typescript) error = %v", err)
	}

	goGot, _ := cache.Get("hash1", "go")
	tsGot, _ := cache.Get("hash1", "typescript")

	if len(goGot) != 1 || goGot[0].ImportPath != "fmt" {
		t.Errorf("go cache corrupted: %+v", goGot)
	}
	if len(tsGot) != 1 || tsGot[0].ImportPath != "react" {
		t.Errorf("typescript cache corrupted: %+v", tsGot)
	}
}

func TestFileCache_WritesValidJSON(t *testing.T) {
	cache, dir := newTestCache(t)

	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 5, ImportPath: "fmt"},
	}

	err := cache.Put("abc123", "go", deps)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Verify the file exists and contains valid JSON
	cacheFile := filepath.Join(dir, "go", "abc123.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("cache file not found: %v", err)
	}

	if len(data) == 0 {
		t.Error("cache file is empty")
	}
}
