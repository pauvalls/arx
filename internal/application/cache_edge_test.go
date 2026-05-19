package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestIsSourceFile_KnownDetectors(t *testing.T) {
	tests := []struct {
		path     string
		detector string
		want     bool
	}{
		{"main.go", "go", true},
		{"main.ts", "typescript", true},
		{"main.tsx", "typescript", true},
		{"main.js", "typescript", true},
		{"main.py", "python", true},
		{"Main.java", "java", true},
		{"test.txt", "go", false},
		{"main.go", "unknown", true},
		{"file.exe", "unknown", false},
		{"file", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"/"+tt.detector, func(t *testing.T) {
			got := isSourceFile(tt.path, tt.detector)
			if got != tt.want {
				t.Errorf("isSourceFile(%q, %q) = %v, want %v", tt.path, tt.detector, got, tt.want)
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashFile(filePath)
	if err != nil {
		t.Fatalf("hashFile() error = %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Same content should produce same hash
	hash2, err := hashFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if hash != hash2 {
		t.Error("expected same hash for same content")
	}
}

func TestHashFile_Nonexistent(t *testing.T) {
	_, err := hashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestHashProjectFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	hash, err := hashProjectFiles(tmpDir, "go")
	if err != nil {
		t.Fatalf("hashProjectFiles() error = %v", err)
	}
	if hash == "" {
		// Empty project might produce an empty hash — acceptable
		t.Logf("hash for empty dir: %s", hash)
	}
}

func TestHashProjectFiles_SkipDirs(t *testing.T) {
	tmpDir := t.TempDir()
	// Create files in skipped directories
	for _, dir := range []string{".git", "node_modules", "vendor", ".arx-cache"} {
		d := filepath.Join(tmpDir, dir, "sub")
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "file.go"), []byte("package x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a real Go file in a non-skipped dir
	realDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "entity.go"), []byte("package domain"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashProjectFiles(tmpDir, "go")
	if err != nil {
		t.Fatalf("hashProjectFiles() error = %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for dir with go files")
	}
}

func TestCacheSaveLoad_HappyPath(t *testing.T) {
	cache := newMockCache()
	deps := []domain.Dependency{
		{SourceFile: "a.go", ImportPath: "fmt"},
	}

	if err := cache.Put("hash1", "go", deps); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	loaded, ok := cache.Get("hash1", "go")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 dep, got %d", len(loaded))
	}
}

func TestCacheSaveLoad_Miss(t *testing.T) {
	cache := newMockCache()

	_, ok := cache.Get("unknown", "go")
	if ok {
		t.Error("expected cache miss for unknown key")
	}
}

func TestCacheSaveLoad_DifferentDetector(t *testing.T) {
	cache := newMockCache()
	deps := []domain.Dependency{{SourceFile: "a.go", ImportPath: "fmt"}}

	if err := cache.Put("hash1", "go", deps); err != nil {
		t.Fatal(err)
	}

	// Same hash, different detector should miss
	_, ok := cache.Get("hash1", "typescript")
	if ok {
		t.Error("expected cache miss for different detector")
	}
}

func TestCacheSaveLoad_EmptyData(t *testing.T) {
	cache := newMockCache()
	if err := cache.Put("hash1", "go", []domain.Dependency{}); err != nil {
		t.Fatal(err)
	}

	loaded, ok := cache.Get("hash1", "go")
	if !ok {
		t.Fatal("expected cache hit for empty deps")
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 deps, got %d", len(loaded))
	}
}
