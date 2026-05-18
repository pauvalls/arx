package application

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// ImportEntry represents a single import found in a source file.
type ImportEntry struct {
	File     string // source file
	Line     int    // line number
	Import   string // full import path
	Language string // "go", etc
	Layer    string // resolved layer name (or "unknown")
}

// ImportSummary shows the dependency flow between layers.
type ImportSummary struct {
	Entries      []ImportEntry
	ByLayer      map[string]map[string]int // sourceLayer → targetLayer → count
	FilesScanned int
	ImportsFound int
}

// languageScanner defines how to scan imports for a specific language.
type languageScanner struct {
	extensions []string
	scannerFn  func(path string, layer string) ([]ImportEntry, error)
}

// ScanImports does a lightweight scan of source files to find imports
// across multiple languages. Uses regex (fast) — no AST parsing needed.
func ScanImports(projectRoot string, layers []domain.Layer) (*ImportSummary, error) {
	summary := &ImportSummary{
		ByLayer: make(map[string]map[string]int),
	}

	scanners := []languageScanner{
		{extensions: []string{".go"}, scannerFn: scanGoImports},
		{extensions: []string{".ts", ".tsx", ".js", ".jsx"}, scannerFn: scanTSImports},
	}

	resolveLayer := buildLayerResolver(projectRoot, layers)

	for _, lang := range scanners {
		for _, ext := range lang.extensions {
			err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if !strings.HasSuffix(path, ext) {
					return nil
				}
				rel, _ := filepath.Rel(projectRoot, path)
				if isSkippedDir(rel) {
					return nil
				}

				summary.FilesScanned++
				sourceLayer := resolveLayer(rel)
				entries, err := lang.scannerFn(path, sourceLayer)
				if err != nil {
					return nil
				}
				for _, e := range entries {
					e.File = rel
					e.Layer = resolveImportLayer(e.Import, layers)
					summary.Entries = append(summary.Entries, e)
					summary.ImportsFound++
					if _, ok := summary.ByLayer[sourceLayer]; !ok {
						summary.ByLayer[sourceLayer] = make(map[string]int)
					}
					summary.ByLayer[sourceLayer][e.Layer]++
				}
				return nil
			})
			if err != nil {
				return summary, err
			}
		}
	}

	return summary, nil
}

// isSkippedDir returns true for directory prefixes that should be skipped.
func isSkippedDir(rel string) bool {
	return strings.HasPrefix(rel, "vendor") ||
		strings.HasPrefix(rel, ".git") ||
		strings.HasPrefix(rel, "node_modules") ||
		strings.HasPrefix(rel, "dist") ||
		strings.HasPrefix(rel, "build") ||
		strings.HasPrefix(rel, "target")
}

// buildLayerResolver creates a function that resolves a file's layer from its path.
func buildLayerResolver(projectRoot string, layers []domain.Layer) func(string) string {
	type matcher struct {
		name  string
		paths []string
	}
	var matchers []matcher
	for _, l := range layers {
		matchers = append(matchers, matcher{name: l.Name, paths: l.Paths})
	}

	return func(relPath string) string {
		for _, m := range matchers {
			for _, p := range m.paths {
				globPath := strings.TrimSuffix(p, "/**")
				if strings.HasPrefix(relPath, globPath) || strings.HasPrefix(relPath, strings.TrimPrefix(globPath, "./")) {
					return m.name
				}
			}
		}
		return "unknown"
	}
}

// scanGoImports extracts import statements from a Go source file.
func scanGoImports(path, sourceLayer string) ([]ImportEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []ImportEntry
	inImportBlock := false
	lineNum := 0

	singleRE := regexp.MustCompile(`^\s*import\s+(?:"([^"]+)"|\(|$)`)
	blockRE := regexp.MustCompile(`^\s+(?:"([^"]+)"|([a-zA-Z0-9_]+)\s+"([^"]+)")`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
			continue
		}
		if inImportBlock {
			if strings.HasPrefix(line, ")") {
				inImportBlock = false
				continue
			}
			if matches := blockRE.FindStringSubmatch(line); matches != nil {
				importPath := matches[1]
				if importPath == "" {
					importPath = matches[3]
				}
				if importPath != "" {
					entries = append(entries, ImportEntry{Line: lineNum, Import: importPath, Language: "go"})
				}
			}
			continue
		}
		if matches := singleRE.FindStringSubmatch(line); matches != nil && matches[1] != "" {
			entries = append(entries, ImportEntry{Line: lineNum, Import: matches[1], Language: "go"})
		}
	}
	return entries, nil
}

// scanTSImports extracts import statements from TypeScript/JavaScript files.
func scanTSImports(path, sourceLayer string) ([]ImportEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []ImportEntry
	lineNum := 0

	// Match: import ... from '...' or import ... from "..."
	importRE := regexp.MustCompile(`^\s*import\s+(?:\{[^}]*\}\s*from\s+)?(?:\*\s*as\s+\w+\s+from\s+)?(?:\w+\s*,?\s*)?(?:\{[^}]*\}\s*from\s+)?['"]([^'"]+)['"]`)
	// Match: const ... = require('...') or var ... = require('...')
	requireRE := regexp.MustCompile(`(?:const|let|var)\s+\w+\s*=\s*require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	// Match: import '...' (side-effect import)
	sideEffectRE := regexp.MustCompile(`^\s*import\s+['"]([^'"]+)['"]`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if matches := importRE.FindStringSubmatch(line); matches != nil && matches[1] != "" {
			entries = append(entries, ImportEntry{Line: lineNum, Import: matches[1], Language: "typescript"})
		} else if matches := requireRE.FindStringSubmatch(line); matches != nil && matches[1] != "" {
			entries = append(entries, ImportEntry{Line: lineNum, Import: matches[1], Language: "typescript"})
		} else if matches := sideEffectRE.FindStringSubmatch(line); matches != nil && matches[1] != "" {
			entries = append(entries, ImportEntry{Line: lineNum, Import: matches[1], Language: "typescript"})
		}
	}
	return entries, nil
}

// resolveImportLayer tries to determine which layer an import belongs to.
func resolveImportLayer(importPath string, layers []domain.Layer) string {
	for _, l := range layers {
		for _, p := range l.Paths {
			globPath := strings.TrimSuffix(p, "/**")
			// Check if import path contains the layer path
			if strings.Contains(importPath, globPath) {
				return l.Name
			}
		}
	}
	return "external"
}

// ShortStats returns a one-line summary of the dependency scan.
func (s *ImportSummary) ShortStats() string {
	if s == nil || s.ImportsFound == 0 {
		return ""
	}
	var layerCount int
	for range s.ByLayer {
		layerCount++
	}
	return fmt.Sprintf("  Dependencies: %d imports across %d layers (%d files scanned)", s.ImportsFound, layerCount, s.FilesScanned)
}

// FormatSummary returns a human-readable dependency summary.
func (s *ImportSummary) FormatSummary() string {
	if s == nil || len(s.ByLayer) == 0 {
		return "  No dependencies detected."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  Files scanned: %d, Imports found: %d\n", s.FilesScanned, s.ImportsFound)
	b.WriteString("\n")

	// Sort layer names
	var layers []string
	for l := range s.ByLayer {
		layers = append(layers, l)
	}
	sort.Strings(layers)

	fmt.Fprintf(&b, "  %-16s %s\n", "Layer", "Dependencies")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", 50))
	for _, from := range layers {
		var targets []string
		for t := range s.ByLayer[from] {
			targets = append(targets, t)
		}
		sort.Strings(targets)
		first := true
		for _, to := range targets {
			count := s.ByLayer[from][to]
			if count == 0 {
				continue
			}
			if first {
				fromDisplay := from
				if len(fromDisplay) > 14 {
					fromDisplay = fromDisplay[:14]
				}
				fmt.Fprintf(&b, "  %-14s  → %s (%d)\n", fromDisplay, to, count)
				first = false
			} else {
				fmt.Fprintf(&b, "  %-14s  → %s (%d)\n", "", to, count)
			}
		}
	}
	return b.String()
}
