package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// RunDetectorsCachedWithStatus executes all applicable detectors with caching and returns per-detector status.
// If cache is nil, falls back to RunDetectorsWithStatus (backward compatible).
func RunDetectorsCachedWithStatus(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector, cache ports.Cache) (*DetectorResult, error) {
	if len(detectors) == 0 {
		return nil, fmt.Errorf("no detectors provided")
	}

	// If no cache, fall back to original behavior
	if cache == nil {
		return RunDetectorsWithStatus(ctx, projectRoot, layers, detectors)
	}

	var allDependencies []domain.Dependency
	statuses := make([]DetectorStatus, len(detectors))

	for i, detector := range detectors {
		if detector == nil {
			continue
		}

		idx := i
		status := DetectorStatus{Name: detector.Name()}

		// Check if this detector is applicable
		applicable, err := detector.Detect(ctx, projectRoot)
		if err != nil {
			status.Error = err.Error()
			statuses[idx] = status
			return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, fmt.Errorf("detector %q detection failed: %w", detector.Name(), err)
		}

		status.Applicable = applicable

		if !applicable {
			statuses[idx] = status
			continue
		}

		// Compute combined hash of all source files for this detector
		projectHash, err := hashProjectFiles(projectRoot, detector.Name())
		if err != nil {
			status.Error = err.Error()
			statuses[idx] = status
			return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, fmt.Errorf("detector %q file hashing failed: %w", detector.Name(), err)
		}

		// Check cache
		if cached, ok := cache.Get(projectHash, detector.Name()); ok {
			status.DepCount = len(cached)
			statuses[idx] = status
			allDependencies = append(allDependencies, cached...)
			continue
		}

		// Cache miss: call detector
		deps, err := detector.ExtractImports(ctx, projectRoot, layers)
		if err != nil {
			status.Error = err.Error()
			statuses[idx] = status
			return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, fmt.Errorf("detector %q extraction failed: %w", detector.Name(), err)
		}

		// Store in cache
		if err := cache.Put(projectHash, detector.Name(), deps); err != nil {
			// Log but don't fail on cache write errors
		}

		status.DepCount = len(deps)
		statuses[idx] = status
		allDependencies = append(allDependencies, deps...)
	}

	return &DetectorResult{Dependencies: allDependencies, Statuses: statuses}, nil
}

// RunDetectorsCached executes all applicable detectors with caching.
// For each detector, it computes a combined hash of all relevant source files,
// checks cache, and on miss calls ExtractImports and stores the result.
// If cache is nil, behaves identically to RunDetectors (backward compatible).
func RunDetectorsCached(ctx context.Context, projectRoot string, layers []domain.Layer, detectors []ports.Detector, cache ports.Cache) ([]domain.Dependency, error) {
	if len(detectors) == 0 {
		return nil, fmt.Errorf("no detectors provided")
	}

	// If no cache, fall back to original behavior
	if cache == nil {
		return RunDetectors(ctx, projectRoot, layers, detectors)
	}

	var allDependencies []domain.Dependency

	for _, detector := range detectors {
		if detector == nil {
			continue
		}

		// Check if this detector is applicable
		applicable, err := detector.Detect(ctx, projectRoot)
		if err != nil {
			return nil, fmt.Errorf("detector %q detection failed: %w", detector.Name(), err)
		}

		if !applicable {
			continue
		}

		// Compute combined hash of all source files for this detector
		projectHash, err := hashProjectFiles(projectRoot, detector.Name())
		if err != nil {
			return nil, fmt.Errorf("detector %q file hashing failed: %w", detector.Name(), err)
		}

		// Check cache
		if cached, ok := cache.Get(projectHash, detector.Name()); ok {
			allDependencies = append(allDependencies, cached...)
			continue
		}

		// Cache miss: call detector
		deps, err := detector.ExtractImports(ctx, projectRoot, layers)
		if err != nil {
			return nil, fmt.Errorf("detector %q extraction failed: %w", detector.Name(), err)
		}

		// Store in cache
		if err := cache.Put(projectHash, detector.Name(), deps); err != nil {
			// Log but don't fail on cache write errors
		}

		allDependencies = append(allDependencies, deps...)
	}

	return allDependencies, nil
}

// hashProjectFiles computes a combined SHA-256 hash of all relevant source files.
// The hash is deterministic: same files in any order produce the same hash.
func hashProjectFiles(projectRoot string, detectorName string) (string, error) {
	var fileHashes []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".arx-cache" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only hash files relevant to this detector
		if !isSourceFile(path, detectorName) {
			return nil
		}

		h, err := hashFile(path)
		if err != nil {
			return nil
		}
		fileHashes = append(fileHashes, h)
		return nil
	})

	if err != nil {
		return "", err
	}

	// Sort for deterministic ordering
	sort.Strings(fileHashes)

	// Combine all file hashes into a single project hash
	combined := strings.Join(fileHashes, "")
	h := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(h[:]), nil
}

// isSourceFile checks if a file is relevant for the given detector.
func isSourceFile(path string, detectorName string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch detectorName {
	case "go":
		return ext == ".go"
	case "typescript":
		return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
	case "python":
		return ext == ".py"
	case "java":
		return ext == ".java"
	default:
		// For unknown detectors, hash all non-binary files
		return ext != "" && ext != ".exe" && ext != ".dll" && ext != ".so"
	}
}

// hashFile computes SHA-256 of a file's content.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
