package php

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestPHPDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "php" {
		t.Errorf("Expected name 'php', got %q", name)
	}
}

func TestPHPDetector_Detect_ComposerJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create composer.json
	content := `{
    "name": "test/project",
    "require": {
        "php": ">=8.0"
    },
    "autoload": {
        "psr-4": {
            "App\\": "src/"
        }
    }
}`
	os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(content), 0644)

	detector := New()
	isPHP, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isPHP {
		t.Error("Expected to detect PHP project with composer.json")
	}
}

func TestPHPDetector_Detect_NotPHPProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isPHP, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isPHP {
		t.Error("Expected to not detect PHP project")
	}
}

func TestPHPDetector_ShouldSkip_Vendor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"vendor base", "vendor", true},
		{"vendor with path", "/project/vendor", true},
		{"vendor nested path", "/project/vendor/symfony", true},
		{"vendor with suffix", "vendored", false},
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

func TestPHPDetector_ShouldSkip_Tests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"tests base", "tests", true},
		{"tests with path", "/project/tests", true},
		{"tests nested path", "/project/tests/Unit", true},
		{"test with suffix", "testify", false},
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

func TestPHPDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"src/domain/service.php", "/project/src/domain/service.php", false},
		{"src/application/order.php", "/project/src/application/order.php", false},
		{"app/models/user.php", "/project/app/models/user.php", false},
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

func TestPHPDetector_SkipsVendor(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .php file in vendor/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "vendor", "symfony", "http-foundation"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "vendor", "symfony", "http-foundation", "Request.php"), []byte("<?php namespace Symfony;"), 0644)

	// Create valid .php file in src/
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "Order.php"), []byte("<?php namespace App;"), 0644)

	detector := New()
	files, err := detector.FindPHPFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindPHPFiles() error = %v", err)
	}

	// Should only find the file in src/, not in vendor/
	if len(files) != 1 {
		t.Errorf("Expected 1 PHP file (vendor/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "Order.php") {
		t.Errorf("Expected file in src/, got %q", files[0])
	}
}

func TestPHPDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test PHP file
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "Order.php"), []byte("<?php namespace App;"), 0644)

	// Create test file (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "tests"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "tests", "OrderTest.php"), []byte("<?php class OrderTest {}"), 0644)

	detector := New()
	files, err := detector.FindPHPFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindPHPFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 PHP file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestPHPDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "Domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "Infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "Application"), 0755)

	// Create PHP file with various use/require types
	phpContent := `<?php

use App\Domain\Order;
use App\Infrastructure\OrderRepository;
use Symfony\Component\HttpFoundation\Request;
require_once __DIR__ . '/../Domain/Order.php';

class OrderService
{
    public function process(): void
    {
        $order = new Order();
    }
}`
	os.WriteFile(filepath.Join(tmpDir, "src", "Application", "OrderService.php"), []byte(phpContent), 0644)

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

	// Should find local imports (excluding external like Symfony)
	if len(deps) < 1 {
		t.Errorf("Expected at least 1 dependency (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find domain layer import")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestPHPDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "app.php"), []byte("<?php"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestPHPDetector_ExtractImports_SkipExternal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "src", "Application"), 0755)

	phpContent := `<?php

use Symfony\Component\HttpFoundation\Request;
use Doctrine\ORM\EntityManager;
use Psr\Log\LoggerInterface;
use App\Domain\Order;

class OrderService {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "Application", "Service.php"), []byte(phpContent), 0644)

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

	// Should only find App\ imports, NOT external packages
	for _, dep := range deps {
		if strings.HasPrefix(dep.ImportPath, "Symfony\\") {
			t.Error("Expected 'Symfony\\...' to be skipped as external dependency")
		}
		if strings.HasPrefix(dep.ImportPath, "Doctrine\\") {
			t.Error("Expected 'Doctrine\\...' to be skipped as external dependency")
		}
		if strings.HasPrefix(dep.ImportPath, "Psr\\") {
			t.Error("Expected 'Psr\\...' to be skipped as external dependency")
		}
	}
}

func TestPHPDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested vendor in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", "vendor"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "vendor", "Package.php"), []byte("<?php"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "tests"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "tests", "Test.php"), []byte("<?php"), 0644)

	// Create valid PHP files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "src", "App.php"), []byte("<?php"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "src", "App.php"), []byte("<?php"), 0644)

	detector := New()
	files, err := detector.FindPHPFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindPHPFiles() error = %v", err)
	}

	// Should find 2 files (module1/src/App.php and module2/src/App.php), skip vendor/ and tests/
	if len(files) != 2 {
		t.Errorf("Expected 2 PHP files (nested vendor/ and tests/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
