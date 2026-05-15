package output

import (
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/application"
)

// GenerateASCII creates an ASCII art representation of the dependency graph
func GenerateASCII(result *application.DiagramResult) string {
	if len(result.Layers) == 0 {
		return "No layers defined in configuration\n"
	}

	var sb strings.Builder

	// Build layer dependency summary
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

	// Draw layer boxes with dependencies
	for i, layer := range result.Layers {
		// Draw layer box
		sb.WriteString(drawLayerBox(layer.Name, layerDeps[layer.Name], violationSet))

		// Draw dependency arrows to next layer
		if i < len(result.Layers)-1 {
			if deps, ok := layerDeps[layer.Name]; ok && len(deps) > 0 {
				for targetLayer, count := range deps {
					isViolation := violationSet[fmt.Sprintf("%s->%s", layer.Name, targetLayer)]
					sb.WriteString(drawDependencyArrow(layer.Name, targetLayer, count, isViolation))
				}
			}
		}
	}

	// Add summary
	sb.WriteString("\n")
	sb.WriteString(drawSummary(result))

	return sb.String()
}

// drawLayerBox creates a box representation for a layer
func drawLayerBox(name string, deps map[string]int, violations map[string]bool) string {
	var sb strings.Builder

	width := 50
	if len(name) > width-4 {
		width = len(name) + 4
	}

	// Top border
	sb.WriteString("┌" + strings.Repeat("─", width) + "┐\n")

	// Layer name
	sb.WriteString(fmt.Sprintf("│ %-*s │\n", width-2, centerText(name, width-4)))

	// Dependencies summary
	if len(deps) > 0 {
		sb.WriteString("├" + strings.Repeat("─", width) + "┤\n")
		for target, count := range deps {
			isViolation := violations[fmt.Sprintf("%s->%s", name, target)]
			prefix := "  "
			if isViolation {
				prefix = "[!] "
			}
			line := fmt.Sprintf("%s→ %s (%d)", prefix, target, count)
			if len(line) > width-2 {
				line = line[:width-3] + "…"
			}
			sb.WriteString(fmt.Sprintf("│ %-*s │\n", width-2, line))
		}
	}

	// Bottom border
	sb.WriteString("└" + strings.Repeat("─", width) + "┘\n")

	return sb.String()
}

// drawDependencyArrow creates an arrow between layers
func drawDependencyArrow(from, to string, count int, isViolation bool) string {
	var sb strings.Builder

	if isViolation {
		sb.WriteString(fmt.Sprintf("│\n"))
		sb.WriteString(fmt.Sprintf("▼ [VIOLATION] %d dependency/ies\n", count))
	} else {
		sb.WriteString(fmt.Sprintf("│\n"))
		sb.WriteString(fmt.Sprintf("▼ %d import/ies\n", count))
	}
	sb.WriteString("│\n")

	return sb.String()
}

// drawSummary creates a summary section
func drawSummary(result *application.DiagramResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString(" SUMMARY\n")
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Layers:        %d\n", len(result.Layers)))
	sb.WriteString(fmt.Sprintf("Dependencies:  %d\n", len(result.Dependencies)))
	sb.WriteString(fmt.Sprintf("Violations:    %d\n", len(result.Violations)))

	if len(result.Violations) > 0 {
		sb.WriteString("\n[!] = Violation\n")
	}

	return sb.String()
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := width - len(text)
	leftPadding := padding / 2
	rightPadding := padding - leftPadding
	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}
