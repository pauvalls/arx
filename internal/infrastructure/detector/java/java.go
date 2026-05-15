package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// JavaDetector implements dependency extraction for Java projects (Maven/Gradle)
type JavaDetector struct {
	modulePrefix string
	sourceDirs   []string
}

// New creates a new Java detector
func New() *JavaDetector {
	return &JavaDetector{
		modulePrefix: "",
		sourceDirs: []string{
			"src/main/java",
			"src/test/java",
		},
	}
}

// Name returns the detector name
func (d *JavaDetector) Name() string {
	return "java"
}

// Detect checks if this is a Java project by looking for build files
func (d *JavaDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for Maven pom.xml
	pomPath := filepath.Join(projectRoot, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		// Try to extract module prefix from pom.xml
		d.extractModulePrefix(pomPath)
		return true, nil
	}

	// Check for Gradle build.gradle
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	if _, err := os.Stat(gradlePath); err == nil {
		// Try to extract module prefix from build.gradle
		d.modulePrefix = extractModulePrefixFromGradle(gradlePath)
		return true, nil
	}

	// Check for Gradle Kotlin build.gradle.kts
	gradleKtsPath := filepath.Join(projectRoot, "build.gradle.kts")
	if _, err := os.Stat(gradleKtsPath); err == nil {
		// Try to extract module prefix from build.gradle.kts
		d.modulePrefix = extractModulePrefixFromGradle(gradleKtsPath)
		return true, nil
	}

	return false, nil
}

// ExtractImports parses Java files and extracts import statements
func (d *JavaDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all Java files
	javaFiles, err := d.FindJavaFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Java files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range javaFiles {
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

// FindJavaFiles finds all Java files in the project, skipping build directories
func (d *JavaDetector) FindJavaFiles(projectRoot string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common non-source directories
		if info.IsDir() && shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Check if it's a Java file
		if strings.HasSuffix(path, ".java") {
			// Skip test files
			if strings.HasSuffix(path, "Test.java") || strings.HasSuffix(path, "Tests.java") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseFile extracts imports from a single Java file
func (d *JavaDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
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
func (d *JavaDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Skip external dependencies (Java standard library and third-party)
	// Standard Java packages: java.*, javax.*, sun.*, com.sun.*
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
			if importMatchesLayer(importAsPath, layerPath) {
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
func (d *JavaDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Convert package path to file path
	packagePath := strings.ReplaceAll(importPath, ".", "/")

	// Try common source locations
	locations := []string{
		filepath.Join(projectRoot, "src/main/java", packagePath+".java"),
		filepath.Join(projectRoot, "src/main/java", packagePath, "*.java"),
		filepath.Join(projectRoot, "src/test/java", packagePath+".java"),
		filepath.Join(projectRoot, packagePath+".java"),
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

// extractModulePrefix reads pom.xml and extracts the module prefix using XML parsing
func (d *JavaDetector) extractModulePrefix(pomPath string) {
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)
	if err != nil {
		// Log error but don't fail - modulePrefix will remain empty
		return
	}
	d.modulePrefix = result.ModulePrefix
}

// isExternalDependency checks if an import is from external libraries
func isExternalDependency(importPath string) bool {
	// Java standard library packages
	standardPackages := []string{
		"java.",
		"javax.",
		"sun.",
		"com.sun.",
		"org.ietf.",
		"org.w3c.",
		"org.xml.",
	}

	for _, pkg := range standardPackages {
		if strings.HasPrefix(importPath, pkg) {
			return true
		}
	}

	return false
}
