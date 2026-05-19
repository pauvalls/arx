package go_detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestName(t *testing.T) {
	d := New()
	if d.Name() != "go" {
		t.Errorf("Name() = %q, want %q", d.Name(), "go")
	}
}

func TestDetect(t *testing.T) {
	t.Run("with go.mod returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test/example\n\ngo 1.23\n"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ok, err := d.Detect(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if !ok {
			t.Error("Detect() = false, want true")
		}
		if d.modulePrefix != "test/example" {
			t.Errorf("modulePrefix = %q, want %q", d.modulePrefix, "test/example")
		}
	})

	t.Run("without go.mod returns false", func(t *testing.T) {
		d := New()
		ok, err := d.Detect(context.Background(), t.TempDir())
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if ok {
			t.Error("Detect() = true, want false")
		}
	})
}

func TestReadModulePrefix(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{"simple", "module github.com/user/project\n", "github.com/user/project", false},
		{"with version", "module github.com/user/project\n\ngo 1.23\n", "github.com/user/project", false},
		{"with comments", "// comment\nmodule github.com/user/project\n", "github.com/user/project", false},
		{"no module", "go 1.23\n", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "go.mod")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			d := New()
			got, err := d.readModulePrefix(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readModulePrefix() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("readModulePrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFileImports(t *testing.T) {
	t.Run("simple imports", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")
		content := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {}
`)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatal(err)
		}

		d := New()
		relPath := "main.go"
	deps, err := d.extractFileImports(filePath, relPath, tmpDir, nil)
		if err != nil {
			t.Fatalf("extractFileImports() error = %v", err)
		}
		if len(deps) != 2 {
			t.Fatalf("got %d deps, want 2", len(deps))
		}
		if deps[0].ImportPath != "fmt" || deps[1].ImportPath != "os" {
			t.Errorf("unexpected imports: %v", deps)
		}
	})

	t.Run("single import", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "main.go")
		content := []byte(`package main

import "fmt"

func main() {}
`)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatal(err)
		}

		d := New()
		deps, err := d.extractFileImports(filePath, "main.go", tmpDir, nil)
		if err != nil {
			t.Fatalf("extractFileImports() error = %v", err)
		}
		if len(deps) != 1 {
			t.Fatalf("got %d deps, want 1", len(deps))
		}
		if deps[0].ImportPath != "fmt" {
			t.Errorf("ImportPath = %q, want %q", deps[0].ImportPath, "fmt")
		}
	})

	t.Run("no imports", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty.go")
		if err := os.WriteFile(filePath, []byte("package empty\n"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		deps, err := d.extractFileImports(filePath, "empty.go", tmpDir, nil)
		if err != nil {
			t.Fatalf("extractFileImports() error = %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("got %d deps, want 0", len(deps))
		}
	})

	t.Run("malformed file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "bad.go")
		if err := os.WriteFile(filePath, []byte("this is not go code {{{"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		_, err := d.extractFileImports(filePath, "bad.go", tmpDir, nil)
		if err == nil {
			t.Error("expected error for malformed Go file")
		}
	})
}

func TestResolveLayer(t *testing.T) {
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
		{Name: "application", Paths: []string{"internal/application/**"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
	}

	t.Run("local module import", func(t *testing.T) {
		d := &Detector{modulePrefix: "github.com/myapp"}
		got := d.resolveLayer("github.com/myapp/internal/domain/user", "/project", layers)
		if got != "domain" {
			t.Errorf("resolveLayer() = %q, want %q", got, "domain")
		}
	})

	t.Run("external import", func(t *testing.T) {
		d := &Detector{modulePrefix: "github.com/myapp"}
		got := d.resolveLayer("github.com/somepkg/lib", "/project", layers)
		if got != "" {
			t.Errorf("resolveLayer() = %q, want empty string", got)
		}
	})

	t.Run("internal/ prefix", func(t *testing.T) {
		d := &Detector{}
		got := d.resolveLayer("internal/infrastructure/db", "/project", layers)
		if got != "infrastructure" {
			t.Errorf("resolveLayer() = %q, want %q", got, "infrastructure")
		}
	})
}

func TestExtractImports(t *testing.T) {
	t.Run("basic extraction", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test/app\n"), 0644); err != nil {
			t.Fatal(err)
		}
		domainDir := filepath.Join(tmpDir, "internal", "domain")
		if err := os.MkdirAll(domainDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(domainDir, "user.go"), []byte(`package domain

import "test/app/internal/infrastructure/db"
`), 0644); err != nil {
			t.Fatal(err)
		}

		d := New()
		ctx := context.Background()
		if _, err := d.Detect(ctx, tmpDir); err != nil {
			t.Fatal(err)
		}

		layers := []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
		}
		deps, err := d.ExtractImports(ctx, tmpDir, layers)
		if err != nil {
			t.Fatalf("ExtractImports() error = %v", err)
		}
		// Should have at least 1 dep (domain importing infra)
		if len(deps) == 0 {
			t.Fatal("ExtractImports() returned 0 deps")
		}
		// The dep should be from domain → infrastructure (if resolved)
		for _, dep := range deps {
			if dep.ImportPath == "test/app/internal/infrastructure/db" {
				if dep.ResolvedLayer != "infrastructure" {
					t.Errorf("resolved layer = %q, want %q", dep.ResolvedLayer, "infrastructure")
				}
				if dep.Language != "go" {
					t.Errorf("Language = %q, want %q", dep.Language, "go")
				}
				return
			}
		}
		t.Error("expected dep not found")
	})

	t.Run("skips test files", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main\nimport \"fmt\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ctx := context.Background()
		if _, err := d.Detect(ctx, tmpDir); err != nil {
			t.Fatal(err)
		}
		deps, err := d.ExtractImports(ctx, tmpDir, nil)
		if err != nil {
			t.Fatalf("ExtractImports() error = %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("expected 0 deps from test files, got %d", len(deps))
		}
	})

	t.Run("skips vendor directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatal(err)
		}
		vendorDir := filepath.Join(tmpDir, "vendor")
		if err := os.MkdirAll(vendorDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(vendorDir, "dep.go"), []byte("package dep\n"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ctx := context.Background()
		if _, err := d.Detect(ctx, tmpDir); err != nil {
			t.Fatal(err)
		}
		deps, err := d.ExtractImports(ctx, tmpDir, nil)
		if err != nil {
			t.Fatalf("ExtractImports() error = %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("expected 0 deps from vendor, got %d", len(deps))
		}
	})
}

// mockCache implements ports.Cache for testing the Go detector's caching behavior.
type mockCache struct {
	entries map[string][]domain.Dependency
}

func newMockCache() *mockCache {
	return &mockCache{entries: make(map[string][]domain.Dependency)}
}

func (m *mockCache) Get(fileHash string, detectorName string) ([]domain.Dependency, bool) {
	key := detectorName + ":" + fileHash
	deps, ok := m.entries[key]
	return deps, ok
}

func (m *mockCache) Put(fileHash string, detectorName string, deps []domain.Dependency) error {
	key := detectorName + ":" + fileHash
	m.entries[key] = deps
	return nil
}

func (m *mockCache) GetFile(key ports.FileCacheKey) ([]domain.Dependency, bool) {
	ck := key.DetectorName + ":file:" + key.RelativePath + ":" + key.ContentHash
	deps, ok := m.entries[ck]
	return deps, ok
}

func (m *mockCache) PutFile(key ports.FileCacheKey, deps []domain.Dependency) error {
	ck := key.DetectorName + ":file:" + key.RelativePath + ":" + key.ContentHash
	m.entries[ck] = deps
	return nil
}

func (m *mockCache) SetConfigHash(hash string) error { return nil }

func (m *mockCache) ConfigHash() (string, error) { return "", nil }

func (m *mockCache) Clear() error {
	m.entries = make(map[string][]domain.Dependency)
	return nil
}

func TestDetector_ExtractFileImports_FileCacheHit(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

import "fmt"

func main() {}
`)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cache := newMockCache()
	d := NewWithCache(cache)
	relPath := "main.go"

	// First call: cache miss, should parse and cache
	deps1, err := d.extractFileImports(filePath, relPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("first extractFileImports() error = %v", err)
	}
	if len(deps1) != 1 {
		t.Fatalf("first call got %d deps, want 1", len(deps1))
	}

	// Second call: cache hit, should return cached deps without parsing
	deps2, err := d.extractFileImports(filePath, relPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("second extractFileImports() error = %v", err)
	}
	if len(deps2) != 1 {
		t.Fatalf("second call got %d deps, want 1", len(deps2))
	}
	if deps2[0].ImportPath != "fmt" {
		t.Errorf("ImportPath = %q, want %q", deps2[0].ImportPath, "fmt")
	}
}

func TestDetector_ExtractFileImports_CacheMissOnContentChange(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	relPath := "main.go"

	// Write initial content
	content1 := []byte(`package main

import "fmt"

func main() {}
`)
	if err := os.WriteFile(filePath, content1, 0644); err != nil {
		t.Fatal(err)
	}

	cache := newMockCache()
	d := NewWithCache(cache)

	// First call: cache miss, parse and cache
	deps1, err := d.extractFileImports(filePath, relPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("first call error = %v", err)
	}
	if len(deps1) != 1 {
		t.Fatalf("first call got %d deps, want 1", len(deps1))
	}

	// Change content
	content2 := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {}
`)
	if err := os.WriteFile(filePath, content2, 0644); err != nil {
		t.Fatal(err)
	}

	// Second call: content changed → new sha256 → cache miss → re-parse
	deps2, err := d.extractFileImports(filePath, relPath, tmpDir, nil)
	if err != nil {
		t.Fatalf("second call error = %v", err)
	}
	if len(deps2) != 2 {
		t.Fatalf("second call got %d deps, want 2", len(deps2))
	}
}

func TestDetector_ExtractFileImports_CacheSkippedOnNilCache(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

import "fmt"

func main() {}
`)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// No cache set — should fall back to original behavior
	d := New()
	deps, err := d.extractFileImports(filePath, "main.go", tmpDir, nil)
	if err != nil {
		t.Fatalf("extractFileImports() error = %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("got %d deps, want 1", len(deps))
	}
}

func FuzzGoDetector(f *testing.F) {
	seeds := []string{
		"package main\nimport \"fmt\"\nfunc main() {}",
		"package test\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc Test(t testing.T) {}",
		"package main",
		"package main\nimport _ \"embed\"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content string) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.go")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return
		}
		d := New()
		// Should never panic, even with malformed Go code
		deps, err := d.extractFileImports(filePath, "test.go", tmpDir, nil)
		if err != nil {
			return // Parse errors expected
		}
		_ = deps
	})
}
