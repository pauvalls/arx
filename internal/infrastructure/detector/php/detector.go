package php

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// PHPDetector implements dependency extraction for PHP projects
type PHPDetector struct {
	sourceDirs []string
}

// New creates a new PHP detector
func New() *PHPDetector {
	return &PHPDetector{
		sourceDirs: []string{
			"",    // project root
			"src/",
		},
	}
}

// Name returns the detector name
func (d *PHPDetector) Name() string {
	return "php"
}

// Detect checks if this is a PHP project by looking for composer.json
func (d *PHPDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	composerPath := filepath.Join(projectRoot, "composer.json")
	if _, err := os.Stat(composerPath); err == nil {
		return true, nil
	}
	return false, nil
}

// shouldSkip returns true if the path should be skipped during dependency extraction
func shouldSkip(path string) bool {
	base := filepath.Base(path)

	skipDirs := map[string]bool{
		"vendor":       true, // Composer vendor packages
		".git":         true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
		"tests":        true,
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

// FindPHPFiles finds all PHP files in the project, skipping vendor/ and test directories
func (d *PHPDetector) FindPHPFiles(projectRoot string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common non-source directories
		if info.IsDir() && shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Check if it's a PHP file
		if strings.HasSuffix(path, ".php") {
			// Skip test files
			if strings.HasSuffix(path, "Test.php") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ExtractImports parses PHP files and extracts use/require_once statements
func (d *PHPDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all PHP files
	phpFiles, err := d.FindPHPFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find PHP files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range phpFiles {
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

// parseFile extracts imports from a single PHP file
func (d *PHPDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
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
func (d *PHPDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Skip external dependencies (vendor packages)
	if isExternalDependency(importPath) {
		return ""
	}

	// For relative paths (require_once), resolve them from the source file's directory first
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		sourceDir := filepath.Dir(filePath)
		resolvedPath := filepath.Clean(filepath.Join(sourceDir, importPath))
		relPath, err := filepath.Rel(projectRoot, resolvedPath)
		if err == nil {
			// Strip source directory prefix (e.g., "src/") for layer matching
			for _, srcDir := range d.sourceDirs {
				if srcDir != "" {
					relPath = strings.TrimPrefix(relPath, srcDir)
					relPath = strings.TrimPrefix(relPath, "/")
				}
			}
			// Try matching the resolved relative path against layers
			for name, layer := range layerMap {
				if layer.MatchesPath(relPath) {
					return name
				}
				for _, layerPath := range layer.Paths {
					if shared.MatchImportToLayer(relPath, layerPath) {
						return name
					}
				}
			}
		}
	}

	// Convert namespace path to file path format (Namespace\Class -> Namespace/Class)
	importAsPath := strings.ReplaceAll(importPath, "\\", "/")

	// Try to match import path to a layer
	for name, layer := range layerMap {
		if layer.MatchesPath(importAsPath) {
			return name
		}

		// Also try matching against layer paths directly using our custom matcher
		for _, layerPath := range layer.Paths {
			if shared.MatchImportToLayer(importAsPath, layerPath) {
				return name
			}
		}
	}

	// Try to resolve to a local source file
	sourcePath := d.resolveSourcePath(importPath, projectRoot)
	if sourcePath != "" {
		for name, layer := range layerMap {
			if layer.MatchesPath(sourcePath) {
				return name
			}
		}
	}

	return ""
}

// resolveSourcePath tries to resolve a namespace import to a source file path
func (d *PHPDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Convert namespace to file path (Namespace\Class -> Namespace/Class.php)
	packagePath := strings.ReplaceAll(importPath, "\\", "/")

	// Try common source locations
	locations := []string{
		filepath.Join(projectRoot, "src", packagePath+".php"),
		filepath.Join(projectRoot, "src", packagePath, filepath.Base(packagePath)+".php"),
		filepath.Join(projectRoot, packagePath+".php"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}
