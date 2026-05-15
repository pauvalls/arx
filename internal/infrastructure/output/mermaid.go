package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pauvalls/arx/internal/application"
)

// GenerateMermaid creates a Mermaid flowchart representation of the dependency graph
func GenerateMermaid(result *application.DiagramResult) string {
	var sb strings.Builder

	sb.WriteString("flowchart TD\n")
	sb.WriteString("\n")

	// Build layer dependency counts
	layerDeps := make(map[string]map[string]int)
	for _, dep := range result.Dependencies {
		sourceLayer := resolveLayer(dep.SourceFile, result.Layers)
		targetLayer := dep.ResolvedLayer
		if sourceLayer != "" && targetLayer != "" && sourceLayer != targetLayer {
			if layerDeps[sourceLayer] == nil {
				layerDeps[sourceLayer] = make(map[string]int)
			}
			layerDeps[sourceLayer][targetLayer]++
		}
	}

	// Build violation lookup
	violationSet := make(map[string]bool)
	for _, v := range result.Violations {
		key := fmt.Sprintf("%s->%s", v.SourceLayer, v.TargetLayer)
		violationSet[key] = true
	}

	// Sort layer names for consistent output
	layerNames := make([]string, 0, len(result.Layers))
	for _, layer := range result.Layers {
		layerNames = append(layerNames, layer.Name)
	}
	sort.Strings(layerNames)

	// Create subgraphs for each layer
	for _, layerName := range layerNames {
		sanitizedID := sanitizeMermaidID(layerName)
		sb.WriteString(fmt.Sprintf("  subgraph %s[\"%s\"]\n", sanitizedID, layerName))
		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", sanitizedID, layerName))
		sb.WriteString("  end\n")
		sb.WriteString("\n")
	}

	// Create edges for dependencies
	if len(layerDeps) == 0 {
		sb.WriteString("  %% No inter-layer dependencies detected\n")
	} else {
		// Sort source layers for consistent output
		sourceLayers := make([]string, 0, len(layerDeps))
		for source := range layerDeps {
			sourceLayers = append(sourceLayers, source)
		}
		sort.Strings(sourceLayers)

		for _, sourceLayer := range sourceLayers {
			targets := layerDeps[sourceLayer]
			
			// Sort target layers for consistent output
			targetList := make([]string, 0, len(targets))
			for target := range targets {
				targetList = append(targetList, target)
			}
			sort.Strings(targetList)

			for _, targetLayer := range targetList {
				count := targets[targetLayer]
				isViolation := violationSet[fmt.Sprintf("%s->%s", sourceLayer, targetLayer)]

				sourceID := sanitizeMermaidID(sourceLayer)
				targetID := sanitizeMermaidID(targetLayer)

				if isViolation {
					// Violation: dashed red edge
					sb.WriteString(fmt.Sprintf("  %s -.->|VIOLATION| %s\n", sourceID, targetID))
					sb.WriteString(fmt.Sprintf("  style %s --> %s stroke:red,stroke-dasharray: 5 5\n", sourceID, targetID))
				} else {
					// Clean dependency: solid green edge
					sb.WriteString(fmt.Sprintf("  %s -->|%d deps| %s\n", sourceID, count, targetID))
					sb.WriteString(fmt.Sprintf("  style %s --> %s stroke:green\n", sourceID, targetID))
				}
			}
		}
	}

	sb.WriteString("\n")

	// Add styling for layer nodes
	sb.WriteString("  %% Layer styling\n")
	layerColors := map[string]string{
		"domain":         "#E3F2FD",
		"application":    "#E8F5E9",
		"infrastructure": "#FFF9C4",
		"interface":      "#FCE4EC",
		"ports":          "#E0F7FA",
		"adapters":       "#FFEBEE",
	}

	for _, layerName := range layerNames {
		sanitizedID := sanitizeMermaidID(layerName)
		color := layerColors[layerName]
		if color == "" {
			color = "#F5F5F5"
		}
		sb.WriteString(fmt.Sprintf("  style %s fill:%s,stroke:#333,stroke-width:2px\n", sanitizedID, color))
	}

	return sb.String()
}

// sanitizeMermaidID makes a string safe for Mermaid node IDs
func sanitizeMermaidID(s string) string {
	// Replace special characters that Mermaid doesn't handle well in IDs
	s = strings.ReplaceAll(s, "-", "_dash_")
	s = strings.ReplaceAll(s, "_", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
