package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	arxcache "github.com/pauvalls/arx/internal/infrastructure/cache"
	"github.com/pauvalls/arx/internal/ports"
)

// mockCache implements ports.Cache for testing
type mockCache struct {
	entries    map[string][]domain.Dependency
	configHash string
	getCalls   int
	putCalls   int
}

func newMockCache() *mockCache {
	return &mockCache{
		entries: make(map[string][]domain.Dependency),
	}
}

func (m *mockCache) cacheKey(fileHash, detectorName string) string {
	return detectorName + ":" + fileHash
}

func (m *mockCache) Get(fileHash string, detectorName string) ([]domain.Dependency, bool) {
	m.getCalls++
	key := m.cacheKey(fileHash, detectorName)
	deps, ok := m.entries[key]
	return deps, ok
}

func (m *mockCache) Put(fileHash string, detectorName string, deps []domain.Dependency) error {
	m.putCalls++
	key := m.cacheKey(fileHash, detectorName)
	m.entries[key] = deps
	return nil
}

func (m *mockCache) SetConfigHash(hash string) error {
	m.configHash = hash
	return nil
}

func (m *mockCache) ConfigHash() (string, error) {
	return m.configHash, nil
}

func (m *mockCache) GetFile(key ports.FileCacheKey) ([]domain.Dependency, bool) {
	m.getCalls++
	ck := key.DetectorName + ":file:" + key.RelativePath + ":" + key.ContentHash
	deps, ok := m.entries[ck]
	return deps, ok
}

func (m *mockCache) PutFile(key ports.FileCacheKey, deps []domain.Dependency) error {
	m.putCalls++
	ck := key.DetectorName + ":file:" + key.RelativePath + ":" + key.ContentHash
	m.entries[ck] = deps
	return nil
}

func (m *mockCache) Clear() error {
	m.entries = make(map[string][]domain.Dependency)
	return nil
}

// countingDetector wraps a mockDetector to track ExtractImports calls
type countingDetector struct {
	*mockDetector
	extractCalls int
}

func (c *countingDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	c.extractCalls++
	return c.mockDetector.ExtractImports(ctx, projectRoot, layers)
}

func setupTempProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir error: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write error: %v", err)
		}
	}
	return dir
}

func TestRunDetectorsCached_CacheMiss(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	projectDir := setupTempProject(t, map[string]string{
		"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n",
	})

	detector := &countingDetector{
		mockDetector: &mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps: []domain.Dependency{
				{SourceFile: "main.go", SourceLine: 3, ImportPath: "fmt"},
			},
		},
	}

	deps, err := RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, cache)
	if err != nil {
		t.Fatalf("RunDetectorsCached() error = %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("RunDetectorsCached() returned %d deps, want 1", len(deps))
	}

	// On miss, ExtractImports should be called
	if detector.extractCalls != 1 {
		t.Errorf("ExtractImports called %d times on miss, want 1", detector.extractCalls)
	}

	// Put should have been called to store the result
	if cache.putCalls != 1 {
		t.Errorf("Cache.Put called %d times, want 1", cache.putCalls)
	}
}

func TestRunDetectorsCached_CacheHit(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	projectDir := setupTempProject(t, map[string]string{
		"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hello\") }\n",
	})

	detector := &countingDetector{
		mockDetector: &mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps: []domain.Dependency{
				{SourceFile: "main.go", SourceLine: 3, ImportPath: "fmt"},
			},
		},
	}

	// First run: cache miss
	deps1, err := RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, cache)
	if err != nil {
		t.Fatalf("First run error = %v", err)
	}

	// Second run: cache hit
	deps2, err := RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, cache)
	if err != nil {
		t.Fatalf("Second run error = %v", err)
	}

	// Both runs should return the same deps
	if len(deps1) != len(deps2) {
		t.Errorf("deps count mismatch: first=%d, second=%d", len(deps1), len(deps2))
	}

	// ExtractImports should only be called once (on the miss)
	if detector.extractCalls != 1 {
		t.Errorf("ExtractImports called %d times, want 1 (second run should hit cache)", detector.extractCalls)
	}

	// Get should be called twice (once per run)
	if cache.getCalls != 2 {
		t.Errorf("Cache.Get called %d times, want 2", cache.getCalls)
	}
}

func TestRunDetectorsCached_NilCache(t *testing.T) {
	ctx := context.Background()

	projectDir := setupTempProject(t, map[string]string{
		"main.go": "package main\n",
	})

	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
		},
	}

	deps, err := RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, nil)
	if err != nil {
		t.Fatalf("RunDetectorsCached() with nil cache error = %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("RunDetectorsCached() returned %d deps, want 1", len(deps))
	}
}

func TestRunDetectorsCached_NoDetectors(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	_, err := RunDetectorsCached(ctx, "/test", []domain.Layer{}, []ports.Detector{}, cache)
	if err == nil {
		t.Errorf("RunDetectorsCached() with no detectors should return error")
	}
}

func TestRunDetectorsCached_InapplicableDetector(t *testing.T) {
	ctx := context.Background()
	cache := newMockCache()

	detector := &countingDetector{
		mockDetector: &mockDetector{
			name:         "go",
			detectResult: false, // not applicable
		},
	}

	deps, err := RunDetectorsCached(ctx, "/test", []domain.Layer{}, []ports.Detector{detector}, cache)
	if err != nil {
		t.Fatalf("RunDetectorsCached() error = %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("RunDetectorsCached() returned %d deps, want 0", len(deps))
	}

	// ExtractImports should NOT be called for inapplicable detector
	if detector.extractCalls != 0 {
		t.Errorf("ExtractImports called %d times for inapplicable detector, want 0", detector.extractCalls)
	}
}

func TestRunDetectorsCached_ConfigChangeInvalidates(t *testing.T) {
	ctx := context.Background()

	// Use real FileCache for this test
	cacheDir := filepath.Join(t.TempDir(), ".arx-cache")
	realCache := arxcache.NewFileCache(cacheDir)

	projectDir := setupTempProject(t, map[string]string{
		"main.go": "package main\n\nimport \"fmt\"\n",
	})

	detector := &countingDetector{
		mockDetector: &mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps: []domain.Dependency{
				{SourceFile: "main.go", SourceLine: 3, ImportPath: "fmt"},
			},
		},
	}

	// Set initial config hash
	if err := realCache.SetConfigHash("config-v1"); err != nil {
		t.Fatalf("SetConfigHash error = %v", err)
	}

	// First run with config-v1
	_, err := RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, realCache)
	if err != nil {
		t.Fatalf("First run error = %v", err)
	}
	firstCalls := detector.extractCalls

	// Second run with same config: should hit cache
	_, err = RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, realCache)
	if err != nil {
		t.Fatalf("Second run error = %v", err)
	}
	if detector.extractCalls != firstCalls {
		t.Errorf("ExtractImports called again on same config, expected cache hit")
	}

	// Change config hash
	if err := realCache.SetConfigHash("config-v2"); err != nil {
		t.Fatalf("SetConfigHash error = %v", err)
	}

	// Third run with different config: should miss
	_, err = RunDetectorsCached(ctx, projectDir, []domain.Layer{}, []ports.Detector{detector}, realCache)
	if err != nil {
		t.Fatalf("Third run error = %v", err)
	}
	if detector.extractCalls != firstCalls+1 {
		t.Errorf("ExtractImports not called after config change: calls=%d, want=%d", detector.extractCalls, firstCalls+1)
	}
}

func TestCheckService_DetectCached(t *testing.T) {
	cache := newMockCache()
	reader := &mockConfigReader{
		config: &domain.Config{
			Version: domain.SchemaVersion{Major: 1, Minor: 0},
			Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
			Rules:   []domain.Rule{},
		},
	}
	detectors := []ports.Detector{
		&mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
	}

	service := NewCheckServiceWithCache(reader, detectors, nil, cache)

	ctx := context.Background()
	deps, err := service.DetectCached(ctx, "/test", nil)
	if err != nil {
		t.Fatalf("DetectCached() error = %v", err)
	}

	// Empty deps is fine (no .go files in /test)
	if deps == nil {
		deps = []domain.Dependency{}
	}
}

func TestCheckService_Detect_BackwardCompat(t *testing.T) {
	reader := &mockConfigReader{
		config: &domain.Config{
			Version: domain.SchemaVersion{Major: 1, Minor: 0},
			Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
			Rules:   []domain.Rule{},
		},
	}
	detectors := []ports.Detector{
		&mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
	}

	// Create service without cache
	service := NewCheckService(reader, detectors, nil)

	ctx := context.Background()
	deps, err := service.Detect(ctx, "/test", nil)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Empty deps is fine (no .go files in /test)
	if deps == nil {
		deps = []domain.Dependency{}
	}
}
