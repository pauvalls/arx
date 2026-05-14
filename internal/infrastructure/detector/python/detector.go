package python

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// Detector implements dependency extraction for Python projects
type Detector struct {
	importPatterns []*regexp.Regexp
}

// New creates a new Python detector
func New() *Detector {
	return &Detector{
		importPatterns: []*regexp.Regexp{
			// import module
			regexp.MustCompile(`^import\s+([\w.]+)`),
			// from module import ...
			regexp.MustCompile(`^from\s+([\w.]+)\s+import`),
			// from .module import ... (relative)
			regexp.MustCompile(`^from\s+(\.[\w.]+)\s+import`),
		},
	}
}

// Name returns the detector name
func (d *Detector) Name() string {
	return "python"
}

// Detect checks if this is a Python project
func (d *Detector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	// Check for common Python project markers
	markers := []string{
		"requirements.txt",
		"setup.py",
		"pyproject.toml",
		"Pipfile",
		"poetry.lock",
	}

	for _, marker := range markers {
		path := filepath.Join(projectRoot, marker)
		if _, err := os.Stat(path); err == nil {
			return true, nil
		}
	}

	// Also check for .py files
	pyFiles, err := filepath.Glob(filepath.Join(projectRoot, "**/*.py"))
	if err == nil && len(pyFiles) > 0 {
		return true, nil
	}

	return false, nil
}

// ExtractImports parses Python files and extracts import statements
func (d *Detector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Find all Python files
	pyFiles, err := d.findPythonFiles(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to find Python files: %w", err)
	}

	// Build layer map for quick lookup
	layerMap := make(map[string]*domain.Layer)
	for i := range layers {
		layerMap[layers[i].Name] = &layers[i]
	}

	// Parse each file
	for _, file := range pyFiles {
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

// findPythonFiles finds all Python files in the project
func (d *Detector) findPythonFiles(projectRoot string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common non-source directories
		if info.IsDir() {
			switch info.Name() {
			case "venv", ".venv", "env", ".env", "__pycache__", ".git", "node_modules":
				return filepath.SkipDir
			}
		}

		// Check if it's a Python file
		if strings.HasSuffix(path, ".py") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseFile extracts imports from a single Python file
func (d *Detector) parseFile(filePath, projectRoot string, layerMap map[string]*domain.Layer) ([]domain.Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dependencies []domain.Dependency

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Try each import pattern
		for _, pattern := range d.importPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}

			importPath := matches[1]

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

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dependencies, nil
}

// resolveImport resolves an import path to a layer
func (d *Detector) resolveImport(importPath, filePath, projectRoot string, layerMap map[string]*domain.Layer) string {
	// Handle relative imports
	if strings.HasPrefix(importPath, ".") {
		// Convert relative to absolute based on file location
		fileDir := filepath.Dir(filePath)
		relLevel := 0
		for strings.HasPrefix(importPath, ".") {
			relLevel++
			importPath = strings.TrimPrefix(importPath, ".")
		}

		// Go up directories based on dot count
		for i := 0; i < relLevel-1 && fileDir != projectRoot; i++ {
			fileDir = filepath.Dir(fileDir)
		}

		if importPath == "" {
			// Importing from same directory
			importPath = filepath.Base(fileDir)
		} else {
			importPath = filepath.Join(fileDir, strings.ReplaceAll(importPath, ".", "/"))
		}
	}

	// Try to match import path to a layer
	for name, layer := range layerMap {
		if layer.MatchesPath(importPath) {
			return name
		}

		// Also try matching the import path directly against layer paths
		for _, layerPath := range layer.Paths {
			// Convert glob pattern to check if import matches
			if d.importMatchesLayer(importPath, layerPath) {
				return name
			}
		}
	}

	// Try to resolve to a local module
	modulePath := d.resolveModulePath(importPath, projectRoot)
	if modulePath != "" {
		for name, layer := range layerMap {
			if layer.MatchesPath(modulePath) {
				return name
			}
		}
	}

	return ""
}

// importMatchesLayer checks if an import path matches a layer pattern
func (d *Detector) importMatchesLayer(importPath, layerPattern string) bool {
	// Convert glob pattern to regex
	// First, escape any regex special characters except * and ?
	escaped := regexp.QuoteMeta(layerPattern)
	// Then replace escaped * with regex patterns
	escaped = strings.ReplaceAll(escaped, `\*\*/`, "(/.*)?")
	escaped = strings.ReplaceAll(escaped, `\*\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\*`, "[^/]*")
	pattern := "^" + escaped + "$"

	matched, err := regexp.MatchString(pattern, importPath)
	if err != nil {
		return false
	}

	return matched
}

// resolveModulePath tries to resolve a module import to a file path
func (d *Detector) resolveModulePath(importPath, projectRoot string) string {
	// Try common Python module locations
	locations := []string{
		filepath.Join(projectRoot, strings.ReplaceAll(importPath, ".", "/")+".py"),
		filepath.Join(projectRoot, strings.ReplaceAll(importPath, ".", "/"), "__init__.py"),
		filepath.Join(projectRoot, "src", strings.ReplaceAll(importPath, ".", "/")+".py"),
		filepath.Join(projectRoot, "src", strings.ReplaceAll(importPath, ".", "/"), "__init__.py"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	return ""
}
