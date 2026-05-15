package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/java"
)

func TestJavaDetector_MavenFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use the Maven fixture
	fixturePath := getFixturePath("java-maven")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Maven fixture not found at %s", fixturePath)
	}

	detector := java.New()

	// Detect
	isJava, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isJava {
		t.Error("Expected to detect Maven Java project")
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

	// Should find domain imports
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

	// Should skip test files (*Test.java)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderServiceTest.java" {
			t.Error("Should skip test files (*Test.java)")
		}
	}
}

func TestJavaDetector_GradleFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("java-gradle")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Gradle fixture not found at %s", fixturePath)
	}

	detector := java.New()

	isJava, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isJava {
		t.Error("Expected to detect Gradle Java project")
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"com/example/domain/**"}},
		{Name: "infrastructure", Paths: []string{"com/example/infrastructure/**"}},
		{Name: "application", Paths: []string{"com/example/app/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	if len(deps) == 0 {
		t.Error("Expected to extract dependencies from Gradle fixture")
	}
}

func TestJavaDetector_SkipsTargetAndBuildDirs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("java-maven")

	detector := java.New()
	files, err := detector.FindJavaFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindJavaFiles() error = %v", err)
	}

	// Should not find files in target/ directory
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "target" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "target" {
			t.Errorf("Should skip target/ directory, found: %s", f)
		}
	}
}

func getFixturePath(fixtureName string) string {
	// Integration tests run from test/integration/, fixtures are at test/fixtures/
	abs, _ := filepath.Abs(filepath.Join("..", "fixtures", fixtureName))
	return abs
}
