package java

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

var (
	importPattern        = regexp.MustCompile(`^import\s+([\w.*]+);`)
	importStaticPattern  = regexp.MustCompile(`^import\s+static\s+([\w.*]+);`)
	packagePattern       = regexp.MustCompile(`^package\s+([\w.]+);`)
)

// parseJavaImports extracts imports from Java source code
func parseJavaImports(content, filePath, projectRoot string, layers []domain.Layer, modulePrefix string) ([]domain.Dependency, error) {
	var deps []domain.Dependency

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse import statements
		var importPath string
		if matches := importPattern.FindStringSubmatch(line); len(matches) > 1 {
			importPath = matches[1]
		} else if matches := importStaticPattern.FindStringSubmatch(line); len(matches) > 1 {
			importPath = matches[1]
		}

		if importPath == "" {
			continue
		}

		// Skip external imports (java.*, javax.*, etc.)
		if isExternalImport(importPath) {
			continue
		}

		// Resolve layer for target import
		targetLayer := resolveImportLayer(importPath, layers, modulePrefix)

		dep := domain.Dependency{
			SourceFile:    filePath,
			SourceLine:    lineNum,
			ImportPath:    importPath,
			ResolvedLayer: targetLayer,
		}

		deps = append(deps, dep)
	}

	return deps, nil
}

// isExternalImport checks if an import is from external libraries
func isExternalImport(importPath string) bool {
	externalPrefixes := []string{
		"java.",
		"javax.",
		"sun.",
		"com.sun.",
		"oracle.",
	}

	for _, prefix := range externalPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}

	return false
}

// resolveImportLayer determines which layer an import belongs to
func resolveImportLayer(importPath string, layers []domain.Layer, modulePrefix string) string {
	// Normalize import path
	normalizedImport := importPath
	if modulePrefix != "" && strings.HasPrefix(importPath, modulePrefix) {
		normalizedImport = strings.TrimPrefix(importPath, modulePrefix+".")
	}

	// Check each layer's path patterns
	for _, layer := range layers {
		for _, pattern := range layer.Paths {
			if matchLayerPattern(pattern, normalizedImport) {
				return layer.Name
			}
		}
	}

	return ""
}

// matchLayerPattern checks if an import matches a layer pattern
func matchLayerPattern(pattern, importPath string) bool {
	// Convert glob pattern to package pattern
	// e.g., "com/example/domain/**" -> matches "com.example.domain.Order"
	pattern = strings.ReplaceAll(pattern, "**", "*")
	pattern = strings.ReplaceAll(pattern, "/", ".")
	
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(importPath, prefix+".")
	}
	
	return importPath == pattern
}
