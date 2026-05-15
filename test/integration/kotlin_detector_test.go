package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/kotlin"
)

func TestKotlinDetector_GradleKtsFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("kotlin-gradle")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Kotlin Gradle fixture not found at %s", fixturePath)
	}

	detector := kotlin.New()

	// Detect
	isKotlin, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isKotlin {
		t.Error("Expected to detect Kotlin project with build.gradle.kts")
	}

	// Extract imports
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"com/example/domain/**"}},
		{Name: "infrastructure", Paths: []string{"com/example/infrastructure/**"}},
		{Name: "application", Paths: []string{"com/example/app/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find domain and infrastructure layer imports
	foundDomain := false
	foundInfrastructure := false
	for _, dep := range deps {
		if dep.ResolvedLayer == "domain" {
			foundDomain = true
		}
		if dep.ResolvedLayer == "infrastructure" {
			foundInfrastructure = true
		}
	}

	if !foundDomain {
		t.Error("Expected to find domain layer imports")
	}
	if !foundInfrastructure {
		t.Error("Expected to find infrastructure layer imports")
	}

	// Should skip test files (*Test.kt)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "EntityTest.kt" {
			t.Error("Should skip test files (*Test.kt)")
		}
	}
}

func TestKotlinDetector_SkipsBuildDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("kotlin-gradle")

	detector := kotlin.New()
	files, err := detector.FindKotlinFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindKotlinFiles() error = %v", err)
	}

	// Should not find files in build/ directory
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "build" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "build" {
			t.Errorf("Should skip build/ directory, found: %s", f)
		}
	}
}

func TestKotlinDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("kotlin-gradle")

	detector := kotlin.New()

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"com/example/domain/**"}},
		{Name: "infrastructure", Paths: []string{"com/example/infrastructure/**"}},
		{Name: "application", Paths: []string{"com/example/app/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Service.kt (application) imports Database (infrastructure) and Entity (domain as alias)
	// Database.kt (infrastructure) imports Entity (domain)
	// Entity.kt (domain) imports Service (application) -- this is a violation!

	for _, dep := range deps {
		t.Logf("  Dependency: source=%s import=%s layer=%s",
			filepath.Base(dep.SourceFile), dep.ImportPath, dep.ResolvedLayer)
	}

	// Check that application layer correctly resolves domain imports
	serviceDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "Service.kt" && dep.ResolvedLayer == "domain" {
			serviceDomainImports = true
			break
		}
	}
	if !serviceDomainImports {
		t.Error("Expected Service.kt (application) to have domain layer imports")
	}

	// Check that infrastructure correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "Database.kt" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected Database.kt (infrastructure) to have domain layer imports")
	}
}
