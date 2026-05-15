package kotlin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestKotlinDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "kotlin" {
		t.Errorf("Expected name 'kotlin', got %q", name)
	}
}

func TestKotlinDetector_Detect_BuildGradleKts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create build.gradle.kts
	content := `plugins {
    kotlin("jvm") version "1.9.0"
}`
	os.WriteFile(filepath.Join(tmpDir, "build.gradle.kts"), []byte(content), 0644)

	detector := New()
	isKotlin, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isKotlin {
		t.Error("Expected to detect Kotlin project with build.gradle.kts")
	}
}

func TestKotlinDetector_Detect_PomXmlWithKtFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create pom.xml
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(`<?xml version="1.0"?><project><groupId>com.example</groupId><artifactId>app</artifactId></project>`), 0644)

	// Create a .kt file
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "App.kt"), []byte("package com.example"), 0644)

	detector := New()
	isKotlin, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isKotlin {
		t.Error("Expected to detect Kotlin project with pom.xml and .kt files")
	}
}

func TestKotlinDetector_Detect_PomXmlWithoutKtFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create pom.xml only (no .kt files)
	os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(`<?xml version="1.0"?><project><groupId>com.example</groupId><artifactId>app</artifactId></project>`), 0644)

	detector := New()
	isKotlin, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isKotlin {
		t.Error("Expected to NOT detect Kotlin project without .kt files (pom.xml alone)")
	}
}

func TestKotlinDetector_Detect_SettingsGradleKts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create settings.gradle.kts
	os.WriteFile(filepath.Join(tmpDir, "settings.gradle.kts"), []byte(`rootProject.name = "myapp"`), 0644)

	detector := New()
	isKotlin, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isKotlin {
		t.Error("Expected to detect Kotlin project with settings.gradle.kts")
	}
}

func TestKotlinDetector_Detect_NotKotlinProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isKotlin, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isKotlin {
		t.Error("Expected to not detect Kotlin project")
	}
}

func TestKotlinDetector_ShouldSkip_Target(t *testing.T) {
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

func TestKotlinDetector_ShouldSkip_Build(t *testing.T) {
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

func TestKotlinDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"src/main/kotlin", "/project/src/main/kotlin", false},
		{"src/test/kotlin", "/project/src/test/kotlin", false},
		{"custom source", "/project/src/main/kotlin/com/example", false},
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

func TestKotlinDetector_Detect_SkipsTargetBuild(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .kt file in target/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "target", "classes"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "target", "classes", "App.kt"), []byte("package com.example"), 0644)

	// Create .kt file in build/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "build", "classes"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "build", "classes", "App.kt"), []byte("package com.example"), 0644)

	// Create valid .kt file in src/main/kotlin
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "App.kt"), []byte("package com.example"), 0644)

	detector := New()
	files, err := detector.FindKotlinFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindKotlinFiles() error = %v", err)
	}

	// Should only find the file in src/main/kotlin, not in target/ or build/
	if len(files) != 1 {
		t.Errorf("Expected 1 Kotlin file (target/ and build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "App.kt") {
		t.Errorf("Expected file in src/main/kotlin, got %q", files[0])
	}
}

func TestKotlinDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test Kotlin file
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "Service.kt"), []byte("package com.example"), 0644)

	// Create test file (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "src", "test", "kotlin", "com", "example"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "test", "kotlin", "com", "example", "ServiceTest.kt"), []byte("package com.example"), 0644)

	// Create another test file with Tests suffix
	os.WriteFile(filepath.Join(tmpDir, "src", "test", "kotlin", "com", "example", "ServiceTests.kt"), []byte("package com.example"), 0644)

	detector := New()
	files, err := detector.FindKotlinFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindKotlinFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 Kotlin file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestKotlinDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app"), 0755)

	// Create Kotlin file with various import types
	ktContent := `package com.example.app

import java.util.List
import com.example.domain.Order
import com.example.infrastructure.Database
import com.example.domain.*
import com.example.domain.Order as DomainOrder

class OrderService {
    // Some code here
}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app", "OrderService.kt"), []byte(ktContent), 0644)

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
	// Expected: com.example.domain.Order, com.example.infrastructure.Database, com.example.domain (wildcard)
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

	// Check that we found the wildcard domain import
	foundWildcard := false
	for _, dep := range deps {
		if dep.ImportPath == "com.example.domain" && dep.ResolvedLayer == "domain" {
			foundWildcard = true
			break
		}
	}
	if !foundWildcard {
		t.Error("Expected to find com.example.domain (wildcard) resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestKotlinDetector_ExtractImports_InvalidSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app"), 0755)

	// Create Kotlin file with some valid and invalid syntax
	ktContent := `package com.example.app

import java.util.List
import com.example.domain.Order
import com.example                                    // invalid import (no class/package after)
import com.example.app.InvalidService

class InvalidService {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app", "InvalidService.kt"), []byte(ktContent), 0644)

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

	// Should still extract valid imports
	if len(deps) == 0 {
		t.Error("Expected to extract at least some valid imports despite syntax issues")
	}
}

func TestKotlinDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "App.kt"), []byte("package com.example"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestKotlinDetector_ExtractImports_SkipKotlinStdLib(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app"), 0755)

	ktContent := `package com.example.app

import kotlin.collections.List
import kotlinx.coroutines.Deferred
import com.example.domain.Order

class AppService {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example", "app", "AppService.kt"), []byte(ktContent), 0644)

	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"com/example/domain/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should only find com.example.domain.Order, NOT kotlin.* or kotlinx.*
	for _, dep := range deps {
		if dep.ImportPath == "kotlin.collections.List" {
			t.Error("Expected kotlin.collections.List to be skipped as external dependency")
		}
		if dep.ImportPath == "kotlinx.coroutines.Deferred" {
			t.Error("Expected kotlinx.coroutines.Deferred to be skipped as external dependency")
		}
	}

	if len(deps) != 1 {
		t.Errorf("Expected exactly 1 dependency (domain), got %d", len(deps))
	}
}

func TestKotlinDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested target/build in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", "target"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "target", "App.kt"), []byte("fake"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "build"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "build", "App.kt"), []byte("fake"), 0644)

	// Create valid Kotlin files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "src", "main", "kotlin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "src", "main", "kotlin", "App1.kt"), []byte("package com.example"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "src", "main", "kotlin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "src", "main", "kotlin", "App2.kt"), []byte("package com.example"), 0644)

	detector := New()
	files, err := detector.FindKotlinFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindKotlinFiles() error = %v", err)
	}

	// Should find 2 files (App1.kt and App2.kt), skip files in target/ and build/
	if len(files) != 2 {
		t.Errorf("Expected 2 Kotlin files (nested target/ and build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
