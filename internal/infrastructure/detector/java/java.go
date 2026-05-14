package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// Detector detects Java dependencies
type Detector struct {
	modulePrefix string
	sourceDirs   []string
}

// New creates a new Java detector
func New() *Detector {
	return &Detector{
		sourceDirs: []string{"src/main/java", "src/test/java"},
	}
}

// Name returns the detector name
func (d *Detector) Name() string {
	return "java"
}

// Detect checks if this detector can handle the given project
func (d *Detector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for Java files
	for _, srcDir := range d.sourceDirs {
		fullPath := filepath.Join(projectRoot, srcDir)
		if _, err := os.Stat(fullPath); err == nil {
			return true, nil
		}
	}

	// Check for pom.xml or build.gradle
	pomPath := filepath.Join(projectRoot, "pom.xml")
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	
	if _, err := os.Stat(pomPath); err == nil {
		return true, nil
	}
	if _, err := os.Stat(gradlePath); err == nil {
		return true, nil
	}

	return false, nil
}

// ExtractImports extracts all dependencies from the project
func (d *Detector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var deps []domain.Dependency

	// Try to detect module prefix from pom.xml or build.gradle
	d.modulePrefix = d.detectModulePrefix(projectRoot)

	// Find all Java files
	javaFiles, err := d.findJavaFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Java files: %w", err)
	}

	// Extract imports from each file
	for _, file := range javaFiles {
		fileDeps, err := d.extractFileImports(file, projectRoot, layers)
		if err != nil {
			continue // Skip files with errors
		}
		deps = append(deps, fileDeps...)
	}

	return deps, nil
}

// detectModulePrefix tries to detect the module prefix from Maven or Gradle
func (d *Detector) detectModulePrefix(projectRoot string) string {
	// Try Maven first
	pomPath := filepath.Join(projectRoot, "pom.xml")
	if prefix, err := parseMavenPom(pomPath); err == nil && prefix != "" {
		return prefix
	}

	// Try Gradle
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	if prefix, err := parseGradleFile(gradlePath); err == nil && prefix != "" {
		return prefix
	}

	return ""
}

// findJavaFiles finds all Java files in the project
func (d *Detector) findJavaFiles(projectRoot string) ([]string, error) {
	var files []string

	for _, srcDir := range d.sourceDirs {
		fullPath := filepath.Join(projectRoot, srcDir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".java" {
				return nil
			}
			if d.shouldSkipPath(path) {
				return nil
			}
			files = append(files, path)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// shouldSkipPath checks if a path should be skipped
func (d *Detector) shouldSkipPath(path string) bool {
	skipDirs := []string{"target", "build", ".git", "node_modules", ".idea", ".vscode"}
	
	for _, skipDir := range skipDirs {
		if strings.Contains(path, "/"+skipDir+"/") || strings.HasSuffix(path, "/"+skipDir) {
			return true
		}
	}
	return false
}

// extractFileImports extracts imports from a single Java file
func (d *Detector) extractFileImports(filePath, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return parseJavaImports(string(content), filePath, projectRoot, layers, d.modulePrefix)
}
