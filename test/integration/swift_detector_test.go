package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/swift"
)

func TestSwiftDetector_PackageSwiftFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("swift-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Swift fixture not found at %s", fixturePath)
	}

	detector := swift.New()

	// Detect
	isSwift, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isSwift {
		t.Error("Expected to detect Swift project with Package.swift")
	}

	// Extract imports
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
		{Name: "application", Paths: []string{"application/**"}},
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

	// Should skip test files (*Tests.swift)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderTests.swift" {
			t.Error("Should skip test files (*Tests.swift)")
		}
	}

	// Should skip system frameworks (Foundation)
	for _, dep := range deps {
		if dep.ImportPath == "Foundation" {
			t.Error("Should skip 'Foundation' as system framework")
		}
	}
}

func TestSwiftDetector_SkipsTestsDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("swift-project")

	detector := swift.New()
	files, err := detector.FindSwiftFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindSwiftFiles() error = %v", err)
	}

	// Should not find files in Tests/ directory
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "Tests" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "Tests" {
			t.Errorf("Should skip Tests/ directory, found: %s", f)
		}
	}
}

func TestSwiftDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("swift-project")

	detector := swift.New()

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
		{Name: "application", Paths: []string{"application/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	for _, dep := range deps {
		t.Logf("  Dependency: source=%s import=%s layer=%s",
			filepath.Base(dep.SourceFile), dep.ImportPath, dep.ResolvedLayer)
	}

	// Check that OrderService.swift (application) correctly resolves domain imports
	appDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderService.swift" && dep.ResolvedLayer == "domain" {
			appDomainImports = true
			break
		}
	}
	if !appDomainImports {
		t.Error("Expected OrderService.swift (application) to have domain layer imports")
	}

	// Check that OrderRepository.swift (infrastructure) correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "OrderRepository.swift" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected OrderRepository.swift (infrastructure) to have domain layer imports")
	}
}
