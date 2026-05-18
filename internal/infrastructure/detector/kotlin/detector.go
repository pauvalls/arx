package kotlin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// KotlinDetector implements dependency extraction for Kotlin projects (Gradle Kotlin DSL, Maven)
type KotlinDetector struct {
	modulePrefix string
	sourceDirs   []string
}

// New creates a new Kotlin detector
func New() *KotlinDetector {
	return &KotlinDetector{
		modulePrefix: "",
		sourceDirs: []string{
			"src/main/kotlin",
			"src/test/kotlin",
		},
	}
}

// Name returns the detector name
func (d *KotlinDetector) Name() string {
	return "kotlin"
}

// Detect checks if this is a Kotlin project by looking for build files
func (d *KotlinDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for Gradle Kotlin DSL build.gradle.kts
	gradleKtsPath := filepath.Join(projectRoot, "build.gradle.kts")
	if _, err := os.Stat(gradleKtsPath); err == nil {
		return true, nil
	}

	// Check for Gradle settings.gradle.kts
	settingsKtsPath := filepath.Join(projectRoot, "settings.gradle.kts")
	if _, err := os.Stat(settingsKtsPath); err == nil {
		return true, nil
	}

	// Check for Maven pom.xml (may be mixed Java/Kotlin project)
	pomPath := filepath.Join(projectRoot, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		// Check for .kt files to confirm Kotlin usage
		hasKtFiles, err := d.hasKotlinFiles(projectRoot)
		if err != nil {
			return false, err
		}
		if hasKtFiles {
			return true, nil
		}
	}

	return false, nil
}

// hasKotlinFiles checks if there are any .kt files in the project (excluding build directories)
func (d *KotlinDetector) hasKotlinFiles(projectRoot string) (bool, error) {
	found := false
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common non-source directories
		if info.IsDir() && shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Check for .kt files (not in test dirs for this quick check)
		if strings.HasSuffix(path, ".kt") && !strings.HasSuffix(path, "Test.kt") && !strings.HasSuffix(path, "Tests.kt") {
			found = true
			return filepath.SkipAll // Stop walking once we find one
		}

		return nil
	})
	return found, err
}

// ExtractImports parses Kotlin files and extracts import statements
func (d *KotlinDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all Kotlin files
	kotlinFiles, err := d.FindKotlinFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Kotlin files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range kotlinFiles {
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

// shouldSkip returns true if the path should be skipped during dependency extraction
func shouldSkip(path string) bool {
	// Get the base name of the directory
	base := filepath.Base(path)

	// Skip common build and non-source directories
	skipDirs := map[string]bool{
		"target":       true, // Maven build output
		"build":        true, // Gradle build output
		".git":         true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
	}

	return skipDirs[base]
}

// shouldSkipPath checks if any component of the path should be skipped
func shouldSkipPath(path string) bool {
	// Check each component of the path
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if shouldSkip(part) {
			return true
		}
	}
	return false
}

// FindKotlinFiles finds all Kotlin files in the project, skipping build directories
func (d *KotlinDetector) FindKotlinFiles(projectRoot string) ([]string, error) {
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

		// Check if it's a Kotlin file
		if strings.HasSuffix(path, ".kt") {
			// Skip test files
			if strings.HasSuffix(path, "Test.kt") || strings.HasSuffix(path, "Tests.kt") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseFile extracts imports from a single Kotlin file
func (d *KotlinDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
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
					Language:      "kotlin",
				})
			}
		}
	}

	return dependencies, nil
}

// resolveImport resolves an import path to a layer
func (d *KotlinDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Skip external dependencies (Kotlin standard library, Java standard library, and third-party)
	if isExternalDependency(importPath) {
		return ""
	}

	// Convert import path to file path format (dots to slashes)
	importAsPath := strings.ReplaceAll(importPath, ".", "/")

	// Try to match import path to a layer
	for name, layer := range layerMap {
		// Check if layer matches the import path
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

// resolveSourcePath tries to resolve an import to a source file path
func (d *KotlinDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Convert package path to file path
	packagePath := strings.ReplaceAll(importPath, ".", "/")

	// Try common source locations
	locations := []string{
		filepath.Join(projectRoot, "src/main/kotlin", packagePath+".kt"),
		filepath.Join(projectRoot, "src/main/kotlin", packagePath, "*.kt"),
		filepath.Join(projectRoot, "src/test/kotlin", packagePath+".kt"),
		filepath.Join(projectRoot, packagePath+".kt"),
	}

	for _, loc := range locations {
		// For glob patterns, check if any file matches
		if strings.Contains(loc, "*") {
			matches, err := filepath.Glob(loc)
			if err == nil && len(matches) > 0 {
				return matches[0]
			}
		} else {
			if _, err := os.Stat(loc); err == nil {
				return loc
			}
		}
	}

	return ""
}
