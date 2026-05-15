package java

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGradleParser_ValidBuildGradle(t *testing.T) {
	// Create a temporary build.gradle file
	content := `
plugins {
    id 'java'
    id 'application'
}

group = 'com.example'
version = '1.0.0'

rootProject.name = 'myapp'

dependencies {
    implementation 'com.google.guava:guava:31.0.1-jre'
    testImplementation 'junit:junit:4.13.2'
}
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "com.example" {
		t.Errorf("Group = %q, want %q", result.Group, "com.example")
	}

	if result.ProjectName != "myapp" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myapp")
	}

	expectedPrefix := "com.example.myapp"
	if result.ModulePrefix != expectedPrefix {
		t.Errorf("ModulePrefix = %q, want %q", result.ModulePrefix, expectedPrefix)
	}
}

func TestGradleParser_NoGroup(t *testing.T) {
	// Create a temporary build.gradle file without group
	content := `
plugins {
    id 'java'
}

rootProject.name = 'myapp'

dependencies {
    implementation 'com.google.guava:guava:31.0.1-jre'
}
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "" {
		t.Errorf("Group = %q, want empty string", result.Group)
	}

	if result.ProjectName != "myapp" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myapp")
	}

	// ModulePrefix should be empty when group is missing
	if result.ModulePrefix != "" {
		t.Errorf("ModulePrefix = %q, want empty string", result.ModulePrefix)
	}
}

func TestGradleParser_NoName(t *testing.T) {
	// Create a temporary build.gradle file without project name
	content := `
plugins {
    id 'java'
}

group = 'com.example'
version = '1.0.0'

dependencies {
    implementation 'com.google.guava:guava:31.0.1-jre'
}
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "com.example" {
		t.Errorf("Group = %q, want %q", result.Group, "com.example")
	}

	if result.ProjectName != "" {
		t.Errorf("ProjectName = %q, want empty string", result.ProjectName)
	}

	// ModulePrefix should be empty when projectName is missing
	if result.ModulePrefix != "" {
		t.Errorf("ModulePrefix = %q, want empty string", result.ModulePrefix)
	}
}

func TestGradleParser_EmptyFile(t *testing.T) {
	// Create an empty build.gradle file
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "" {
		t.Errorf("Group = %q, want empty string", result.Group)
	}

	if result.ProjectName != "" {
		t.Errorf("ProjectName = %q, want empty string", result.ProjectName)
	}

	if result.ModulePrefix != "" {
		t.Errorf("ModulePrefix = %q, want empty string", result.ModulePrefix)
	}
}

func TestGradleParser_ProjectNameFallback(t *testing.T) {
	// Create a temporary build.gradle file with project.name instead of rootProject.name
	content := `
plugins {
    id 'java'
}

group = 'org.test'
project.name = 'testproject'
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "org.test" {
		t.Errorf("Group = %q, want %q", result.Group, "org.test")
	}

	if result.ProjectName != "testproject" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "testproject")
	}

	expectedPrefix := "org.test.testproject"
	if result.ModulePrefix != expectedPrefix {
		t.Errorf("ModulePrefix = %q, want %q", result.ModulePrefix, expectedPrefix)
	}
}

func TestGradleParser_DoubleQuotes(t *testing.T) {
	// Create a temporary build.gradle file with double quotes
	content := `
plugins {
    id 'java'
}

group = "com.doublequotes"
rootProject.name = "myproject"
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "com.doublequotes" {
		t.Errorf("Group = %q, want %q", result.Group, "com.doublequotes")
	}

	if result.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "myproject")
	}

	expectedPrefix := "com.doublequotes.myproject"
	if result.ModulePrefix != expectedPrefix {
		t.Errorf("ModulePrefix = %q, want %q", result.ModulePrefix, expectedPrefix)
	}
}

func TestGradleParser_WithComments(t *testing.T) {
	// Create a temporary build.gradle file with comments
	content := `
plugins {
    id 'java'
}

// This is the group for our project
group = 'com commented'

// rootProject.name = 'oldname'  // commented out
rootProject.name = 'newname'
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse the file
	parser := NewGradleParser()
	result, err := parser.Parse(gradlePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify results
	if result.Group != "com commented" {
		t.Errorf("Group = %q, want %q", result.Group, "com commented")
	}

	if result.ProjectName != "newname" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "newname")
	}

	expectedPrefix := "com commented.newname"
	if result.ModulePrefix != expectedPrefix {
		t.Errorf("ModulePrefix = %q, want %q", result.ModulePrefix, expectedPrefix)
	}
}

func TestGetGradleModulePrefix(t *testing.T) {
	content := `
group = 'com.example'
rootProject.name = 'testapp'
`
	tmpDir := t.TempDir()
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test GetGradleModulePrefix helper
	prefix, err := GetGradleModulePrefix(gradlePath)
	if err != nil {
		t.Fatalf("GetModulePrefix() error = %v", err)
	}

	expected := "com.example.testapp"
	if prefix != expected {
		t.Errorf("GetModulePrefix() = %q, want %q", prefix, expected)
	}
}

func TestHasGradleBuild(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with build.gradle present
	gradlePath := filepath.Join(tmpDir, "build.gradle")
	err := os.WriteFile(gradlePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !HasGradleBuild(tmpDir) {
		t.Error("HasGradleBuild() = false, want true")
	}

	// Test without build.gradle
	tmpDir2 := t.TempDir()
	if HasGradleBuild(tmpDir2) {
		t.Error("HasGradleBuild() = true, want false")
	}
}

func TestHasGradleBuildKts(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with build.gradle.kts present
	gradleKtsPath := filepath.Join(tmpDir, "build.gradle.kts")
	err := os.WriteFile(gradleKtsPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !HasGradleBuildKts(tmpDir) {
		t.Error("HasGradleBuildKts() = false, want true")
	}

	// Test without build.gradle.kts
	tmpDir2 := t.TempDir()
	if HasGradleBuildKts(tmpDir2) {
		t.Error("HasGradleBuildKts() = true, want false")
	}
}

func TestIsGradleFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"build.gradle", true},
		{"build.gradle.kts", true},
		{"pom.xml", false},
		{"settings.gradle", false},
		{"build.gradle.bak", false},
		{"gradle.properties", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isGradleFile(tt.filename); got != tt.want {
				t.Errorf("isGradleFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
