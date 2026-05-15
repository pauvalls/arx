package java

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestJavaDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "java" {
		t.Errorf("Expected name 'java', got %q", name)
	}
}

func TestJavaDetector_Detect_FindsJavaFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create pom.xml
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.example</groupId>
    <artifactId>my-app</artifactId>
</project>`
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(pomContent), 0644)

	detector := New()
	isJava, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isJava {
		t.Error("Expected to detect Java project with pom.xml")
	}
}

func TestJavaDetector_Detect_GradleProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create build.gradle
	gradleContent := `plugins {
    id 'java'
}

dependencies {
    implementation 'com.google.guava:guava:31.0.1-jre'
}`
	os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte(gradleContent), 0644)

	detector := New()
	isJava, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isJava {
		t.Error("Expected to detect Java project with build.gradle")
	}
}

func TestJavaDetector_Detect_NotJavaProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isJava, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isJava {
		t.Error("Expected to not detect Java project")
	}
}

func TestJavaDetector_Detect_SkipsTargetBuild(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create Java file in target/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "target", "classes"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "target", "classes", "App.class"), []byte("fake class"), 0644)

	// Create Java file in build/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "build", "classes"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "build", "classes", "App.class"), []byte("fake class"), 0644)

	// Create valid Java file in src/main/java
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java"), 0755)
	javaContent := `package com.example;
import java.util.List;

public class App {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "java", "App.java"), []byte(javaContent), 0644)

	detector := New()
	files, err := detector.FindJavaFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindJavaFiles() error = %v", err)
	}

	// Should only find the file in src/main/java, not in target/ or build/
	if len(files) != 1 {
		t.Errorf("Expected 1 Java file (target/ and build/ skipped), got %d", len(files))
	}
	if !strings.HasSuffix(files[0], "App.java") {
		t.Errorf("Expected file in src/main/java, got %q", files[0])
	}
}

func TestJavaDetector_ShouldSkip_Target(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"target base", "target", true},
		{"target with path", "/project/target", true},
		{"target nested path", "/project/target/classes", true},
		{"target with suffix", "target-gen", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := shouldSkipPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestJavaDetector_ShouldSkip_Build(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"build base", "build", true},
		{"build with path", "/project/build", true},
		{"build nested path", "/project/build/classes/java/main", true},
		{"build with suffix", "buildkit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := shouldSkipPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestJavaDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	// Test nested build directories in multi-module projects
	tmpDir := t.TempDir()

	// Create nested target/build in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", "target"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "target", "App.java"), []byte("fake"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "build"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "build", "App.java"), []byte("fake"), 0644)

	// Create valid Java files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "src", "main", "java"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "src", "main", "java", "App1.java"), []byte("package com.example;"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "src", "main", "java"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "src", "main", "java", "App2.java"), []byte("package com.example;"), 0644)

	detector := New()
	files, err := detector.FindJavaFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindJavaFiles() error = %v", err)
	}

	// Should find 2 files (App1.java and App2.java), skip files in target/ and build/
	if len(files) != 2 {
		t.Errorf("Expected 2 Java files (nested target/ and build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestJavaDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"src/main/java", "/project/src/main/java", false},
		{"src/test/java", "/project/src/test/java", false},
		{"custom source", "/project/sources/java", false},
		{"deep nested source", "/project/module/src/main/java/com/example", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := shouldSkipPath(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkipPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestJavaDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "app"), 0755)

	// Create Java file with various import types
	javaContent := `package com.example.app;

import java.util.List;
import java.util.ArrayList;
import com.example.domain.Order;
import com.example.infrastructure.Database;
import static java.lang.Math.PI;
import static org.junit.Assert.assertEquals;
import com.example.domain.*;

public class OrderService {
    // Some code here
}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "app", "OrderService.java"), []byte(javaContent), 0644)

	// Define layers
	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"com/example/domain/**"},
		},
		{
			Name:  "infrastructure",
			Paths: []string{"com/example/infrastructure/**"},
		},
		{
			Name:  "application",
			Paths: []string{"com/example/app/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find imports (excluding java.* standard library)
	// Expected: com.example.domain.Order, com.example.infrastructure.Database, com.example.domain (wildcard), org.junit.Assert
	if len(deps) < 3 {
		t.Errorf("Expected at least 3 dependencies (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if dep.ImportPath == "com.example.domain.Order" && dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find com.example.domain.Order resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestJavaDetector_ExtractImports_InvalidSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "app"), 0755)

	// Create Java file with invalid import syntax
	javaContent := `package com.example.app;

import java.util.List
import static java.lang.Math.PI;
import com.example.domain.Order;

public class InvalidService {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "java", "com", "example", "app", "InvalidService.java"), []byte(javaContent), 0644)

	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"com/example/domain/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() should not fail on invalid syntax, got error = %v", err)
	}

	// Should still extract valid imports (static import and standard import are valid)
	// The function should be resilient to syntax errors
	// At minimum, the static import should be found
	if len(deps) == 0 {
		t.Error("Expected to extract at least some valid imports despite syntax errors")
		for _, dep := range deps {
			t.Logf("  Found: %+v", dep)
		}
	}
}

func TestJavaDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "java"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "java", "App.java"), []byte("package com.example;"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func Test_extractImportsFromLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "standard import",
			line:     "import java.util.List;",
			expected: []string{"java.util.List"},
		},
		{
			name:     "static import",
			line:     "import static java.lang.Math.PI;",
			expected: []string{"java.lang.Math.PI"},
		},
		{
			name:     "wildcard import",
			line:     "import com.example.domain.*;",
			expected: []string{"com.example.domain"},
		},
		{
			name:     "import with spaces",
			line:     "  import   java.util.ArrayList  ;  ",
			expected: []string{"java.util.ArrayList"},
		},
		{
			name:     "not an import",
			line:     "public class MyClass {}",
			expected: []string{},
		},
		{
			name:     "comment line",
			line:     "// import fake.Import;",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractImportsFromLine(tt.line)
			if len(result) != len(tt.expected) {
				t.Errorf("extractImportsFromLine(%q) returned %d imports, expected %d",
					tt.line, len(result), len(tt.expected))
			}
			for i, exp := range tt.expected {
				if i >= len(result) || result[i] != exp {
					t.Errorf("extractImportsFromLine(%q)[%d] = %q, want %q",
						tt.line, i, result[i], exp)
				}
			}
		})
	}
}

func Test_extractPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "valid package",
			line:     "package com.example.app;",
			expected: "com.example.app",
		},
		{
			name:     "package with spaces",
			line:     "  package   com.example.app  ;  ",
			expected: "com.example.app",
		},
		{
			name:     "not a package",
			line:     "import com.example.app;",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractPackage(tt.line)
			if result != tt.expected {
				t.Errorf("extractPackage(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func Test_isExternalDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		importPath string
		expected bool
	}{
		{"java standard", "java.util.List", true},
		{"javax standard", "javax.servlet.http.HttpServletRequest", true},
		{"sun internal", "sun.misc.Unsafe", true},
		{"com.sun", "com.sun.net.httpserver.HttpServer", true},
		{"custom domain", "com.example.domain.Order", false},
		{"spring framework", "org.springframework.boot.SpringApplication", false},
		{"junit", "org.junit.Assert", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isExternalDependency(tt.importPath)
			if result != tt.expected {
				t.Errorf("isExternalDependency(%q) = %v, want %v",
					tt.importPath, result, tt.expected)
			}
		})
	}
}

func Test_importMatchesLayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		importPath   string
		layerPattern string
		expected     bool
	}{
		{"exact match", "com/example/domain", "com/example/domain/**", true},
		{"nested match", "com/example/domain/order", "com/example/domain/**", true},
		{"no match", "com/example/infrastructure", "com/example/domain/**", false},
		{"single star", "com/example/domain", "com/example/*", true},
		{"single star no nested", "com/example/domain/order", "com/example/*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := importMatchesLayer(tt.importPath, tt.layerPattern)
			if result != tt.expected {
				t.Errorf("importMatchesLayer(%q, %q) = %v, want %v",
					tt.importPath, tt.layerPattern, result, tt.expected)
			}
		})
	}
}
