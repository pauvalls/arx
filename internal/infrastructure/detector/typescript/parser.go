package typescript_detector

import (
	"regexp"
	"strings"
)

// Regex patterns for TypeScript import statements
var (
	// Standard import: import { X } from "module", import X from "module",
	// import * as X from "module"
	importFromRegex = regexp.MustCompile(`import\s+(?:[\w\s{},*]+\s+from\s+)?["']([^"']+)["']`)

	// require("...")
	requireRegex = regexp.MustCompile(`require\s*\(\s*["']([^"']+)["']\s*\)`)

	// export { X } from "module" or export * from "module"
	exportFromRegex = regexp.MustCompile(`export\s+(?:\*|\{[^}]*\}|\w+)\s+from\s+["']([^"']+)["']`)

	// import type { ... } from "..."
	importTypeRegex = regexp.MustCompile(`import\s+type\s+\{[^}]*\}\s+from\s+["']([^"']+)["']`)

	// dynamic import: import("module") or await import("module")
	dynamicImportRegex = regexp.MustCompile(`\bimport\s*\(\s*["']([^"']+)["']\s*\)`)
)

// extractImportsFromLine extracts all import paths from a single line of TypeScript code.
func extractImportsFromLine(line string) []string {
	trimmed := strings.TrimSpace(line)

	// Skip comment lines
	if strings.HasPrefix(trimmed, "//") {
		return nil
	}

	var paths []string
	seen := make(map[string]bool)

	// Check import type { ... } from ...
	if matches := importTypeRegex.FindStringSubmatch(line); matches != nil {
		if !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
		// Don't let importFromRegex also match this line
		return paths
	}

	// Check import ... from
	if matches := importFromRegex.FindStringSubmatch(line); matches != nil {
		// Group 1 is the module path (group 0 is the full match)
		modulePath := matches[len(matches)-1]
		if modulePath != "" && !seen[modulePath] {
			seen[modulePath] = true
			paths = append(paths, modulePath)
		}
	}

	// Check require()
	if matches := requireRegex.FindStringSubmatch(line); matches != nil {
		if !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
	}

	// Check export ... from
	if matches := exportFromRegex.FindStringSubmatch(line); matches != nil {
		if !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
	}

	// Check dynamic imports
	if matches := dynamicImportRegex.FindStringSubmatch(line); matches != nil {
		if !seen[matches[1]] {
			seen[matches[1]] = true
			paths = append(paths, matches[1])
		}
	}

	return paths
}
