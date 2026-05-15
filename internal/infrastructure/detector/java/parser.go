package java

import (
	"regexp"

	"github.com/pauvalls/arx/internal/infrastructure/detector/shared"
)

// Regex patterns for Java import statements
var (
	// Standard import: import package.Class;
	// Examples: import java.util.List; import com.example.domain.Order;
	standardImportPattern = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;`)

	// Static import: import static package.Class.FIELD;
	// Examples: import static java.lang.Math.PI; import static org.junit.Assert.assertEquals;
	staticImportPattern = regexp.MustCompile(`^\s*import\s+static\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*;`)

	// Wildcard import: import package.*;
	// Examples: import com.example.domain.*; import java.util.*;
	wildcardImportPattern = regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_.]*)\.\*\s*;`)

	// Package declaration: package com.example.app;
	packagePattern = regexp.MustCompile(`^\s*package\s+([a-zA-Z_][a-zA-Z0-9_.]*)\s*;`)
)

// extractImportsFromLine extracts all import paths from a single line of Java code
func extractImportsFromLine(line string) []string {
	var imports []string

	// Check standard import
	if matches := standardImportPattern.FindStringSubmatch(line); matches != nil {
		imports = append(imports, matches[1])
	}

	// Check static import
	if matches := staticImportPattern.FindStringSubmatch(line); matches != nil {
		imports = append(imports, matches[1])
	}

	// Check wildcard import
	if matches := wildcardImportPattern.FindStringSubmatch(line); matches != nil {
		// For wildcard imports, we keep the base package without the .*
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

// importMatchesLayer delegates to the shared MatchImportToLayer utility.
// This wrapper preserves the unexported name so existing callers in the
// java package and its tests work without changes.
func importMatchesLayer(importPath, layerPattern string) bool {
	return shared.MatchImportToLayer(importPath, layerPattern)
}
