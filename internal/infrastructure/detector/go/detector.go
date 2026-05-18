package go_detector

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// Detector implements the ports.Detector interface for Go source files
type Detector struct {
	modulePrefix string
}

// New creates a new Go detector
func New() *Detector {
	return &Detector{}
}

// Name returns the detector name
func (d *Detector) Name() string {
	return "go"
}

// Detect checks if this detector can handle the given project
// Returns true if go.mod is present in the project root
func (d *Detector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	_, err := os.Stat(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking go.mod: %w", err)
	}

	// Read module prefix from go.mod
	modulePrefix, err := d.readModulePrefix(goModPath)
	if err != nil {
		return false, fmt.Errorf("reading module prefix: %w", err)
	}
	d.modulePrefix = modulePrefix

	return true, nil
}

// readModulePrefix extracts the module path from go.mod
func (d *Detector) readModulePrefix(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}

// ExtractImports extracts all dependencies from Go source files
// Returns a list of dependencies with resolved layers
func (d *Detector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	var dependencies []domain.Dependency

	// Load .arxignore patterns
	ignore, _ := domain.LoadArxIgnore(projectRoot)

	// Walk through all .go files
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

		// Skip directories and non-Go files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		// Skip vendor and hidden directories
		if strings.Contains(relPath, "vendor/") || strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}

		// Parse the Go file
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

// extractFileImports parses a single Go file and extracts its imports
func (d *Detector) extractFileImports(filePath, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	var dependencies []domain.Dependency

	// Extract imports from AST
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		// Get the position (line number)
		pos := fset.Position(imp.Pos())
		line := pos.Line

		// Resolve the import path to a layer
		resolvedLayer := d.resolveLayer(importPath, projectRoot, layers)

		// Get relative file path
		relPath, err := filepath.Rel(projectRoot, filePath)
		if err != nil {
			relPath = filePath
		}

		dependencies = append(dependencies, domain.Dependency{
			SourceFile:    relPath,
			SourceLine:    line,
			ImportPath:    importPath,
			ResolvedLayer: resolvedLayer,
			Language:      "go",
		})
	}

	return dependencies, nil
}

// resolveLayer determines which layer an import path belongs to
func (d *Detector) resolveLayer(importPath, projectRoot string, layers []domain.Layer) string {
	// First, try to match against module prefix
	if d.modulePrefix != "" && strings.HasPrefix(importPath, d.modulePrefix) {
		// This is a local import - extract the relative path
		relativePath := strings.TrimPrefix(importPath, d.modulePrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// Check which layer this path matches
		for _, layer := range layers {
			if layer.MatchesPath(relativePath) {
				return layer.Name
			}
		}
	}

	// Handle relative imports (./package or ../package)
	if strings.HasPrefix(importPath, ".") {
		// Convert relative import to absolute path relative to project root
		absPath := filepath.Join(filepath.Dir(projectRoot), importPath)
		relPath, err := filepath.Rel(projectRoot, absPath)
		if err == nil {
			for _, layer := range layers {
				if layer.MatchesPath(relPath) {
					return layer.Name
				}
			}
		}
	}

	// Handle internal/ packages (special Go semantics)
	if strings.HasPrefix(importPath, "internal/") {
		for _, layer := range layers {
			if layer.MatchesPath(importPath) {
				return layer.Name
			}
		}
	}

	// External dependency (not part of any layer)
	return ""
}

// Ensure Detector implements ports.Detector interface
var _ ports.Detector = (*Detector)(nil)
