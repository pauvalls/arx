package java

import (
	"encoding/xml"
	"os"
)

// MavenPom represents a pom.xml file
type MavenPom struct {
	XMLName     xml.Name `xml:"project"`
	GroupID     string   `xml:"groupId"`
	ArtifactID  string   `xml:"artifactId"`
	Version     string   `xml:"version"`
	Packaging   string   `xml:"packaging"`
	Modules     []string `xml:"modules>module"`
}

// parseMavenPom parses a pom.xml file and returns module prefix
func parseMavenPom(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var pom MavenPom
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&pom); err != nil {
		return "", err
	}

	// Build module prefix from groupId and artifactId
	if pom.GroupID != "" && pom.ArtifactID != "" {
		return pom.GroupID + "." + pom.ArtifactID, nil
	}

	return pom.GroupID, nil
}

// getMavenModules returns modules defined in pom.xml
func getMavenModules(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var pom MavenPom
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&pom); err != nil {
		return nil, err
	}

	return pom.Modules, nil
}
