package php

import (
	"regexp"
	"strings"
)

// Regex patterns for PHP import statements
var (
	// use Namespace\Class;
	// Examples: use App\Domain\Order;
	useStandardPattern = regexp.MustCompile(`^\s*use\s+([A-Z][A-Za-z0-9_]*(?:\\[A-Z][A-Za-z0-9_]*)+)\s*;\s*$`)

	// use Namespace\Class as Alias;
	// Examples: use App\Domain\Order as DomainOrder;
	useAliasPattern = regexp.MustCompile(`^\s*use\s+([A-Z][A-Za-z0-9_]*(?:\\[A-Z][A-Za-z0-9_]*)+)\s+as\s+\w+\s*;\s*$`)

	// use function Namespace\fn;
	// Examples: use function App\Helpers\format_money;
	useFunctionPattern = regexp.MustCompile(`^\s*use\s+function\s+([A-Za-z_][A-Za-z0-9_]*(?:\\[A-Za-z_][A-Za-z0-9_]*)+)\s*;\s*$`)

	// use const Namespace\CONST;
	// Examples: use const App\Constants\MAX_ITEMS;
	useConstPattern = regexp.MustCompile(`^\s*use\s+const\s+([A-Z_][A-Za-z0-9_]*(?:\\[A-Z_][A-Za-z0-9_]*)+)\s*;\s*$`)

	// require_once __DIR__ . '/path/to/file.php';
	// Examples: require_once __DIR__ . '/../Domain/Order.php';
	requireOncePattern = regexp.MustCompile(`^\s*require_once\s+__DIR__\s*\.\s*['"]\.?/([^'"]+)['"]\s*;?\s*$`)
)

// extractImportsFromLine extracts all import paths from a single line of PHP code
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

	// Check use ... as Alias first (more specific — aliased import)
	if matches := useAliasPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check use function (function import)
	if matches := useFunctionPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check use const (constant import)
	if matches := useConstPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check use standard (class/interface import)
	if matches := useStandardPattern.FindStringSubmatch(codePart); matches != nil {
		imports = append(imports, matches[1])
		return imports
	}

	// Check require_once (local file import)
	if matches := requireOncePattern.FindStringSubmatch(codePart); matches != nil {
		path := matches[1]
		// Go regex may capture a leading / — strip it for consistency
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		imports = append(imports, path)
		return imports
	}

	return imports
}

// isExternalDependency checks if an import is from an external library (Composer package).
// In PHP, use statements with vendor namespaces (e.g., Symfony, Doctrine) are external.
// require_once with relative paths (./ or ../) is always local.
func isExternalDependency(importPath string) bool {
	// Relative paths from require_once are always local
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		return false
	}

	// Check if the path contains a vendor/ prefix (from resolved file paths)
	if strings.HasPrefix(importPath, "vendor/") {
		return true
	}

	// Common external PHP package namespaces
	externalPrefixes := []string{
		"Symfony",
		"Doctrine",
		"GuzzleHttp",
		"Psr",
		"Monolog",
		"Composer",
	}

	for _, prefix := range externalPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"\\") {
			return true
		}
	}

	// use statements without relative paths are namespace-based;
	// if they don't match known external prefixes, treat as internal
	// (the project's own namespaces like App\, Domain\, etc.)
	return false
}
