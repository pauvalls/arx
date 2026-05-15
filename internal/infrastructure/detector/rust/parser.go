package rust

import (
	"regexp"
	"strings"
)

// Regex patterns for Rust use statements
var (
	// Standard use: use path::to::Module;
	// Examples: use std::collections::HashMap;
	standardUsePattern = regexp.MustCompile(`^\s*use\s+([a-zA-Z_][a-zA-Z0-9_:]*)\s*;\s*$`)

	// Crate-relative use: use crate::path::to::Module;
	// Examples: use crate::domain::model::Order;
	crateUsePattern = regexp.MustCompile(`^\s*use\s+(crate::[a-zA-Z_][a-zA-Z0-9_:]*)\s*;\s*$`)

	// Self-relative use: use self::path::to::Module;
	// Examples: use self::submodule::Helper;
	selfUsePattern = regexp.MustCompile(`^\s*use\s+(self::[a-zA-Z_][a-zA-Z0-9_:]*)\s*;\s*$`)

	// Parent-relative use: use super::path::to::Module;
	// Examples: use super::parent_module::Something;
	superUsePattern = regexp.MustCompile(`^\s*use\s+(super::[a-zA-Z_][a-zA-Z0-9_:]*)\s*;\s*$`)

	// Re-export: pub use path::to::Module;
	// Examples: pub use crate::domain::Model;
	pubUsePattern = regexp.MustCompile(`^\s*pub\s+use\s+([a-zA-Z_][a-zA-Z0-9_:]*)\s*;\s*$`)

	// Module declaration: pub mod module_name;
	// Examples: pub mod models;
	pubModPattern = regexp.MustCompile(`^\s*pub\s+mod\s+([a-zA-Z_][a-zA-Z0-9_]+)\s*;\s*$`)
)

// extractImportsFromLine extracts all import paths from a single line of Rust code
func extractImportsFromLine(line string) []string {
	// Skip comment lines
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
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

	// Check pub use first (more specific because it starts with "pub use")
	if matches := pubUsePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check crate-relative
	if matches := crateUsePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check self-relative
	if matches := selfUsePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check super-relative
	if matches := superUsePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check standard use (matches any remaining use statements)
	if matches := standardUsePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
	}

	return imports
}

// extractPubMod extracts a module declaration from a line
func extractPubMod(line string) string {
	if matches := pubModPattern.FindStringSubmatch(line); matches != nil {
		return matches[1]
	}
	return ""
}

// isExternalDependency checks if an import is from external libraries
// (Rust standard library or common external crates)
func isExternalDependency(importPath string) bool {
	externalPrefixes := []string{
		"std::",
		"core::",
		"alloc::",
		"test::",
	}

	for _, prefix := range externalPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}

	return false
}
