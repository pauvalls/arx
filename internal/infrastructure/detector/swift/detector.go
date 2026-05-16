package swift

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// SwiftDetector implements dependency extraction for Swift projects
type SwiftDetector struct {
	sourceDirs []string
}

// New creates a new Swift detector
func New() *SwiftDetector {
	return &SwiftDetector{
		sourceDirs: []string{
			"Sources/",
		},
	}
}

// Name returns the detector name
func (d *SwiftDetector) Name() string {
	return "swift"
}

// Detect checks if this is a Swift project by looking for Package.swift
func (d *SwiftDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	packagePath := filepath.Join(projectRoot, "Package.swift")
	if _, err := os.Stat(packagePath); err == nil {
		return true, nil
	}
	return false, nil
}

// shouldSkip returns true if the path should be skipped during dependency extraction
func shouldSkip(path string) bool {
	base := filepath.Base(path)

	skipDirs := map[string]bool{
		".build":       true, // SPM build artifacts
		".git":         true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
		"DerivedData":  true, // Xcode build artifacts
		"Tests":        true, // SPM test targets
	}

	return skipDirs[base]
}

// shouldSkipPath checks if any component of the path should be skipped
func shouldSkipPath(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if shouldSkip(part) {
			return true
		}
	}
	return false
}

// FindSwiftFiles finds all Swift files in the project, skipping Tests/ and build directories
func (d *SwiftDetector) FindSwiftFiles(projectRoot string) ([]string, error) {
	var files []string

	ignore, _ := domain.LoadArxIgnore(projectRoot)

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}

		if ignore != nil && ignore.IsIgnored(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common non-source directories
		if info.IsDir() && shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Check if it's a Swift file
		if strings.HasSuffix(path, ".swift") {
			// Skip test files
			if strings.HasSuffix(path, "Tests.swift") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ExtractImports parses Swift files and extracts import statements
func (d *SwiftDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all Swift files
	swiftFiles, err := d.FindSwiftFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Swift files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range swiftFiles {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		deps, err := d.parseFile(file, projectRoot, layerMap)
		if err != nil {
			// Log but continue on parse errors
			continue
		}

		dependencies = append(dependencies, deps...)
	}

	return dependencies, nil
}

// parseFile extracts imports from a single Swift file
func (d *SwiftDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var dependencies []domain.Dependency
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		lineNum++ // 1-indexed line numbers

		// Extract imports using parser functions
		imports := extractImportsFromLine(line)

		for _, importPath := range imports {
			// Resolve the import to a layer
			resolvedLayer := d.resolveImport(importPath, filePath, projectRoot, layerMap)

			if resolvedLayer != "" {
				dependencies = append(dependencies, domain.Dependency{
					SourceFile:    filePath,
					SourceLine:    lineNum,
					ImportPath:    importPath,
					ResolvedLayer: resolvedLayer,
				})
			}
		}
	}

	return dependencies, nil
}

// resolveImport resolves an import path to a layer
func (d *SwiftDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Skip external dependencies (system frameworks)
	if isExternalDependency(importPath) {
		return ""
	}

	// Swift module names map to layer names (case-insensitive)
	// e.g., "Domain" import → "domain" layer
	for name := range layerMap {
		if strings.EqualFold(importPath, name) {
			return name
		}
	}

	// Try to match import path to a layer using path patterns
	for name, layer := range layerMap {
		if layer.MatchesPath(importPath) {
			return name
		}

		// Also try matching against layer paths directly using our custom matcher
		for _, layerPath := range layer.Paths {
			if shared.MatchImportToLayer(importPath, layerPath) {
				return name
			}
		}
	}

	// Try to resolve to a local source file
	sourcePath := d.resolveSourcePath(importPath, projectRoot)
	if sourcePath != "" {
		relPath, err := filepath.Rel(projectRoot, sourcePath)
		if err == nil {
			// Strip source directory prefix (e.g., "Sources/") for layer matching
			for _, srcDir := range d.sourceDirs {
				if srcDir != "" {
					relPath = strings.TrimPrefix(relPath, srcDir)
					relPath = strings.TrimPrefix(relPath, "/")
				}
			}
			for name, layer := range layerMap {
				if layer.MatchesPath(relPath) {
					return name
				}
			}
		}
	}

	return ""
}

// resolveSourcePath tries to resolve an import to a source file path
func (d *SwiftDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Try common source locations under Sources/
	locations := []string{
		filepath.Join(projectRoot, "Sources", importPath+".swift"),
		filepath.Join(projectRoot, "Sources", importPath, filepath.Base(importPath)+".swift"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}
