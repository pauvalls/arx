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

// ScanImports does a lightweight scan of Go files to find imports.
// Unlike the full detector pipeline, this is fast and doesn't require
// parsing the entire AST (uses regex).
func ScanImports(projectRoot string, layers []domain.Layer) (*ImportSummary, error) {
	summary := &ImportSummary{
		ByLayer: make(map[string]map[string]int),
	}

	// Build layer path matchers
	type layerMatcher struct {
		name  string
		paths []string
	}
	var matchers []layerMatcher
	for _, l := range layers {
		matchers = append(matchers, layerMatcher{name: l.Name, paths: l.Paths})
	}

	// Resolve layer for a file path
	resolveLayer := func(filePath string) string {
		rel, err := filepath.Rel(projectRoot, filePath)
		if err != nil {
			return "unknown"
		}
		for _, m := range matchers {
			for _, p := range m.paths {
				globPath := strings.TrimSuffix(p, "/**")
				if strings.HasPrefix(rel, globPath) || strings.HasPrefix(rel, strings.TrimPrefix(globPath, "./")) {
					return m.name
				}
			}
		}
		return "unknown"
	}

	// Regex for Go imports: import ( "path" or import "path"
	importRE := regexp.MustCompile(`^\s*import\s+(?:"([^"]+)"|\(|$)`)
	importLineRE := regexp.MustCompile(`^\s+(?:"([^"]+)"|([a-zA-Z0-9_]+)\s+"([^"]+)")`)

	// Walk Go files
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip vendor, node_modules, .git
		rel, _ := filepath.Rel(projectRoot, path)
		if strings.HasPrefix(rel, "vendor") || strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, "node_modules") {
			return nil
		}

		summary.FilesScanned++
		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		sourceLayer := resolveLayer(path)
		inImportBlock := false
		lineNum := 0
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
				// Parse import within block: "path" or alias "path"
				matches := importLineRE.FindStringSubmatch(line)
				if matches != nil {
					importPath := matches[1]
					if importPath == "" {
						importPath = matches[3]
					}
					if importPath != "" {
						targetLayer := resolveImportLayer(importPath, layers)
						summary.Entries = append(summary.Entries, ImportEntry{
							File: rel, Line: lineNum, Import: importPath,
							Language: "go", Layer: targetLayer,
						})
						summary.ImportsFound++
						// Track by layer
						if _, ok := summary.ByLayer[sourceLayer]; !ok {
							summary.ByLayer[sourceLayer] = make(map[string]int)
						}
						summary.ByLayer[sourceLayer][targetLayer]++
					}
				}
				continue
			}
			// Single import line: import "path"
			if matches := importRE.FindStringSubmatch(line); matches != nil && matches[1] != "" {
				targetLayer := resolveImportLayer(matches[1], layers)
				summary.Entries = append(summary.Entries, ImportEntry{
					File: rel, Line: lineNum, Import: matches[1],
					Language: "go", Layer: targetLayer,
				})
				summary.ImportsFound++
				if _, ok := summary.ByLayer[sourceLayer]; !ok {
					summary.ByLayer[sourceLayer] = make(map[string]int)
				}
				summary.ByLayer[sourceLayer][targetLayer]++
			}
		}
		return nil
	})

	return summary, err
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
