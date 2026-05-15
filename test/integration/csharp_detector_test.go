package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/csharp"
)

func TestCSharpDetector_CsProjFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("csharp-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("C# fixture not found at %s", fixturePath)
	}

	detector := csharp.New()

	// Detect
	isCSharp, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isCSharp {
		t.Error("Expected to detect C# project with .csproj")
	}

	// Extract imports
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"Domain/**"}},
		{Name: "infrastructure", Paths: []string{"Infrastructure/**"}},
		{Name: "application", Paths: []string{"Application/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find domain, infrastructure, and application layer imports
	foundDomain := false
	foundInfrastructure := false
	foundApplication := false
	for _, dep := range deps {
		if dep.ResolvedLayer == "domain" {
			foundDomain = true
		}
		if dep.ResolvedLayer == "infrastructure" {
			foundInfrastructure = true
		}
		if dep.ResolvedLayer == "application" {
			foundApplication = true
		}
	}

	if !foundDomain {
		t.Error("Expected to find domain layer imports")
	}
	if !foundInfrastructure {
		t.Error("Expected to find infrastructure layer imports")
	}
	if !foundApplication {
		t.Error("Expected to find application layer imports")
	}

	// Should skip test files (*Test.cs, *Tests.cs)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "UserTest.cs" {
			t.Error("Should skip test files (*Test.cs)")
		}
	}

	// Should skip external dependencies (System.*, Microsoft.*)
	for _, dep := range deps {
		if dep.ImportPath == "System" || dep.ImportPath == "System.Collections.Generic" {
			t.Error("Should skip System.* as external dependency")
		}
		if dep.ImportPath == "Microsoft.EntityFrameworkCore" {
			t.Error("Should skip Microsoft.EntityFrameworkCore as external dependency")
		}
	}
}

func TestCSharpDetector_SkipsBinObj(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("csharp-project")

	detector := csharp.New()
	files, err := detector.FindCSharpFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindCSharpFiles() error = %v", err)
	}

	// Should not find files in bin/ or obj/ directories
	for _, f := range files {
		relPath, err := filepath.Rel(fixturePath, f)
		if err != nil {
			continue
		}
		
		// Check if path contains bin or obj directory
		parts := filepath.SplitList(relPath)
		for _, part := range parts {
			if part == "bin" || part == "obj" {
				t.Errorf("Should skip bin/obj/ directories, found: %s", f)
			}
		}
		
		// Alternative check: look for bin/ or obj/ in path
		if filepath.Base(filepath.Dir(f)) == "bin" || 
		   filepath.Base(filepath.Dir(f)) == "obj" ||
		   filepath.Base(filepath.Dir(filepath.Dir(f))) == "bin" ||
		   filepath.Base(filepath.Dir(filepath.Dir(f))) == "obj" {
			t.Errorf("Should skip bin/obj/ directories, found: %s", f)
		}
	}
}

func TestCSharpDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("csharp-project")

	detector := csharp.New()

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"Domain/**"}},
		{Name: "infrastructure", Paths: []string{"Infrastructure/**"}},
		{Name: "application", Paths: []string{"Application/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	for _, dep := range deps {
		t.Logf("  Dependency: source=%s import=%s layer=%s",
			filepath.Base(dep.SourceFile), dep.ImportPath, dep.ResolvedLayer)
	}

	// Check that UserRepository.cs (infrastructure) correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "UserRepository.cs" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected UserRepository.cs (infrastructure) to have domain layer imports")
	}

	// Check that UserService.cs (application) correctly resolves infrastructure imports
	appInfraImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "UserService.cs" && dep.ResolvedLayer == "infrastructure" {
			appInfraImports = true
			break
		}
	}
	if !appInfraImports {
		t.Error("Expected UserService.cs (application) to have infrastructure layer imports")
	}
}
