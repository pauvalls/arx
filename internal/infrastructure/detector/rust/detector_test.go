package rust

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestRustDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "rust" {
		t.Errorf("Expected name 'rust', got %q", name)
	}
}

func TestRustDetector_Detect_CargoToml(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create Cargo.toml
	content := `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = "1.0"`
	os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(content), 0644)

	detector := New()
	isRust, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isRust {
		t.Error("Expected to detect Rust project with Cargo.toml")
	}
}

func TestRustDetector_Detect_NotRustProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isRust, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isRust {
		t.Error("Expected to not detect Rust project")
	}
}

func TestRustDetector_ShouldSkip_Target(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"target base", "target", true},
		{"target with path", "/project/target", true},
		{"target nested path", "/project/target/debug", true},
		{"target with suffix", "target-gen", false},
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

func TestRustDetector_ShouldSkip_Build(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"build base", "build", true},
		{"build with path", "/project/build", true},
		{"build nested path", "/project/build/scripts", true},
		{"build with suffix", "buildkit", false},
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

func TestRustDetector_NoSkip_ValidPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"src/main.rs", "/project/src/main.rs", false},
		{"src/lib.rs", "/project/src/lib.rs", false},
		{"src/domain/mod.rs", "/project/src/domain/mod.rs", false},
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

func TestRustDetector_SkipsTargetBuild(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .rs file in target/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "target", "debug"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "target", "debug", "app.rs"), []byte("fn main() {}"), 0644)

	// Create .rs file in build/ directory (should be skipped)
	os.MkdirAll(filepath.Join(tmpDir, "build", "scripts"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "build", "scripts", "build.rs"), []byte("fn main() {}"), 0644)

	// Create valid .rs file in src/
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "lib.rs"), []byte("pub fn hello() {}"), 0644)

	detector := New()
	files, err := detector.FindRustFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRustFiles() error = %v", err)
	}

	// Should only find the file in src/, not in target/ or build/
	if len(files) != 1 {
		t.Errorf("Expected 1 Rust file (target/ and build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
	if !strings.HasSuffix(files[0], "lib.rs") {
		t.Errorf("Expected file in src/, got %q", files[0])
	}
}

func TestRustDetector_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create non-test Rust file
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "lib.rs"), []byte("pub fn hello() {}"), 0644)

	// Create test file (should be skipped)
	os.WriteFile(filepath.Join(tmpDir, "src", "lib_test.rs"), []byte("#[test] fn test_hello() {}"), 0644)

	// Create another test file
	os.WriteFile(filepath.Join(tmpDir, "src", "helpers_test.rs"), []byte("#[test] fn test_helper() {}"), 0644)

	detector := New()
	files, err := detector.FindRustFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRustFiles() error = %v", err)
	}

	// Should only find the non-test file
	if len(files) != 1 {
		t.Errorf("Expected 1 Rust file (test files skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}

func TestRustDetector_ExtractImports_Valid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project structure
	os.MkdirAll(filepath.Join(tmpDir, "src", "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "infrastructure"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src", "app"), 0755)

	// Create Rust file with various use types
	rsContent := `use std::collections::HashMap;
use crate::domain::model::Order;
use crate::infrastructure::Database;
use self::submodule::Helper;
use super::parent_module::Something;
pub use crate::domain::Model;

pub mod submodule;

fn do_something() {
    let map = HashMap::new();
}`
	os.WriteFile(filepath.Join(tmpDir, "src", "app", "service.rs"), []byte(rsContent), 0644)

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
			Paths: []string{"app/**"},
		},
	}

	detector := New()
	deps, err := detector.ExtractImports(context.Background(), tmpDir, layers)

	if err != nil {
		t.Fatalf("ExtractImports() error = %v", err)
	}

	// Should find imports (excluding std::)
	// Expected: crate::domain::model::Order, crate::infrastructure::Database, self::submodule::Helper,
	//           super::parent_module::Something, crate::domain::Model (pub use)
	if len(deps) < 3 {
		t.Errorf("Expected at least 3 dependencies (non-external), got %d", len(deps))
		for _, dep := range deps {
			t.Logf("  Found: %v", dep)
		}
	}

	// Check that we found the domain import
	foundDomain := false
	for _, dep := range deps {
		if dep.ImportPath == "crate::domain::model::Order" && dep.ResolvedLayer == "domain" {
			foundDomain = true
			break
		}
	}
	if !foundDomain {
		t.Error("Expected to find crate::domain::model::Order resolved to 'domain' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}

	// Check that we found the infrastructure import
	foundInfra := false
	for _, dep := range deps {
		if dep.ImportPath == "crate::infrastructure::Database" && dep.ResolvedLayer == "infrastructure" {
			foundInfra = true
			break
		}
	}
	if !foundInfra {
		t.Error("Expected to find crate::infrastructure::Database resolved to 'infrastructure' layer")
		for _, dep := range deps {
			t.Logf("  Dependency: %+v", dep)
		}
	}
}

func TestRustDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "lib.rs"), []byte("fn main() {}"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestRustDetector_ExtractImports_SkipExternal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "src", "app"), 0755)

	rsContent := `use std::collections::HashMap;
use core::mem::MaybeUninit;
use alloc::sync::Arc;
use test::Bencher;
use crate::domain::Order;

fn do_something() {}`
	os.WriteFile(filepath.Join(tmpDir, "src", "app", "service.rs"), []byte(rsContent), 0644)

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

	// Should only find crate::domain::Order, NOT std::*, core::*, alloc::*, test::*
	for _, dep := range deps {
		if dep.ImportPath == "std::collections::HashMap" {
			t.Error("Expected std::collections::HashMap to be skipped as external dependency")
		}
		if dep.ImportPath == "core::mem::MaybeUninit" {
			t.Error("Expected core::mem::MaybeUninit to be skipped as external dependency")
		}
		if dep.ImportPath == "alloc::sync::Arc" {
			t.Error("Expected alloc::sync::Arc to be skipped as external dependency")
		}
		if dep.ImportPath == "test::Bencher" {
			t.Error("Expected test::Bencher to be skipped as external dependency")
		}
	}

	if len(deps) != 1 {
		t.Errorf("Expected exactly 1 dependency (domain), got %d", len(deps))
	}
}

func TestRustDetector_ShouldSkip_Nested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create nested target/build in submodules
	os.MkdirAll(filepath.Join(tmpDir, "module1", "target"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "target", "app.rs"), []byte("fn main() {}"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "build"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "build", "app.rs"), []byte("fn main() {}"), 0644)

	// Create valid Rust files
	os.MkdirAll(filepath.Join(tmpDir, "module1", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module1", "src", "lib.rs"), []byte("fn main() {}"), 0644)

	os.MkdirAll(filepath.Join(tmpDir, "module2", "src"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "module2", "src", "lib.rs"), []byte("fn main() {}"), 0644)

	detector := New()
	files, err := detector.FindRustFiles(tmpDir)

	if err != nil {
		t.Fatalf("FindRustFiles() error = %v", err)
	}

	// Should find 2 files (module1/src/lib.rs and module2/src/lib.rs), skip target/ and build/
	if len(files) != 2 {
		t.Errorf("Expected 2 Rust files (nested target/ and build/ skipped), got %d", len(files))
		for _, f := range files {
			t.Logf("  Found: %s", f)
		}
	}
}
