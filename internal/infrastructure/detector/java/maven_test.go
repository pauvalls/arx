package java

import (
	"os"
	"path/filepath"
	"testing"
)

// Test fixtures
const validPomXML = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>com.example</groupId>
    <artifactId>my-app</artifactId>
    <version>1.0.0</version>
    
    <modules>
        <module>module-a</module>
        <module>module-b</module>
        <module>module-c</module>
    </modules>
</project>`

const pomXMLNoModules = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>org.test</groupId>
    <artifactId>single-module</artifactId>
    <version>2.0.0</version>
</project>`

const invalidXML = `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.example
    <!-- Missing closing tag and invalid structure -->
</project>`

const emptyPomXML = ``

// TestMavenParser_ValidPom tests parsing a valid pom.xml with modules
func TestMavenParser_ValidPom(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(validPomXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.GroupID != "com.example" {
		t.Errorf("Expected GroupID 'com.example', got '%s'", result.GroupID)
	}

	if result.ArtifactID != "my-app" {
		t.Errorf("Expected ArtifactID 'my-app', got '%s'", result.ArtifactID)
	}

	if result.ModulePrefix != "com.example.my-app" {
		t.Errorf("Expected ModulePrefix 'com.example.my-app', got '%s'", result.ModulePrefix)
	}

	if len(result.Modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(result.Modules))
	}

	expectedModules := []string{"module-a", "module-b", "module-c"}
	for i, expected := range expectedModules {
		if i >= len(result.Modules) {
			t.Errorf("Missing module at index %d: %s", i, expected)
			continue
		}
		if result.Modules[i] != expected {
			t.Errorf("Expected module[%d] '%s', got '%s'", i, expected, result.Modules[i])
		}
	}
}

// TestMavenParser_NoModules tests parsing a pom.xml without modules section
func TestMavenParser_NoModules(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(pomXMLNoModules), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.GroupID != "org.test" {
		t.Errorf("Expected GroupID 'org.test', got '%s'", result.GroupID)
	}

	if result.ArtifactID != "single-module" {
		t.Errorf("Expected ArtifactID 'single-module', got '%s'", result.ArtifactID)
	}

	if result.ModulePrefix != "org.test.single-module" {
		t.Errorf("Expected ModulePrefix 'org.test.single-module', got '%s'", result.ModulePrefix)
	}

	if len(result.Modules) != 0 {
		t.Errorf("Expected 0 modules, got %d: %v", len(result.Modules), result.Modules)
	}
}

// TestMavenParser_InvalidXML tests parsing an invalid pom.xml
func TestMavenParser_InvalidXML(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(invalidXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)

	// Assert
	if err == nil {
		t.Fatal("Expected error for invalid XML, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for invalid XML, got non-nil")
	}
}

// TestMavenParser_ModulePrefix tests the module prefix calculation
func TestMavenParser_ModulePrefix(t *testing.T) {
	tests := []struct {
		name           string
		groupID        string
		artifactID     string
		expectedPrefix string
	}{
		{
			name:           "standard prefix",
			groupID:        "com.example",
			artifactID:     "my-app",
			expectedPrefix: "com.example.my-app",
		},
		{
			name:           "multi-level group",
			groupID:        "io.github.user",
			artifactID:     "project",
			expectedPrefix: "io.github.user.project",
		},
		{
			name:           "empty groupID",
			groupID:        "",
			artifactID:     "my-app",
			expectedPrefix: "",
		},
		{
			name:           "empty artifactID",
			groupID:        "com.example",
			artifactID:     "",
			expectedPrefix: "",
		},
		{
			name:           "both empty",
			groupID:        "",
			artifactID:     "",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pom.xml with specific groupID and artifactID
			pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>` + tt.groupID + `</groupId>
    <artifactId>` + tt.artifactID + `</artifactId>
</project>`

			tmpDir := t.TempDir()
			pomPath := filepath.Join(tmpDir, "pom.xml")
			
			if err := os.WriteFile(pomPath, []byte(pomContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Parse
			parser := NewMavenParser()
			result, err := parser.Parse(pomPath)

			// Assert
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result.ModulePrefix != tt.expectedPrefix {
				t.Errorf("Expected ModulePrefix '%s', got '%s'", tt.expectedPrefix, result.ModulePrefix)
			}
		})
	}
}

// TestMavenParser_EmptyFile tests parsing an empty pom.xml
func TestMavenParser_EmptyFile(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(emptyPomXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse
	parser := NewMavenParser()
	result, err := parser.Parse(pomPath)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result for empty file")
	}

	if len(result.Modules) != 0 {
		t.Errorf("Expected 0 modules for empty file, got %d", len(result.Modules))
	}
}

// TestMavenParser_FileNotFound tests parsing a non-existent pom.xml
func TestMavenParser_FileNotFound(t *testing.T) {
	parser := NewMavenParser()
	result, err := parser.Parse("/non/existent/path/pom.xml")

	// Assert
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for non-existent file, got non-nil")
	}
}

// TestGetModulePrefix tests the helper function
func TestGetModulePrefix(t *testing.T) {
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(validPomXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	prefix, err := GetModulePrefix(pomPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if prefix != "com.example.my-app" {
		t.Errorf("Expected prefix 'com.example.my-app', got '%s'", prefix)
	}
}

// TestGetModules tests the helper function
func TestGetModules(t *testing.T) {
	tmpDir := t.TempDir()
	pomPath := filepath.Join(tmpDir, "pom.xml")
	
	if err := os.WriteFile(pomPath, []byte(validPomXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	modules, err := GetModules(pomPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(modules))
	}
}

// TestHasMavenPom tests the helper function
func TestHasMavenPom(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Test without pom.xml
	if HasMavenPom(tmpDir) {
		t.Error("Expected false for directory without pom.xml")
	}

	// Test with pom.xml
	pomPath := filepath.Join(tmpDir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(validPomXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !HasMavenPom(tmpDir) {
		t.Error("Expected true for directory with pom.xml")
	}
}
