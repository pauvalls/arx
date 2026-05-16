package swift

import (
	"regexp"
	"strings"
)

// Regex patterns for Swift import statements
var (
	// @_exported import Module — re-export pattern (checked first)
	// Examples: @_exported import Foundation, @_exported import MyModule
	exportedImportPattern = regexp.MustCompile(`^\s*@_exported\s+import\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)

	// import struct|class|enum|protocol|typealias|let|var|func Module.Type — specific member import
	// Examples: import struct Foundation.URL, import class UIKit.UIView
	importMemberPattern = regexp.MustCompile(`^\s*import\s+(?:struct|class|enum|protocol|typealias|let|var|func)\s+([A-Za-z_][A-Za-z0-9_]*)\.\S+\s*$`)

	// import Module — standard import
	// Examples: import Foundation, import MyModule
	importStandardPattern = regexp.MustCompile(`^\s*import\s+([A-Za-z_][A-Za-z0-9_]*)\s*$`)
)

// extractImportsFromLine extracts all import paths from a single line of Swift code
func extractImportsFromLine(line string) []string {
	// Skip comment lines
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") {
		return nil
	}

	// Check for inline comment after code — if there's code before a //, still process
	var codePart string
	if idx := strings.Index(trimmed, "//"); idx >= 0 {
		codePart = strings.TrimSpace(trimmed[:idx])
	} else {
		codePart = trimmed
	}

	if codePart == "" {
		return nil
	}

	var imports []string

	// Check @_exported import first (more specific — re-export)
	if matches := exportedImportPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check specific member import (import struct/class/etc)
	if matches := importMemberPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check standard import
	if matches := importStandardPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
	}

	return imports
}

// isExternalDependency checks if an import is from a system framework
// Swift system modules are provided by Apple SDKs and are not project dependencies
func isExternalDependency(importPath string) bool {
	// Swift system module whitelist
	systemModules := map[string]bool{
		"Foundation":    true,
		"UIKit":         true,
		"SwiftUI":       true,
		"AppKit":        true,
		"CoreData":      true,
		"Combine":       true,
		"Dispatch":      true,
		"os":            true,
		"CoreGraphics":  true,
		"QuartzCore":    true,
	}

	return systemModules[importPath]
}
