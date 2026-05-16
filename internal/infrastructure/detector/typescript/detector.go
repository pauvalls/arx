package typescript_detector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// Detector implements the ports.Detector interface for TypeScript source files
type Detector struct {
	pathAliases map[string]string
	baseUrl     string
}

// New creates a new TypeScript detector
func New() *Detector {
	return &Detector{
		pathAliases: make(map[string]string),
	}
}

// Name returns the detector name
func (d *Detector) Name() string {
	return "typescript"
}

// Detect checks if this detector can handle the given project
// Returns true if tsconfig.json or package.json is present
func (d *Detector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for tsconfig.json
	tsconfigPath := filepath.Join(projectRoot, "tsconfig.json")
	if _, err := os.Stat(tsconfigPath); err == nil {
		// Load path aliases from tsconfig
		if err := d.loadTsConfig(tsconfigPath); err != nil {
			return false, fmt.Errorf("loading tsconfig: %w", err)
		}
		return true, nil
	}

	// Check for package.json as fallback
	packagePath := filepath.Join(projectRoot, "package.json")
	if _, err := os.Stat(packagePath); err == nil {
		return true, nil
	}

	return false, nil
}

// loadTsConfig reads tsconfig.json and extracts path aliases and baseUrl
func (d *Detector) loadTsConfig(tsconfigPath string) error {
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return err
	}

	var config struct {
		CompilerOptions struct {
			BaseUrl string            `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing tsconfig.json: %w", err)
	}

	d.baseUrl = config.CompilerOptions.BaseUrl

	// Convert TypeScript path patterns to simple prefixes
	for alias, paths := range config.CompilerOptions.Paths {
		if len(paths) > 0 {
			// Remove trailing * from alias (e.g., "@domain/*" -> "@domain")
			cleanAlias := strings.TrimSuffix(alias, "/*")
			// Remove trailing * from path (e.g., "src/domain/*" -> "src/domain")
			cleanPath := strings.TrimSuffix(paths[0], "/*")
			d.pathAliases[cleanAlias] = cleanPath
		}
	}

	return nil
}

// ExtractImports extracts all dependencies from TypeScript source files
// Uses regex-based extraction for speed (handles 95% of cases)
func (d *Detector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// TypeScript file extensions
	tsExtensions := []string{".ts", ".tsx", ".mts", ".cts"}

	// Load .arxignore patterns
	ignore, _ := domain.LoadArxIgnore(projectRoot)

	// Walk through all TypeScript files
	err := filepath.WalkDir(projectRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}

		if ignore != nil && ignore.IsIgnored(relPath) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if entry.IsDir() {
			// Skip node_modules, .git, and hidden directories
			name := entry.Name()
			if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a TypeScript file
		isTsFile := false
		for _, ext := range tsExtensions {
			if strings.HasSuffix(entry.Name(), ext) {
				isTsFile = true
				break
			}
		}
		if !isTsFile {
			return nil
		}

		// Skip test and spec files
		if strings.HasSuffix(entry.Name(), ".test.ts") ||
			strings.HasSuffix(entry.Name(), ".spec.ts") ||
			strings.HasSuffix(entry.Name(), ".test.tsx") ||
			strings.HasSuffix(entry.Name(), ".spec.tsx") {
			return nil
		}

		// Extract imports from file
		deps, err := d.extractFileImports(path, projectRoot, layers)
		if err != nil {
			// Log but continue - don't fail on single file parse errors
			return nil
		}

		dependencies = append(dependencies, deps...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking project directory: %w", err)
	}

	return dependencies, nil
}

// Regex patterns for TypeScript imports
var (
	// import ... from "..."
	importFromRegex = regexp.MustCompile(`import\s+(?:type\s+)?(?:[\w\s{},*]+\s+from\s+)?["']([^"']+)["']`)
	// require("...")
	requireRegex = regexp.MustCompile(`require\s*\(\s*["']([^"']+)["']\s*\)`)
	// export ... from "..."
	exportFromRegex = regexp.MustCompile(`export\s+(?:\*|\{[^}]*\}|\w+)\s+from\s+["']([^"']+)["']`)
	// import type { ... } from "..."
	importTypeRegex = regexp.MustCompile(`import\s+type\s+\{[^}]*\}\s+from\s+["']([^"']+)["']`)
)

// extractFileImports parses a single TypeScript file and extracts its imports
func (d *Detector) extractFileImports(filePath, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer file.Close()

	var dependencies []domain.Dependency
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Try each regex pattern
		importPaths := d.extractImportPaths(line)

		for _, importPath := range importPaths {
			// Resolve path aliases
			resolvedPath := d.resolveAlias(importPath)

			// Resolve to layer
			resolvedLayer := d.resolveLayer(resolvedPath, projectRoot, layers)

			// Get relative file path
			relPath, err := filepath.Rel(projectRoot, filePath)
			if err != nil {
				relPath = filePath
			}

			dependencies = append(dependencies, domain.Dependency{
				SourceFile:    relPath,
				SourceLine:    lineNum,
				ImportPath:    importPath,
				ResolvedLayer: resolvedLayer,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	return dependencies, nil
}

// extractImportPaths extracts all import paths from a line of TypeScript code
func (d *Detector) extractImportPaths(line string) []string {
	var paths []string

	// Check import ... from
	if matches := importFromRegex.FindStringSubmatch(line); matches != nil {
		paths = append(paths, matches[1])
	}

	// Check require()
	if matches := requireRegex.FindStringSubmatch(line); matches != nil {
		paths = append(paths, matches[1])
	}

	// Check export ... from
	if matches := exportFromRegex.FindStringSubmatch(line); matches != nil {
		paths = append(paths, matches[1])
	}

	// Check import type
	if matches := importTypeRegex.FindStringSubmatch(line); matches != nil {
		paths = append(paths, matches[1])
	}

	return paths
}

// resolveAlias converts TypeScript path aliases to actual paths
func (d *Detector) resolveAlias(importPath string) string {
	// Check for exact match
	if replacement, ok := d.pathAliases[importPath]; ok {
		return replacement
	}

	// Check for prefix match (e.g., @domain/users -> src/domain/users)
	for alias, replacement := range d.pathAliases {
		if strings.HasPrefix(importPath, alias+"/") {
			remaining := strings.TrimPrefix(importPath, alias+"/")
			return filepath.Join(replacement, remaining)
		}
	}

	return importPath
}

// resolveLayer determines which layer an import path belongs to
func (d *Detector) resolveLayer(importPath, projectRoot string, layers []domain.Layer) string {
	// Skip external dependencies (node_modules packages)
	if !strings.HasPrefix(importPath, ".") && !strings.HasPrefix(importPath, "/") {
		// Check if it's a scoped package
		if strings.HasPrefix(importPath, "@") {
			parts := strings.Split(importPath, "/")
			if len(parts) >= 2 {
				// Check if the alias matches a layer
				for _, layer := range layers {
					if layer.Name == parts[0][1:] || layer.Name == parts[1] {
						return layer.Name
					}
				}
			}
		}

		// For non-relative imports, check if any layer name matches
		for _, layer := range layers {
			if strings.HasPrefix(importPath, layer.Name) || strings.Contains(importPath, "/"+layer.Name+"/") {
				return layer.Name
			}
		}

		// External dependency (not part of any layer)
		return ""
	}

	// Relative import - resolve to absolute path
	baseDir := filepath.Dir(projectRoot)
	if d.baseUrl != "" {
		baseDir = filepath.Join(projectRoot, d.baseUrl)
	}

	var absPath string
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		absPath = filepath.Join(baseDir, importPath)
	} else {
		absPath = filepath.Join(baseDir, importPath)
	}

	// Get relative path from project root
	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		relPath = importPath
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Check which layer this path matches
	for _, layer := range layers {
		if layer.MatchesPath(relPath) {
			return layer.Name
		}
	}

	return ""
}

// Ensure Detector implements ports.Detector interface
var _ ports.Detector = (*Detector)(nil)
