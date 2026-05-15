package csharp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestCSharpDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "csharp" {
		t.Errorf("Expected name 'csharp', got %q", name)
	}
}

func TestCSharpDetector_Detect_CsProj(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .csproj file
	content := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
</Project>`
	os.WriteFile(filepath.Join(tmpDir, "MyApp.csproj"), []byte(content), 0644)

	detector := New()
	isCSharp, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isCSharp {
		t.Error("Expected to detect C# project with .csproj")
	}
}

func TestCSharpDetector_Detect_Sln(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .sln file
	content := `Microsoft Visual Studio Solution File, Format Version 12.00
# Visual Studio Version 16
VisualStudioVersion = 16.0.30114.105
MinimumVisualStudioVersion = 10.0.40219.1
Project("{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}") = "MyApp", "MyApp.csproj", "{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}"
EndProject`
	os.WriteFile(filepath.Join(tmpDir, "MyApp.sln"), []byte(content), 0644)

	detector := New()
	isCSharp, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isCSharp {
		t.Error("Expected to detect C# project with .sln")
	}
}

func TestCSharpDetector_Detect_NotCSharpProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isCSharp, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isCSharp {
		t.Error("Expected to not detect C# project")
	}
}

func TestCSharpDetector_ShouldSkip_BinObj(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"bin base", "bin", true},
		{"bin with path", "/project/bin", true},
		{"bin nested path", "/project/bin/Debug", true},
		{"bin with suffix", "binary", false},
		{"obj base", "obj", true},
		{"obj with path", "/project/obj", true},
		{"obj nested path", "/project/obj/Release", true},
		{"obj with suffix", "object", false},
		{".vs base", ".vs", true},
		{".vscode base", ".vscode", true},
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

func TestCSharpDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"src/Program.cs", "/project/src/Program.cs", false},
		{"src/Domain/Model.cs", "/project/src/Domain/Model.cs", false},
		{"Application/Service.cs", "/project/Application/Service.cs", false},
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

func TestCSharpDetector_SkipsBinObj(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .cs file in bin/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "bin", "Debug"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "bin", "Debug", "app.cs"), []byte("class App {}"), 0644)

	// Create .cs file in obj/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "obj", "Release"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "obj", "Release", "app.cs"), []byte("class App {}"), 0644)

	// Create valid .cs file in src/
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "Program.cs"), []byte("class Program {}"), 0644)

	detector := New()
	files, err := detector.FindCSharpFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindCSharpFiles() error = %v", err)
	}

	// Should only find the file in src/, not in bin/ or obj/
	if len(files) != 1 {
		t.Errorf("Expected 1 C# file (bin/ and obj/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "Program.cs") {
		t.Errorf("Expected file in src/, got %q", files[0])
	}
}

func TestCSharpDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test C# file
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "Service.cs"), []byte("class Service {}"), 0644)

	// Create test file (should be skipped)
	os.WriteFile(filepath.Join(tmpDir, "src", "ServiceTest.cs"), []byte("class ServiceTest {}"), 0644)

	// Create another test file (should be skipped)
	os.WriteFile(filepath.Join(tmpDir, "src", "ServiceTests.cs"), []byte("class ServiceTests {}"), 0644)

	detector := New()
	files, err := detector.FindCSharpFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindCSharpFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 C# file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestCSharpDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "Domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "Infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "Application"), 0755)

	// Create C# file with various using types
	csContent := `using System;
using System.Collections.Generic;
using static System.Math;
using Alias = MyApp.Domain.Model;
using MyApp.Domain.Entities;
using MyApp.Infrastructure.Repositories;
using MyApp.Application.Services;

namespace MyApp.Application
{
    public class Service
    {
    }
}`
	os.WriteFile(filepath.Join(tmpDir, "Application", "Service.cs"), []byte(csContent), 0644)

	// Define layers
	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"Domain/**"},
		},
		{
			Name:  "infrastructure",
			Paths: []string{"Infrastructure/**"},
		},
		{
			Name:  "application",
			Paths: []string{"Application/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find imports (excluding System.*)
	if len(deps) < 3 {
		t.Errorf("Expected at least 3 dependencies (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if strings.Contains(dep.ImportPath, "Domain") && dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find Domain layer import resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestCSharpDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "Program.cs"), []byte("class Program {}"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestCSharpDetector_ExtractImports_SkipExternal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "Application"), 0755)

	csContent := `using System;
using System.Collections.Generic;
using System.Linq;
using Microsoft.EntityFrameworkCore;
using Mono.Posix;
using MyApp.Domain.Entities;

namespace MyApp.Application
{
    public class Service
    {
    }
}`
	os.WriteFile(filepath.Join(tmpDir, "Application", "Service.cs"), []byte(csContent), 0644)

	layers := []domain.Layer{
		{
			Name:  "domain",
			Paths: []string{"Domain/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should only find MyApp.Domain.Entities, NOT System.*, Microsoft.*, Mono.*
	for _, dep := range deps {
		if strings.HasPrefix(dep.ImportPath, "System.") {
			t.Errorf("Expected System.* to be skipped as external dependency, got: %s", dep.ImportPath)
		}
		if strings.HasPrefix(dep.ImportPath, "Microsoft.") {
			t.Errorf("Expected Microsoft.* to be skipped as external dependency, got: %s", dep.ImportPath)
		}
		if strings.HasPrefix(dep.ImportPath, "Mono.") {
			t.Errorf("Expected Mono.* to be skipped as external dependency, got: %s", dep.ImportPath)
		}
	}

	if len(deps) != 1 {
		t.Errorf("Expected exactly 1 dependency (domain), got %d", len(deps))
	}
}

func TestCSharpDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested bin/obj in submodules
	os.MkdirAll(filepath.Join(tmpDir, "Module1", "bin"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Module1", "bin", "app.cs"), []byte("class App {}"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "Module2", "obj"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Module2", "obj", "app.cs"), []byte("class App {}"), 0644)

	// Create valid C# files
	os.MkdirAll(filepath.Join(tmpDir, "Module1", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Module1", "src", "Program.cs"), []byte("class Program {}"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "Module2", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "Module2", "src", "Program.cs"), []byte("class Program {}"), 0644)

	detector := New()
	files, err := detector.FindCSharpFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindCSharpFiles() error = %v", err)
	}

	// Should find 2 files (Module1/src/Program.cs and Module2/src/Program.cs), skip bin/ and obj/
	if len(files) != 2 {
		t.Errorf("Expected 2 C# files (nested bin/ and obj/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
