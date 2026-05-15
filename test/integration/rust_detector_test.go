package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/detector/rust"
)

func TestRustDetector_CargoFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("rust-cargo")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Rust Cargo fixture not found at %s", fixturePath)
	}

	detector := rust.New()

	// Detect
	isRust, err := detector.Detect(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isRust {
		t.Error("Expected to detect Rust project with Cargo.toml")
	}

	// Extract imports
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
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

	// Should skip test files (*_test.rs)
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "lib_test.rs" {
			t.Error("Should skip test files (*_test.rs)")
		}
	}

	// Should skip external dependencies (std::)
	for _, dep := range deps {
		if dep.ImportPath == "std::collections::HashMap" {
			t.Error("Should skip std::collections::HashMap as external dependency")
		}
	}
}

func TestRustDetector_SkipsTargetDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("rust-cargo")

	detector := rust.New()
	files, err := detector.FindRustFiles(fixturePath)
	if err != nil {
		t.Fatalf("FindRustFiles() error = %v", err)
	}

	// Should not find files in target/ directory
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "target" || filepath.Base(filepath.Dir(filepath.Dir(f))) == "target" {
			t.Errorf("Should skip target/ directory, found: %s", f)
		}
	}
}

func TestRustDetector_LayerResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fixturePath := getFixturePath("rust-cargo")

	detector := rust.New()

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
	}

	deps, err := detector.ExtractImports(context.Background(), fixturePath, layers)
	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	for _, dep := range deps {
		t.Logf("  Dependency: source=%s import=%s layer=%s",
			filepath.Base(dep.SourceFile), dep.ImportPath, dep.ResolvedLayer)
	}

	// Check that repository.rs correctly resolves domain imports
	infraDomainImports := false
	for _, dep := range deps {
		if filepath.Base(dep.SourceFile) == "repository.rs" && dep.ResolvedLayer == "domain" {
			infraDomainImports = true
			break
		}
	}
	if !infraDomainImports {
		t.Error("Expected repository.rs (infrastructure) to have domain layer imports")
	}
}
