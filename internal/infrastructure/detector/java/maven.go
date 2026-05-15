package java

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// MavenProject represents the structure of a pom.xml file
// We only unmarshal the fields we need for module detection
type MavenProject struct {
	XMLName    xml.Name       `xml:"project"`
	GroupID    string         `xml:"groupId"`
	ArtifactID string         `xml:"artifactId"`
	Modules    MavenModules   `xml:"modules"`
}

// MavenModules represents the <modules> section of pom.xml
type MavenModules struct {
	Module []string `xml:"module"`
}

// MavenParser parses pom.xml files to extract project information
type MavenParser struct{}

// NewMavenParser creates a new Maven parser instance
func NewMavenParser() *MavenParser {
	return &MavenParser{}
}

// ParseResult contains the extracted information from a pom.xml
type ParseResult struct {
	// GroupID is the Maven group ID (e.g., "com.example")
	GroupID string
	// ArtifactID is the Maven artifact ID (e.g., "my-app")
	ArtifactID string
	// ModulePrefix is the concatenation of GroupID and ArtifactID (e.g., "com.example.my-app")
	ModulePrefix string
	// Modules is the list of Maven module names
	Modules []string
}

// Parse reads and parses a pom.xml file
// Returns a ParseResult with extracted information
// Returns error if the file cannot be read or parsed
func (p *MavenParser) Parse(pomPath string) (*ParseResult, error) {
	// Read the file content
	content, err := os.ReadFile(pomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pom.xml: %w", err)
	}

	// Handle empty file
	if len(content) == 0 {
		return &ParseResult{
			Modules: []string{},
		}, nil
	}

	// Unmarshal XML
	var project MavenProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	// Build result
	result := &ParseResult{
		GroupID:    project.GroupID,
		ArtifactID: project.ArtifactID,
		Modules:    project.Modules.Module,
	}

	// Calculate module prefix if both groupID and artifactID are present
	if project.GroupID != "" && project.ArtifactID != "" {
		result.ModulePrefix = project.GroupID + "." + project.ArtifactID
	}

	// Ensure Modules is never nil
	if result.Modules == nil {
		result.Modules = []string{}
	}

	return result, nil
}

// GetModulePrefix extracts the module prefix from a pom.xml file
// Returns "${groupId}.${artifactId}" or empty string if not found
func GetModulePrefix(pomPath string) (string, error) {
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)
	if err != nil {
		return "", err
	}
	return result.ModulePrefix, nil
}

// GetModules extracts the list of module names from a pom.xml file
// Returns an empty slice if no modules are defined
func GetModules(pomPath string) ([]string, error) {
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)
	if err != nil {
		return nil, err
	}
	return result.Modules, nil
}

// HasMavenPom checks if a pom.xml file exists at the given project root
func HasMavenPom(projectRoot string) bool {
	pomPath := filepath.Join(projectRoot, "pom.xml")
	_, err := os.Stat(pomPath)
	return err == nil
}
