package java

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

var (
	gradleGroupPattern  = regexp.MustCompile(`group\s*=\s*['"]([^'"]+)['"]`)
	gradleNamePattern   = regexp.MustCompile(`(?:rootProject\.name|project\.name)\s*=\s*['"]([^'"]+)['"]`)
)

// parseGradleFile parses a build.gradle file and returns module prefix
func parseGradleFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var group, name string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "//") {
			continue
		}

		// Extract group
		if matches := gradleGroupPattern.FindStringSubmatch(line); len(matches) > 1 {
			group = matches[1]
		}

		// Extract name (prefer rootProject.name over project.name)
		if matches := gradleNamePattern.FindStringSubmatch(line); len(matches) > 1 {
			if name == "" || strings.Contains(line, "rootProject.name") {
				name = matches[1]
			}
		}
	}

	// Build module prefix
	if group != "" && name != "" {
		return group + "." + name, nil
	}

	return group, nil
}
