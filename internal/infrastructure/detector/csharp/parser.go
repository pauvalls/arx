package csharp

import (
	"regexp"
	"strings"
)

// Regex patterns for C# using directives
var (
	// Standard using: using System;
	// Examples: using System.Collections.Generic;
	standardUsingPattern = regexp.MustCompile(`^\s*using\s+([a-zA-Z_][a-zA-Z0-9_.<>"\s,]+?)\s*;\s*$`)

	// Static using: using static System.Math;
	// Examples: using static System.Console;
	staticUsingPattern = regexp.MustCompile(`^\s*using\s+static\s+([a-zA-Z_][a-zA-Z0-9_.<>"\s,]+?)\s*;\s*$`)

	// Alias using: using Alias = Namespace.Class;
	// Examples: using StringList = System.Collections.Generic.List<string>;
	aliasUsingPattern = regexp.MustCompile(`^\s*using\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*([a-zA-Z_][a-zA-Z0-9_.<>"\s,]+?)\s*;\s*$`)

	// Namespace declaration: namespace MyApp.Domain
	// Examples: namespace MyApplication.Domain.Services;
	namespacePattern = regexp.MustCompile(`^\s*namespace\s+([a-zA-Z_][a-zA-Z0-9_.]+)\s*`)
)

// extractImportsFromLine extracts all import paths from a single line of C# code
func extractImportsFromLine(line string) []string {
	// Skip comment lines
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
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

	// Check alias using first (most specific pattern with = sign)
	if matches := aliasUsingPattern.FindStringSubmatch(codePart); matches != nil {
		// For alias using, we extract the actual namespace (second capture group)
		imports = append(imports, strings.TrimSpace(matches[2]))
		return imports
	}

	// Check static using (more specific than standard using)
	if matches := staticUsingPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, strings.TrimSpace(matches[1]))
		return imports
	}

	// Check standard using
	if matches := standardUsingPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, strings.TrimSpace(matches[1]))
	}

	return imports
}

// extractNamespace extracts the namespace declaration from a line
func extractNamespace(line string) string {
	if matches := namespacePattern.FindStringSubmatch(line); matches != nil {
		return matches[1]
	}
	return ""
}

// isExternalDependency checks if an import is from external libraries
// (C# standard library, .NET Framework, or common external packages)
func isExternalDependency(importPath string) bool {
	// Exact matches for namespaces without dots
	exactMatches := map[string]bool{
		"System":      true,
		"Mono":        true,
		"UnityEditor": true,
		"UnityEngine": true,
		"Xamarin":     true,
		"Windows":     true,
	}

	if exactMatches[importPath] {
		return true
	}

	// Prefix matches for namespaces with dots
	externalPrefixes := []string{
		"System.",
		"Microsoft.",
		"Mono.",
		"UnityEditor.",
		"UnityEngine.",
		"Xamarin.",
		"Windows.",
		"SystemRuntime.",
		"NETStandard.",
	}

	for _, prefix := range externalPrefixes {
		if strings.HasPrefix(importPath, prefix) {
			return true
		}
	}

	return false
}
