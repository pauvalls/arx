package rust

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// RustDetector implements dependency extraction for Rust projects
type RustDetector struct {
	modulePrefix string
	sourceDirs   []string
}

// New creates a new Rust detector
func New() *RustDetector {
	return &RustDetector{
		modulePrefix: "",
		sourceDirs: []string{
			"src/",
		},
	}
}

// Name returns the detector name
func (d *RustDetector) Name() string {
	return "rust"
}

// Detect checks if this is a Rust project by looking for Cargo.toml
func (d *RustDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	cargoPath := filepath.Join(projectRoot, "Cargo.toml")
	if _, err := os.Stat(cargoPath); err == nil {
		return true, nil
	}
	return false, nil
}

// shouldSkip returns true if the path should be skipped during dependency extraction
func shouldSkip(path string) bool {
	base := filepath.Base(path)

	skipDirs := map[string]bool{
		"target":       true, // Cargo build output
		"build":        true, // Build script output
		".git":         true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
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

// FindRustFiles finds all Rust files in the project, skipping build directories and test files
func (d *RustDetector) FindRustFiles(projectRoot string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common non-source directories
		if info.IsDir() && shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Check if it's a Rust file
		if strings.HasSuffix(path, ".rs") {
			// Skip test files
			if strings.HasSuffix(path, "_test.rs") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ExtractImports parses Rust files and extracts use statements
func (d *RustDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all Rust files
	rustFiles, err := d.FindRustFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Rust files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range rustFiles {
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

// parseFile extracts imports from a single Rust file
func (d *RustDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
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
func (d *RustDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Strip leading crate::, self::, or super:: prefixes for layer matching
	resolved := importPath
	resolved = strings.TrimPrefix(resolved, "crate::")
	resolved = strings.TrimPrefix(resolved, "self::")
	for strings.HasPrefix(resolved, "super::") {
		resolved = strings.TrimPrefix(resolved, "super::")
	}

	// Skip external dependencies
	if isExternalDependency(importPath) {
		return ""
	}

	// Convert Rust path to file path format (:: to /)
	importAsPath := strings.ReplaceAll(resolved, "::", "/")

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
	sourcePath := d.resolveSourcePath(resolved, projectRoot)
	if sourcePath != "" {
		for name, layer := range layerMap {
			if layer.MatchesPath(sourcePath) {
				return name
			}
		}
	}

	return ""
}

// resolveSourcePath tries to resolve an import to a source file path
func (d *RustDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Convert Rust path to file path
	packagePath := strings.ReplaceAll(importPath, "::", "/")

	// Try common source locations
	locations := []string{
		filepath.Join(projectRoot, "src", packagePath+".rs"),
		filepath.Join(projectRoot, "src", packagePath, "mod.rs"),
		filepath.Join(projectRoot, packagePath+".rs"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}
