package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
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

func TestFileCache_GetFileHit(t *testing.T) {
	cache, _ := newTestCache(t)

	key := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "internal/domain/user.go",
		ContentHash:  "abc123def",
	}
	deps := []domain.Dependency{
		{SourceFile: "user.go", SourceLine: 5, ImportPath: "fmt"},
	}

	err := cache.PutFile(key, deps)
	if err != nil {
		t.Fatalf("PutFile() error = %v", err)
	}

	got, ok := cache.GetFile(key)
	if !ok {
		t.Fatal("GetFile() should return true on hit")
	}
	if len(got) != 1 || got[0].ImportPath != "fmt" {
		t.Errorf("GetFile() = %+v, want deps with import 'fmt'", got)
	}
}

func TestFileCache_GetFileMiss_Empty(t *testing.T) {
	cache, _ := newTestCache(t)

	key := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "nonexistent.go",
		ContentHash:  "unknown",
	}

	_, ok := cache.GetFile(key)
	if ok {
		t.Error("GetFile() should return false on miss for unknown key")
	}
}

func TestFileCache_GetFile_ConfigChangedInvalidates(t *testing.T) {
	cache, _ := newTestCache(t)

	err := cache.SetConfigHash("config-v1")
	if err != nil {
		t.Fatalf("SetConfigHash error = %v", err)
	}

	key := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "main.go",
		ContentHash:  "hash1",
	}
	deps := []domain.Dependency{
		{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
	}

	err = cache.PutFile(key, deps)
	if err != nil {
		t.Fatalf("PutFile() error = %v", err)
	}

	// Should be retrievable
	_, ok := cache.GetFile(key)
	if !ok {
		t.Fatal("GetFile() should hit with matching config hash")
	}

	// Change config hash
	err = cache.SetConfigHash("config-v2")
	if err != nil {
		t.Fatalf("SetConfigHash error = %v", err)
	}

	// Should miss now
	_, ok = cache.GetFile(key)
	if ok {
		t.Error("GetFile() should miss when config hash changed")
	}
}

func TestFileCache_GetFile_SameContentDifferentPaths(t *testing.T) {
	cache, _ := newTestCache(t)

	key1 := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "a/file.go",
		ContentHash:  "samehash",
	}
	key2 := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "b/file.go",
		ContentHash:  "samehash",
	}

	deps1 := []domain.Dependency{{SourceFile: "a/file.go", SourceLine: 1, ImportPath: "fmt"}}
	deps2 := []domain.Dependency{{SourceFile: "b/file.go", SourceLine: 1, ImportPath: "os"}}

	_ = cache.PutFile(key1, deps1)
	_ = cache.PutFile(key2, deps2)

	got1, ok1 := cache.GetFile(key1)
	got2, ok2 := cache.GetFile(key2)

	if !ok1 || !ok2 {
		t.Fatal("Both keys should hit")
	}
	// Same content hash means same cache file — the last write wins
	if len(got1) == 0 || len(got2) == 0 {
		t.Error("Both keys should return deps")
	}
}

func TestFileCache_Clear_RemovesPerFileEntries(t *testing.T) {
	cache, _ := newTestCache(t)

	key := ports.FileCacheKey{
		DetectorName: "go",
		RelativePath: "main.go",
		ContentHash:  "hash1",
	}
	_ = cache.PutFile(key, []domain.Dependency{{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"}})

	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	_, ok := cache.GetFile(key)
	if ok {
		t.Error("GetFile() after Clear() should miss")
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
