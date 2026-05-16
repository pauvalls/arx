package swift

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestSwiftDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "swift" {
		t.Errorf("Expected name 'swift', got %q", name)
	}
}

func TestSwiftDetector_Detect_PackageSwift(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create Package.swift
	content := `// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "MyApp",
    targets: [
        .executableTarget(name: "MyApp"),
    ]
)`
	os.WriteFile(filepath.Join(tmpDir, "Package.swift"), []byte(content), 0644)

	detector := New()
	isSwift, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isSwift {
		t.Error("Expected to detect Swift project with Package.swift")
	}
}

func TestSwiftDetector_Detect_NotSwiftProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isSwift, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isSwift {
		t.Error("Expected to not detect Swift project")
	}
}

func TestSwiftDetector_ShouldSkip_BuildDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{".build base", ".build", true},
		{".build with path", "/project/.build", true},
		{".build nested path", "/project/.build/checkouts", true},
		{".build with suffix", "build", false},
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

func TestSwiftDetector_ShouldSkip_DerivedData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"DerivedData base", "DerivedData", true},
		{"DerivedData with path", "/project/DerivedData", true},
		{"DerivedData nested path", "/project/DerivedData/Build", true},
		{"derived data lowercase", "deriveddata", false},
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

func TestSwiftDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Sources/Domain", "/project/Sources/Domain", false},
		{"Sources/Application", "/project/Sources/Application", false},
		{"Sources/Infrastructure", "/project/Sources/Infrastructure", false},
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

func TestSwiftDetector_SkipsBuildDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .swift file in .build/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, ".build", "checkouts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".build", "checkouts", "generated.swift"), []byte("struct Generated {}"), 0644)

	// Create valid .swift file in Sources/
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "App"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Sources", "App", "main.swift"), []byte("print(\"hello\")"), 0644)

	detector := New()
	files, err := detector.FindSwiftFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindSwiftFiles() error = %v", err)
	}

	// Should only find the file in Sources/, not in .build/
	if len(files) != 1 {
		t.Errorf("Expected 1 Swift file (.build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "main.swift") {
		t.Errorf("Expected file in Sources/, got %q", files[0])
	}
}

func TestSwiftDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test Swift file
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "App"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Sources", "App", "main.swift"), []byte("print(\"hello\")"), 0644)

	// Create test file in Sources/ (should be skipped by *Tests.swift)
	os.WriteFile(filepath.Join(tmpDir, "Sources", "App", "AppTests.swift"), []byte("import XCTest"), 0644)

	// Create test file in Tests/ directory (should be skipped by directory)
	os.MkdirAll(filepath.Join(tmpDir, "Tests", "AppTests"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Tests", "AppTests", "tests.swift"), []byte("import XCTest"), 0644)

	detector := New()
	files, err := detector.FindSwiftFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindSwiftFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 Swift file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestSwiftDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "Domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "Infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "Application"), 0755)

	// Create Swift file with various import types
	swiftContent := `import Foundation
import Domain
import Infrastructure
import struct Foundation.URL

class OrderService {
    func create() {}
}`
	os.WriteFile(filepath.Join(tmpDir, "Sources", "Application", "OrderService.swift"), []byte(swiftContent), 0644)

	// Define layers
	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"domain/**"},
		},
		{
			Name:  "infrastructure",
			Paths: []string{"infrastructure/**"},
		},
		{
			Name:  "application",
			Paths: []string{"application/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find local imports (excluding system frameworks like Foundation)
	if len(deps) < 1 {
		t.Errorf("Expected at least 1 dependency (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if dep.ImportPath == "Domain" && dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find Domain import resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestSwiftDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "Sources", "App"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Sources", "App", "main.swift"), []byte("print(\"hello\")"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestSwiftDetector_ExtractImports_SkipSystemFrameworks(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "Sources", "Application"), 0755)

	swiftContent := `import Foundation
import UIKit
import SwiftUI
import Combine
import Dispatch
import Domain

class OrderService {}`
	os.WriteFile(filepath.Join(tmpDir, "Sources", "Application", "service.swift"), []byte(swiftContent), 0644)

	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"domain/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should only find local imports, NOT system frameworks
	for _, dep := range deps {
		if dep.ImportPath == "Foundation" {
			t.Error("Expected 'Foundation' to be skipped as system framework")
		}
		if dep.ImportPath == "UIKit" {
			t.Error("Expected 'UIKit' to be skipped as system framework")
		}
		if dep.ImportPath == "SwiftUI" {
			t.Error("Expected 'SwiftUI' to be skipped as system framework")
		}
		if dep.ImportPath == "Combine" {
			t.Error("Expected 'Combine' to be skipped as system framework")
		}
		if dep.ImportPath == "Dispatch" {
			t.Error("Expected 'Dispatch' to be skipped as system framework")
		}
	}
}

func TestSwiftDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested .build in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", ".build"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", ".build", "generated.swift"), []byte("struct Gen {}"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "DerivedData"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "DerivedData", "build.swift"), []byte("// build"), 0644)

	// Create valid Swift files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "Sources"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "Sources", "app.swift"), []byte("print(\"hello\")"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "Sources"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "Sources", "app.swift"), []byte("print(\"hello\")"), 0644)

	detector := New()
	files, err := detector.FindSwiftFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindSwiftFiles() error = %v", err)
	}

	// Should find 2 files, skip .build/ and DerivedData/
	if len(files) != 2 {
		t.Errorf("Expected 2 Swift files (nested .build/ and DerivedData/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
