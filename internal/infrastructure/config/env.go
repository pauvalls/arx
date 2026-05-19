package config

import (
	"fmt"
	"os"
	"unicode"
)

// InterpolateEnvVars processes raw YAML bytes and replaces environment variable
// references with their values. Supports:
//   - $VAR and ${VAR} syntax
//   - ${VAR:-default} with default values
//   - $$ escape for literal $
//
// Only replaces in YAML scalar values, not keys.
// Missing variable without default returns an error.
func InterpolateEnvVars(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	result := make([]byte, 0, len(data))
	i := 0

	for i < len(data) {
		if data[i] == '$' && i+1 < len(data) {
			next := data[i+1]

			// $$ → literal $
			if next == '$' {
				result = append(result, '$')
				i += 2
				continue
			}

			// ${VAR} or ${VAR:-default}
			if next == '{' {
				end := findClosingBrace(data, i+2)
				if end == -1 {
					return nil, fmt.Errorf("unclosed ${ at position %d", i)
				}

				inner := string(data[i+2 : end])
				varName, defaultValue, hasDefault := parseBraceVar(inner)

				val, ok := os.LookupEnv(varName)
				if !ok {
					if hasDefault {
						result = append(result, []byte(defaultValue)...)
						i = end + 1
						continue
					}
					return nil, fmt.Errorf("environment variable %q is not set and has no default value", varName)
				}
				result = append(result, []byte(val)...)
				i = end + 1
				continue
			}

			// $VAR — unbraced variable (alphanumeric/underscore name)
			if isIdentStart(next) {
				start := i + 1
				end := start
				for end < len(data) && isIdentCont(data[end]) {
					end++
				}
				varName := string(data[start:end])

				val, ok := os.LookupEnv(varName)
				if !ok {
					return nil, fmt.Errorf("environment variable %q is not set", varName)
				}
				result = append(result, []byte(val)...)
				i = end
				continue
			}

			// Not a variable reference — keep $ as-is
			result = append(result, '$')
			i++
			continue
		}

		result = append(result, data[i])
		i++
	}

	return result, nil
}

// findClosingBrace finds the matching '}' for a "${" starting at position start.
// Returns the index of '}' or -1 if not found.
func findClosingBrace(data []byte, start int) int {
	for i := start; i < len(data); i++ {
		if data[i] == '}' {
			return i
		}
	}
	return -1
}

// parseBraceVar parses the content inside ${...}.
// Returns (varName, defaultValue, hasDefault).
func parseBraceVar(inner string) (string, string, bool) {
	for i := 0; i < len(inner); i++ {
		if inner[i] == ':' && i+1 < len(inner) && inner[i+1] == '-' {
			varName := inner[:i]
			defaultValue := inner[i+2:]
			return varName, defaultValue, true
		}
	}
	return inner, "", false
}

// isIdentStart returns true if b can start an env var name.
func isIdentStart(b byte) bool {
	return unicode.IsLetter(rune(b)) || b == '_'
}

// isIdentCont returns true if b can continue an env var name.
func isIdentCont(b byte) bool {
	return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || b == '_'
}
