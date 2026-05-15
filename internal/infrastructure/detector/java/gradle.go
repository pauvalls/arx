package java

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Regex patterns for build.gradle parsing
var (
	// group = 'com.example' or group = "com.example"
	groupRegex = regexp.MustCompile(`(?m)^\s*group\s*=\s*['"]([^'"]+)['"]`)

	// rootProject.name = 'myapp' or rootProject.name = "myapp"
	rootProjectNameRegex = regexp.MustCompile(`(?m)^\s*rootProject\.name\s*=\s*['"]([^'"]+)['"]`)

	// project.name = 'myapp' or project.name = "myapp" (fallback)
	projectNameRegex = regexp.MustCompile(`(?m)^\s*project\.name\s*=\s*['"]([^'"]+)['"]`)
)

// GradleParser parses build.gradle files to extract project information
type GradleParser struct{}

// NewGradleParser creates a new Gradle parser instance
func NewGradleParser() *GradleParser {
	return &GradleParser{}
}

// GradleParseResult contains the extracted information from a build.gradle
type GradleParseResult struct {
	// Group is the Gradle group ID (e.g., "com.example")
	Group string
	// ProjectName is the project name from rootProject.name or project.name
	ProjectName string
	// ModulePrefix is the concatenation of Group and ProjectName (e.g., "com.example.myapp")
	ModulePrefix string
}

// Parse reads and parses a build.gradle file
// Returns a GradleParseResult with extracted information
// Returns error if the file cannot be read or parsed
func (p *GradleParser) Parse(gradlePath string) (*GradleParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(gradlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read build.gradle: %w", err)
	}

	// Handle empty file
	if len(content) == 0 {
		return &GradleParseResult{
			ModulePrefix: "",
		}, nil
	}

	contentStr := string(content)

	// Extract group
	var group string
	if matches := groupRegex.FindStringSubmatch(contentStr); matches != nil {
		group = matches[1]
	}

	// Extract project name (try rootProject.name first, then project.name)
	var projectName string
	if matches := rootProjectNameRegex.FindStringSubmatch(contentStr); matches != nil {
		projectName = matches[1]
	} else if matches := projectNameRegex.FindStringSubmatch(contentStr); matches != nil {
		projectName = matches[1]
	}

	// Build result
	result := &GradleParseResult{
		Group:       group,
		ProjectName: projectName,
	}

	// Calculate module prefix if both group and projectName are present
	if group != "" && projectName != "" {
		result.ModulePrefix = group + "." + projectName
	}

	return result, nil
}

// GetGradleModulePrefix extracts the module prefix from a build.gradle file
// Returns "${group}.${projectName}" or empty string if not found
func GetGradleModulePrefix(gradlePath string) (string, error) {
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		return "", err
	}
	return result.ModulePrefix, nil
}

// HasGradleBuild checks if a build.gradle file exists at the given project root
func HasGradleBuild(projectRoot string) bool {
	gradlePath := projectRoot + "/build.gradle"
	_, err := os.Stat(gradlePath)
	return err == nil
}

// HasGradleBuildKts checks if a build.gradle.kts file exists at the given project root
func HasGradleBuildKts(projectRoot string) bool {
	gradleKtsPath := projectRoot + "/build.gradle.kts"
	_, err := os.Stat(gradleKtsPath)
	return err == nil
}

// extractModulePrefixFromGradle reads build.gradle and extracts the module prefix
// This is a helper for JavaDetector to use during Detect()
func extractModulePrefixFromGradle(gradlePath string) string {
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		// Log error but don't fail - modulePrefix will remain empty
		return ""
	}
	return result.ModulePrefix
}

// isGradleFile checks if a file is a Gradle build file (Groovy or Kotlin)
func isGradleFile(filename string) bool {
	return filename == "build.gradle" || filename == "build.gradle.kts"
}

// normalizeGradleContent strips comments and normalizes whitespace for parsing
// This helps handle multi-line declarations and comments
func normalizeGradleContent(content string) string {
	// Remove single-line comments
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		// Remove comments (but keep the rest of the line)
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}
