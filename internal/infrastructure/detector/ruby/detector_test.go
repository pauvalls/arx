package ruby

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestRubyDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "ruby" {
		t.Errorf("Expected name 'ruby', got %q", name)
	}
}

func TestRubyDetector_Detect_Gemfile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create Gemfile
	content := `source 'https://rubygems.org'

gem 'rails', '~> 7.0'
gem 'pg'
gem 'puma'`
	os.WriteFile(filepath.Join(tmpDir, "Gemfile"), []byte(content), 0644)

	detector := New()
	isRuby, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isRuby {
		t.Error("Expected to detect Ruby project with Gemfile")
	}
}

func TestRubyDetector_Detect_NotRubyProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isRuby, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isRuby {
		t.Error("Expected to not detect Ruby project")
	}
}

func TestRubyDetector_ShouldSkip_Vendor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"vendor base", "vendor", true},
		{"vendor with path", "/project/vendor", true},
		{"vendor nested path", "/project/vendor/bundle", true},
		{"vendor with suffix", "vendorize", false},
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

func TestRubyDetector_ShouldSkip_Bundle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"bundle base", "bundle", true},
		{"bundle with path", "/project/bundle", true},
		{"bundle nested path", "/project/bundle/ruby", true},
		{"bundle with suffix", "bundleup", false},
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

func TestRubyDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"lib/app/service.rb", "/project/lib/app/service.rb", false},
		{"lib/domain/order.rb", "/project/lib/domain/order.rb", false},
		{"app/models/user.rb", "/project/app/models/user.rb", false},
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

func TestRubyDetector_SkipsVendorBundle(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .rb file in vendor/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "vendor", "bundle", "ruby"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "vendor", "bundle", "ruby", "gem.rb"), []byte("class Gem; end"), 0644)

	// Create .rb file in bundle/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "bundle", "ruby"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "bundle", "ruby", "cached.rb"), []byte("# cached"), 0644)

	// Create valid .rb file in lib/
	os.MkdirAll(filepath.Join(tmpDir, "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "lib", "app.rb"), []byte("class App; end"), 0644)

	detector := New()
	files, err := detector.FindRubyFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRubyFiles() error = %v", err)
	}

	// Should only find the file in lib/, not in vendor/ or bundle/
	if len(files) != 1 {
		t.Errorf("Expected 1 Ruby file (vendor/ and bundle/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "app.rb") {
		t.Errorf("Expected file in lib/, got %q", files[0])
	}
}

func TestRubyDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test Ruby file
	os.MkdirAll(filepath.Join(tmpDir, "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "lib", "app.rb"), []byte("class App; end"), 0644)

	// Create spec file (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "spec"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "spec", "app_spec.rb"), []byte("RSpec.describe App; end"), 0644)

	// Create test file (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "test"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "test", "app_test.rb"), []byte("class AppTest; end"), 0644)

	detector := New()
	files, err := detector.FindRubyFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRubyFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 Ruby file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestRubyDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "lib", "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "lib", "infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "lib", "application"), 0755)

	// Create Ruby file with various require types
	rbContent := `require 'rails'
require 'sinatra'
require_relative '../domain/order'
require_relative '../infrastructure/database'
require File.expand_path('../helpers', __dir__)

class OrderService
  def initialize
    @order = Order.new
  end
end`
	os.WriteFile(filepath.Join(tmpDir, "lib", "application", "order_service.rb"), []byte(rbContent), 0644)

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

	// Should find local imports (excluding external gems like 'rails', 'sinatra')
	if len(deps) < 1 {
		t.Errorf("Expected at least 1 dependency (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if dep.ImportPath == "../domain/order" && dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find ../domain/order resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestRubyDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "lib", "app.rb"), []byte("class App; end"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestRubyDetector_ExtractImports_SkipExternal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "lib", "application"), 0755)

	rbContent := `require 'rails'
require 'sinatra/base'
require 'bundler/setup'
require 'rubygems'
require_relative '../domain/order'

class OrderService; end`
	os.WriteFile(filepath.Join(tmpDir, "lib", "application", "service.rb"), []byte(rbContent), 0644)

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

	// Should only find require_relative imports, NOT bare require (gems)
	for _, dep := range deps {
		if dep.ImportPath == "rails" {
			t.Error("Expected 'rails' to be skipped as external dependency")
		}
		if dep.ImportPath == "sinatra/base" {
			t.Error("Expected 'sinatra/base' to be skipped as external dependency")
		}
		if dep.ImportPath == "bundler/setup" {
			t.Error("Expected 'bundler/setup' to be skipped as external dependency")
		}
		if dep.ImportPath == "rubygems" {
			t.Error("Expected 'rubygems' to be skipped as external dependency")
		}
	}
}

func TestRubyDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested vendor/bundle in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", "vendor"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "vendor", "gem.rb"), []byte("class Gem; end"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "bundle"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "bundle", "cached.rb"), []byte("# cached"), 0644)

	// Create valid Ruby files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "lib", "app.rb"), []byte("class App; end"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "lib", "app.rb"), []byte("class App; end"), 0644)

	detector := New()
	files, err := detector.FindRubyFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRubyFiles() error = %v", err)
	}

	// Should find 2 files (module1/lib/app.rb and module2/lib/app.rb), skip vendor/ and bundle/
	if len(files) != 2 {
		t.Errorf("Expected 2 Ruby files (nested vendor/ and bundle/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
