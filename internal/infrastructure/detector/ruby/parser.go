package ruby

import (
	"regexp"
	"strings"
)

// Regex patterns for Ruby require statements
var (
	// require_relative 'path/to/file'
	// Examples: require_relative '../domain/order'
	requireRelativePattern = regexp.MustCompile(`^\s*require_relative\s+['"]([^'"]+)['"]\s*$`)

	// require_all 'path'
	// Examples: require_all 'lib/domain'
	requireAllPattern = regexp.MustCompile(`^\s*require_all\s+['"]([^'"]+)['"]\s*$`)

	// require File.expand_path('path', __dir__)
	// Examples: require File.expand_path('../domain/order', __dir__)
	requireExpandPattern = regexp.MustCompile(`^\s*require\s+File\.expand_path\s*\(\s*['"]([^'"]+)['"]`)

	// Standard require 'library' (external gem or local)
	// Examples: require 'rails', require 'sinatra', require 'bundler/setup'
	requirePattern = regexp.MustCompile(`^\s*require\s+['"]([^'"]+)['"]\s*$`)
)

// extractImportsFromLine extracts all import paths from a single line of Ruby code
func extractImportsFromLine(line string) []string {
	// Skip comment lines
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}

	// Check for inline comment after code — if there's code before a #, still process
	var codePart string
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		codePart = strings.TrimSpace(trimmed[:idx])
	} else {
		codePart = trimmed
	}

	if codePart == "" {
		return nil
	}

	var imports []string

	// Check require_relative first (more specific — local dependency)
	if matches := requireRelativePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check require_all (local dependency)
	if matches := requireAllPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check require File.expand_path (local dependency)
	if matches := requireExpandPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check standard require (matches any remaining require statements — typically gems)
	if matches := requirePattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
	}

	return imports
}

// isExternalDependency checks if an import is from an external library (gem)
// In Ruby, bare require 'gem_name' is external, while require_relative/require_all/File.expand_path are local.
// Since we only have the path string, we use heuristics:
// - Paths starting with './' or '../' are local (relative)
// - Paths containing 'lib/' or internal project paths are local
// - Bundler/rubygems are external
// - Everything else (bare gem names) is external
func isExternalDependency(importPath string) bool {
	// Bundler setup is infrastructure, not a domain dependency
	if importPath == "bundler/setup" {
		return true
	}

	// Ruby standard library / bundler
	externalPrefixes := []string{
		"rubygems",
		"bundler",
	}

	for _, prefix := range externalPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}

	// Relative paths are always local (from require_relative)
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return false
	}

	// Paths that look like project-internal references are local
	// (e.g., 'lib/domain/order', 'app/services/order_service')
	internalPrefixes := []string{
		"lib/",
		"app/",
		"config/",
		"spec/",
		"test/",
	}

	for _, prefix := range internalPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return false
		}
	}

	// Everything else is assumed to be an external gem
	return true
}
