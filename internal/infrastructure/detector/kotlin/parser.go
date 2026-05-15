package kotlin

import (
	"regexp"
	"strings"
)

// Regex patterns for Kotlin import statements
// Kotlin uses similar syntax to Java but with some differences:
// - No semicolons required (but allowed)
// - Supports import alias: import com.example.Something as S
// - No static imports
var (
	// Standard import: import package.Class
	// Examples: import kotlin.collections.List, import com.example.domain.Order
	// Optional trailing semicolon
	standardImportPattern = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_.]*[a-zA-Z0-9_*])\s*(?:as\s+[a-zA-Z_][a-zA-Z0-9_]*\s*)?(?:;)?\s*$`)

	// Wildcard import: import package.*
	// Examples: import com.example.domain.*
	wildcardImportPattern = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_.]*)\.\*\s*(?:;)?\s*$`)

	// Import alias: import package.Class as Alias
	// Examples: import com.example.domain.Order as DomainOrder
	importAliasPattern = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s+as\s+[a-zA-Z_][a-zA-Z0-9_]*\s*(?:;)?\s*$`)

	// Package declaration: package com.example.app
	packagePattern = regexp.MustCompile(`^\s*package\s+([a-zA-Z_][a-zA-Z0-9_.]*)\s*(?:;)?\s*$`)
)

// extractImportsFromLine extracts all import paths from a single line of Kotlin code
func extractImportsFromLine(line string) []string {
	var imports []string

	// Check wildcard import first (more specific pattern)
	if matches := wildcardImportPattern.FindStringSubmatch(line); matches != nil {
		// For wildcard imports, keep the base package without the .*
		imports = append(imports, matches[1])
		return imports
	}

	// Check import alias
	if matches := importAliasPattern.FindStringSubmatch(line); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check standard import (matches any remaining import statements)
	if matches := standardImportPattern.FindStringSubmatch(line); matches != nil {
		imports = append(imports, matches[1])
	}

	return imports
}

// extractPackage extracts the package declaration from a line
func extractPackage(line string) string {
	if matches := packagePattern.FindStringSubmatch(line); matches != nil {
		return matches[1]
	}
	return ""
}

// isExternalDependency checks if an import is from external libraries
// (Kotlin standard library, Java standard library, or JDK)
func isExternalDependency(importPath string) bool {
	// Kotlin standard library packages
	kotlinPackages := []string{
		"kotlin.",
		"kotlinx.",
	}

	// Java standard library packages
	javaPackages := []string{
		"java.",
		"javax.",
		"sun.",
		"com.sun.",
		"org.ietf.",
		"org.w3c.",
		"org.xml.",
	}

	for _, pkg := range kotlinPackages {
		if strings.HasPrefix(importPath, pkg) {
			return true
		}
	}

	for _, pkg := range javaPackages {
		if strings.HasPrefix(importPath, pkg) {
			return true
		}
	}

	return false
}
