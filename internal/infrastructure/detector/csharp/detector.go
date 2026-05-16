package csharp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// CSharpDetector implements dependency extraction for C# projects
type CSharpDetector struct {
	modulePrefix string
	sourceDirs   []string
}

// New creates a new C# detector
func New() *CSharpDetector {
	return &CSharpDetector{
		modulePrefix: "",
		sourceDirs: []string{
			"", // project root (C# convention)
		},
	}
}

// Name returns the detector name
func (d *CSharpDetector) Name() string {
	return "csharp"
}

// Detect checks if this is a C# project by looking for .csproj or .sln files
func (d *CSharpDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for .csproj files at project root
	csprojPattern := filepath.Join(projectRoot, "*.csproj")
	matches, err := filepath.Glob(csprojPattern)
	if err != nil {
		return false, fmt.Errorf("failed to glob for .csproj files: %w", err)
	}
	if len(matches) > 0 {
		return true, nil
	}

	// Check for .sln files at project root
	slnPattern := filepath.Join(projectRoot, "*.sln")
	matches, err = filepath.Glob(slnPattern)
	if err != nil {
		return false, fmt.Errorf("failed to glob for .sln files: %w", err)
	}
	if len(matches) > 0 {
		return true, nil
	}

	return false, nil
}

// shouldSkip returns true if the path should be skipped during dependency extraction
func shouldSkip(path string) bool {
	base := filepath.Base(path)

	skipDirs := map[string]bool{
		"bin":          true, // Build output
		"obj":          true, // Build intermediates
		".vs":          true, // Visual Studio
		".vscode":      true,
		".git":         true,
		"node_modules": true,
		".idea":        true,
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

// FindCSharpFiles finds all C# files in the project, skipping build directories and test files
func (d *CSharpDetector) FindCSharpFiles(projectRoot string) ([]string, error) {
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

		// Check if it's a C# file
		if strings.HasSuffix(path, ".cs") {
			// Skip test files
			baseName := filepath.Base(path)
			if strings.HasSuffix(baseName, "Test.cs") || strings.HasSuffix(baseName, "Tests.cs") {
				return nil
			}
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ExtractImports parses C# files and extracts using directives
func (d *CSharpDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all C# files
	csharpFiles, err := d.FindCSharpFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find C# files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range csharpFiles {
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

// parseFile extracts imports from a single C# file
func (d *CSharpDetector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
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
func (d *CSharpDetector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Skip external dependencies
	if isExternalDependency(importPath) {
		return ""
	}

	// Convert namespace path to file path format (. to /)
	importAsPath := strings.ReplaceAll(importPath, ".", "/")

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
			
			// Try matching just the suffix of the import path
			// This handles cases like "MyApp.Domain.Entities" matching "Domain/**"
			if matchImportSuffix(importAsPath, layerPath) {
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

// matchImportSuffix checks if the import path ends with a pattern that matches the layer
func matchImportSuffix(importPath, layerPattern string) bool {
	// Convert layer pattern to regex
	escaped := regexp.QuoteMeta(layerPattern)
	escaped = strings.ReplaceAll(escaped, `/\*\*`, "(/.*)?")
	escaped = strings.ReplaceAll(escaped, `\*\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\*`, "[^/]*")
	pattern := "(^|/)" + escaped + "$"
	
	matched, err := regexp.MatchString(pattern, importPath)
	if err != nil {
		return false
	}
	
	return matched
}

// resolveSourcePath tries to resolve an import to a source file path
func (d *CSharpDetector) resolveSourcePath(importPath, projectRoot string) string {
	// Convert namespace path to file path
	packagePath := strings.ReplaceAll(importPath, ".", "/")

	// Try common source locations
	locations := []string{
		filepath.Join(projectRoot, packagePath+".cs"),
		filepath.Join(projectRoot, packagePath, "index.cs"),
		filepath.Join(projectRoot, "src", packagePath+".cs"),
		filepath.Join(projectRoot, "src", packagePath, "index.cs"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}
