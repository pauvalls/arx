package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/ruby"
)

func TestRubyDetector_GemfileFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("ruby-project")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Ruby fixture not found at %s", fixturePath)
	}

	detector := ruby.New()

	// Detect
	isRuby, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isRuby {
		t.Error("Expected to detect Ruby project with Gemfile")
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

	// Should skip test files (*_spec.rb)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "order_spec.rb" {
			t.Error("Should skip spec files (*_spec.rb)")
		}
	}

	// Should skip external dependencies (gems like 'rails')
	for _, dep := range deps {
		if dep.ImportPath == "rails" {
			t.Error("Should skip 'rails' as external dependency")
		}
	}
}

func TestRubyDetector_SkipsVendorDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("ruby-project")

	detector := ruby.New()
	files, err := detector.FindRubyFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindRubyFiles() error = %v", err)
	}

	// Should not find files in vendor/ directory
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "vendor" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "vendor" {
			t.Errorf("Should skip vendor/ directory, found: %s", f)
		}
	}
}

func TestRubyDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("ruby-project")

	detector := ruby.New()

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

	// Check that order_service.rb (application) correctly resolves domain imports
	appDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "order_service.rb" && dep.ResolvedLayer == "domain" {
			appDomainImports = true
			break
		}
	}
	if !appDomainImports {
		t.Error("Expected order_service.rb (application) to have domain layer imports")
	}

	// Check that order_repo.rb (infrastructure) correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "order_repo.rb" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected order_repo.rb (infrastructure) to have domain layer imports")
	}
}
