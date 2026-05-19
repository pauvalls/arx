package ports

import (
	"regexp"
	"strings"
)

// MatchImportToLayer checks if an import path matches a layer pattern.
// The layerPattern supports glob syntax: * (single segment), ** (zero or more segments).
// Import paths use forward slashes (e.g., "com/example/domain").
//
// This is shared across all language detectors to ensure consistent matching logic.
func MatchImportToLayer(importPath, layerPattern string) bool {
	// Convert glob pattern to regex
	// First escape all regex metacharacters
	escaped := regexp.QuoteMeta(layerPattern)

	// Replace /** with (/.*)? (matches zero or more path segments, including no segments)
	// Must do this BEFORE replacing single * to avoid conflicts
	escaped = strings.ReplaceAll(escaped, `/\*\*`, "(/.*)?")

	// Replace any remaining ** (without leading /) with .*
	escaped = strings.ReplaceAll(escaped, `\*\*`, ".*")

	// Replace escaped * with [^/]* (matches anything except /)
	escaped = strings.ReplaceAll(escaped, `\*`, "[^/]*")

	// Build final regex pattern
	pattern := "^" + escaped + "$"

	matched, err := regexp.MatchString(pattern, importPath)
	if err != nil {
		return false
	}

	return matched
}
