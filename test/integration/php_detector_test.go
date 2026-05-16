package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/php"
)

func TestPHPDetector_ComposerJSONFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("php-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("PHP fixture not found at %s", fixturePath)
	}

	detector := php.New()

	// Detect
	isPHP, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isPHP {
		t.Error("Expected to detect PHP project with composer.json")
	}

	// Extract imports
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"App/Domain/**"}},
		{Name: "infrastructure", Paths: []string{"App/Infrastructure/**"}},
		{Name: "application", Paths: []string{"App/Application/**"}},
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

	// Should skip test files (*Test.php)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderTest.php" {
			t.Error("Should skip test files (*Test.php)")
		}
	}

	// Should skip external dependencies (Symfony)
	for _, dep := range deps {
		if dep.ImportPath == "Symfony\\Component\\HttpFoundation\\Request" {
			t.Error("Should skip 'Symfony\\...' as external dependency")
		}
	}
}

func TestPHPDetector_SkipsVendorDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("php-project")

	detector := php.New()
	files, err := detector.FindPHPFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindPHPFiles() error = %v", err)
	}

	// Should not find files in vendor/ directory
	for _, f := range files {
		rel, _ := filepath.Rel(fixturePath, f)
		if filepath.HasPrefix(rel, "vendor") {
			t.Errorf("Should skip vendor/ directory, found: %s", f)
		}
	}
}

func TestPHPDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("php-project")

	detector := php.New()

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"App/Domain/**"}},
		{Name: "infrastructure", Paths: []string{"App/Infrastructure/**"}},
		{Name: "application", Paths: []string{"App/Application/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	for _, dep := range deps {
		t.Logf("  Dependency: source=%s import=%s layer=%s",
			filepath.Base(dep.SourceFile), dep.ImportPath, dep.ResolvedLayer)
	}

	// Check that OrderService.php (application) correctly resolves domain imports
	appDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderService.php" && dep.ResolvedLayer == "domain" {
			appDomainImports = true
			break
		}
	}
	if !appDomainImports {
		t.Error("Expected OrderService.php (application) to have domain layer imports")
	}

	// Check that OrderRepository.php (infrastructure) correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderRepository.php" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected OrderRepository.php (infrastructure) to have domain layer imports")
	}
}
