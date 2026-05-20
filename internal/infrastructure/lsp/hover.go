package lsp

import (
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// ComputeHover computes hover information for a given hover request.
// It checks if the position is on an import line and returns layer information.
func ComputeHover(s *Server, params HoverParams) *Hover {
	uri := params.TextDocument.URI
	text := s.GetDocument(uri)
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	line := params.Position.Line
	if line < 0 || line >= len(lines) {
		return nil
	}

	lineText := lines[line]

	// Check if this is an import line — heuristic: contains import path
	importPath := extractImportPath(lineText)
	if importPath == "" {
		return nil
	}

	// Resolve the layer for this import path
	cfg := s.GetConfig()
	if cfg == nil {
		return nil
	}

	var matchedLayer *domain.Layer
	for i := range cfg.Layers {
		if cfg.Layers[i].MatchesPath(importPath) {
			matchedLayer = &cfg.Layers[i]
			break
		}
	}

	if matchedLayer == nil {
		return nil
	}

	// Build hover text
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Layer**: %s\n\n", matchedLayer.Name))
	if matchedLayer.Description != "" {
		b.WriteString(matchedLayer.Description)
		b.WriteString("\n\n")
	}

	// Find applicable rules
	var applicableRules []domain.Rule
	for _, rule := range cfg.Rules {
		if rule.From != "" || rule.Check.Raw != "" {
			applicableRules = append(applicableRules, rule)
		}
	}

	if len(applicableRules) > 0 {
		b.WriteString("**Applicable Rules**:\n")
		for _, rule := range applicableRules {
			b.WriteString(fmt.Sprintf("- %s: %s\n", rule.ID, rule.Explanation))
		}
	}

	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: b.String(),
		},
	}
}

// extractImportPath attempts to extract an import path from a line of code.
// Returns the import path if found, or empty string if the line is not an import.
func extractImportPath(lineText string) string {
	trimmed := strings.TrimSpace(lineText)

	// Go imports: "import \"path\"" or "import \"path\""
	if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import(") {
		// Extract path from within quotes
		if start := strings.Index(trimmed, "\""); start != -1 {
			end := strings.LastIndex(trimmed, "\"")
			if end > start {
				return trimmed[start+1 : end]
			}
		}
	}

	// TypeScript imports: import { ... } from "path" or import "path"
	if strings.HasPrefix(trimmed, "import ") {
		if fromIdx := strings.Index(trimmed, " from "); fromIdx != -1 {
			afterFrom := trimmed[fromIdx+6:]
			if start := strings.Index(afterFrom, "\""); start != -1 {
				end := strings.LastIndex(afterFrom, "\"")
				if end > start {
					return afterFrom[start+1 : end]
				}
			}
		}
		// side-effect import: import "path"
		if start := strings.Index(trimmed, "\""); start != -1 {
			end := strings.LastIndex(trimmed, "\"")
			if end > start && end < len(trimmed) {
				return trimmed[start+1 : end]
			}
		}
	}

	return ""
}
