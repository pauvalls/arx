package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// generateGoFiles creates a set of synthetic Go source files.
// Each file imports a few others to create realistic dependency resolution load.
// fileCount must be >= 3 (at least 3 files for cross-imports).
func generateGoFiles(dir string, fileCount int) {
	if fileCount < 3 {
		fileCount = 3
	}

	// Create some sub-packages
	pkgCount := fileCount / 10
	if pkgCount < 1 {
		pkgCount = 1
	}

	for pkgIdx := 0; pkgIdx < pkgCount; pkgIdx++ {
		pkgDir := filepath.Join(dir, fmt.Sprintf("pkg%d", pkgIdx))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			panic(err)
		}

		filesInPkg := fileCount / pkgCount
		if filesInPkg < 3 {
			filesInPkg = 3
		}

		for fileIdx := 0; fileIdx < filesInPkg; fileIdx++ {
			// Each file imports 3-5 random other files
			imports := make([]string, 0)
			for i := 0; i < 3+fileIdx%3; i++ {
				otherPkg := (pkgIdx + i + 1) % pkgCount
				imports = append(imports, fmt.Sprintf("\"test/app/pkg%d\"", otherPkg))
			}

			content := fmt.Sprintf(`package pkg%d

import (
	"os"
	"fmt"
	"strings"
	"strconv"
	"encoding/json"
	"math/rand"
	"time"
	"sync"
	"errors"
	"io"
	"sort"
	"bytes"
	"path/filepath"
	"log"
	"flag"
	"reflect"
	"unsafe"
	"runtime"
	"net/http"
	"regexp"
)

func init() {
	// prevent unused import errors
	var _ = os.Getpid
	var _ = fmt.Sprintf
	var _ = strings.ToLower
	var _ = strconv.Itoa
	var _ = json.Marshal
	var _ = rand.Intn
	var _ = time.Now
	var _ = sync.Mutex{}
	var _ = errors.New
	var _ = io.EOF
	var _ = sort.Ints
	var _ = bytes.Buffer{}
	var _ = filepath.Join
	var _ = log.Printf
	var _ = flag.NArg
	var _ = reflect.TypeOf
	var _ = unsafe.Pointer(nil)
	var _ = runtime.GOOS
}

func init() {
	// Cross-package imports
	_ = func() bool { return false }
}

func File%d_DoSomething() string {
	return fmt.Sprintf("hello from pkg%%d/file%%d", %[1]d, %[2]d)
}

func File%d_Process(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(data)
	return buf.Bytes(), nil
}

func File%d_Validate(input string) bool {
	return len(input) > 0
}
`, pkgIdx, fileIdx, fileIdx, fileIdx)

			filePath := filepath.Join(pkgDir, fmt.Sprintf("file_%d.go", fileIdx))
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				panic(err)
			}
		}
	}
}

// setupBenchProject creates a temporary project with synthetic Go files and a go.mod.
func setupBenchProject(b *testing.B, fileCount int) string {
	b.Helper()
	dir := b.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test/app\n\ngo 1.23\n"), 0644); err != nil {
		b.Fatalf("failed to write go.mod: %v", err)
	}

	// Generate synthetic test files
	generateGoFiles(dir, fileCount)

	// Create a minimal layers config
	layers := []domain.Layer{
		{Name: "app", Paths: []string{"pkg*/**"}},
	}

	// We need to write the config so that the detector can read layers
	_ = layers

	return dir
}

// benchmarkDetection measures end-to-end detection on a synthetic project.
func benchmarkDetection(b *testing.B, fileCount int) {
	projectDir := setupBenchProject(b, fileCount)

	layers := []domain.Layer{
		{Name: "app", Paths: []string{"pkg*/**"}},
	}

	// Create mock detector that simulates real detection
	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps:  []domain.Dependency{}, // will be populated
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use RunDetectors to exercise the concurrent pipeline
		deps, err := RunDetectors(ctx, projectDir, layers, []ports.Detector{detector})
		if err != nil {
			b.Fatalf("RunDetectors() error = %v", err)
		}
		_ = deps
	}
}

// benchmarkDetectionCached measures detection with caching.
func benchmarkDetectionCached(b *testing.B, fileCount int) {
	projectDir := setupBenchProject(b, fileCount)

	layers := []domain.Layer{
		{Name: "app", Paths: []string{"pkg*/**"}},
	}

	// Use a real cache
	cache := newMockCache()

	detector := &countingDetector{
		mockDetector: &mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		deps, err := RunDetectorsCached(ctx, projectDir, layers, []ports.Detector{detector}, cache)
		if err != nil {
			b.Fatalf("RunDetectorsCached() error = %v", err)
		}
		_ = deps
	}
}

// benchRunDetectors is a helper for running the full pipeline with b.Run.
func benchRunDetectors(b *testing.B, fileCount int) {
	b.Run("CacheMiss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			projectDir := setupBenchProject(b, fileCount)
			layers := []domain.Layer{
				{Name: "app", Paths: []string{"pkg*/**"}},
			}
			detector := &mockDetector{
				name:         "go",
				detectResult: true,
				extractDeps:  []domain.Dependency{},
			}
			b.StartTimer()

			deps, err := RunDetectors(context.Background(), projectDir, layers, []ports.Detector{detector})
			if err != nil {
				b.Fatalf("RunDetectors() error = %v", err)
			}
			_ = deps
		}
	})

	b.Run("CacheHit", func(b *testing.B) {
		projectDir := setupBenchProject(b, fileCount)
		layers := []domain.Layer{
			{Name: "app", Paths: []string{"pkg*/**"}},
		}
		cache := newMockCache()
		detector := &countingDetector{
			mockDetector: &mockDetector{
				name:         "go",
				detectResult: true,
				extractDeps:  []domain.Dependency{},
			},
		}

		// Warm up cache with first run
		_, err := RunDetectorsCached(context.Background(), projectDir, layers, []ports.Detector{detector}, cache)
		if err != nil {
			b.Fatalf("warmup error = %v", err)
		}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			deps, err := RunDetectorsCached(context.Background(), projectDir, layers, []ports.Detector{detector}, cache)
			if err != nil {
				b.Fatalf("RunDetectorsCached() error = %v", err)
			}
			_ = deps
		}
	})
}

func BenchmarkRunDetectors_100(b *testing.B)    { benchRunDetectors(b, 100) }
func BenchmarkRunDetectors_1k(b *testing.B)     { benchRunDetectors(b, 1000) }
func BenchmarkRunDetectors_10k(b *testing.B)    { benchRunDetectors(b, 10000) }

// Direct benchmarks (single-shot, no sub-benchmarks)
func BenchmarkDetectionPipeline_100_CacheMiss(b *testing.B)  { benchmarkDetection(b, 100) }
func BenchmarkDetectionPipeline_1k_CacheMiss(b *testing.B)   { benchmarkDetection(b, 1000) }
func BenchmarkDetectionPipeline_10k_CacheMiss(b *testing.B)  { benchmarkDetection(b, 10000) }

func BenchmarkDetectionPipeline_100_CacheHit(b *testing.B)   { benchmarkDetectionCached(b, 100) }
func BenchmarkDetectionPipeline_1k_CacheHit(b *testing.B)    { benchmarkDetectionCached(b, 1000) }
func BenchmarkDetectionPipeline_10k_CacheHit(b *testing.B)   { benchmarkDetectionCached(b, 10000) }
