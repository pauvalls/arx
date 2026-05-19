package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// GenerateDOT creates a Graphviz DOT representation of the dependency graph
func GenerateDOT(data ports.DiagramData) string {
	var sb strings.Builder

	sb.WriteString("digraph ArxDependencies {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=filled];\n")
	sb.WriteString("\n")

	// Group files by layer
	layerFiles := make(map[string][]string)
	for _, dep := range data.Dependencies {
		layer := resolveLayer(dep.SourceFile, data.Layers)
		if layer != "" {
			if !contains(layerFiles[layer], dep.SourceFile) {
				layerFiles[layer] = append(layerFiles[layer], dep.SourceFile)
			}
		}
	}

	// Add target files that might not be sources
	for _, dep := range data.Dependencies {
		if dep.ResolvedLayer != "" {
			if !contains(layerFiles[dep.ResolvedLayer], dep.ImportPath) {
				layerFiles[dep.ResolvedLayer] = append(layerFiles[dep.ResolvedLayer], dep.ImportPath)
			}
		}
	}

	// Create subgraphs for each layer
	layerColors := map[string]string{
		"domain":         "lightblue",
		"application":    "lightgreen",
		"infrastructure": "lightyellow",
		"interface":      "lightpink",
		"ports":          "lightcyan",
		"adapters":       "lightcoral",
	}

	// Sort layer names for consistent output
	layerNames := make([]string, 0, len(layerFiles))
	for name := range layerFiles {
		layerNames = append(layerNames, name)
	}
	sort.Strings(layerNames)

	for _, layerName := range layerNames {
		files := layerFiles[layerName]
		if len(files) == 0 {
			continue
		}

		color := layerColors[layerName]
		if color == "" {
			color = "lightgray"
		}

		sb.WriteString(fmt.Sprintf("  subgraph cluster_%s {\n", sanitizeID(layerName)))
		sb.WriteString(fmt.Sprintf("    label=%q;\n", layerName))
		sb.WriteString(fmt.Sprintf("    style=filled;\n"))
		sb.WriteString(fmt.Sprintf("    color=%s;\n", color))

		for _, file := range files {
			nodeID := sanitizeID(file)
			label := extractFilename(file)
			sb.WriteString(fmt.Sprintf("    %q [label=%q];\n", nodeID, label))
		}

		sb.WriteString("  }\n\n")
	}

	// Create edges for dependencies
	if len(data.Dependencies) == 0 {
		sb.WriteString("  // No inter-layer dependencies detected\n")
	} else {
		// Build violation lookup for quick access
		violationSet := make(map[string]bool)
		for _, v := range data.Violations {
			key := fmt.Sprintf("%s->%s", v.File, v.TargetLayer)
			violationSet[key] = true
		}

		// Add edges
		for _, dep := range data.Dependencies {
			if dep.ResolvedLayer == "" {
				continue
			}

			sourceID := sanitizeID(dep.SourceFile)
			targetID := sanitizeID(dep.ImportPath)

			// Check if this is a violation
			isViolation := violationSet[fmt.Sprintf("%s->%s", dep.SourceFile, dep.ResolvedLayer)]

			if isViolation {
				sb.WriteString(fmt.Sprintf("  %q -> %q [color=red, penwidth=2, label=\"VIOLATION\"];\n", sourceID, targetID))
			} else {
				sb.WriteString(fmt.Sprintf("  %q -> %q [label=\"import\"];\n", sourceID, targetID))
			}
		}
	}

	sb.WriteString("}\n")

	return sb.String()
}

// resolveLayer finds the layer for a given file path
func resolveLayer(filePath string, layers []domain.Layer) string {
	for _, layer := range layers {
		for _, pattern := range layer.Paths {
			if matchPattern(pattern, filePath) {
				return layer.Name
			}
		}
	}
	return ""
}

// extractFilename extracts the filename from a path
func extractFilename(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

// sanitizeID makes a string safe for DOT node IDs
func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// matchPattern checks if a file path matches a glob pattern
func matchPattern(pattern, filePath string) bool {
	// Simple pattern matching for ** patterns
	pattern = strings.TrimSuffix(pattern, "/**")
	return strings.HasPrefix(filePath, pattern) || strings.Contains(filePath, "/"+pattern+"/")
}
