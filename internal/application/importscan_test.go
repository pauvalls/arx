package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestScanImports_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary == nil {
		t.Fatal("ScanImports() returned nil summary")
	}
	if summary.FilesScanned != 0 {
		t.Errorf("expected 0 files scanned, got %d", summary.FilesScanned)
	}
}

func TestScanImports_GoProject(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "internal", "application")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	goFile := `package application

import (
	"fmt"
	"github.com/pauvalls/arx/internal/domain"
)

func Hello() string {
	return domain.SomeFunc()
}
`
	if err := os.WriteFile(filepath.Join(appDir, "service.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "application", Paths: []string{"internal/application/**"}},
		{Name: "domain", Paths: []string{"internal/domain/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.FilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", summary.FilesScanned)
	}
	if summary.ImportsFound == 0 {
		t.Error("expected imports to be found")
	}
}

func TestScanImports_SingleImport(t *testing.T) {
	tmpDir := t.TempDir()
	domainDir := filepath.Join(tmpDir, "internal", "domain")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatal(err)
	}

	goFile := `package domain

import "fmt"

func Hello() string {
	return fmt.Sprintf("hello")
}
`
	if err := os.WriteFile(filepath.Join(domainDir, "entity.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.FilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", summary.FilesScanned)
	}
}

func TestScanImports_SkipVendor(t *testing.T) {
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor", "somepkg")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("package lib"), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.FilesScanned != 0 {
		t.Errorf("expected 0 files scanned for vendor, got %d", summary.FilesScanned)
	}
}

func TestScanImports_TypeScriptProject(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "src", "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	tsFile := `import { Component } from '@angular/core';
import * as _ from 'lodash';
const express = require('express');
import 'reflect-metadata';
`
	if err := os.WriteFile(filepath.Join(appDir, "main.ts"), []byte(tsFile), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "app", Paths: []string{"src/app/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.FilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", summary.FilesScanned)
	}
	if summary.ImportsFound != 4 {
		t.Errorf("expected 4 imports in TS file, got %d", summary.ImportsFound)
	}
}

func TestScanImports_MultiLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	// Go file
	goDir := filepath.Join(tmpDir, "goapp")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "main.go"), []byte("package main\n\nimport \"fmt\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// TS file
	tsDir := filepath.Join(tmpDir, "tsapp")
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsDir, "app.ts"), []byte("import { foo } from './foo';\n"), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "app", Paths: []string{"**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.FilesScanned != 2 {
		t.Errorf("expected 2 files scanned, got %d", summary.FilesScanned)
	}
}

func TestScanImports_GoBlockImport(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}

	goFile := `package pkg

import (
	"fmt"
	"os"
	"strings"
)

func Foo() {}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pkg", "foo.go"), []byte(goFile), 0644); err != nil {
		t.Fatal(err)
	}

	layers := []domain.Layer{
		{Name: "pkg", Paths: []string{"pkg/**"}},
	}

	summary, err := ScanImports(tmpDir, layers)
	if err != nil {
		t.Fatalf("ScanImports() error = %v", err)
	}
	if summary.ImportsFound != 3 {
		t.Errorf("expected 3 imports, got %d", summary.ImportsFound)
	}
}

func TestIsSkippedDir(t *testing.T) {
	tests := []struct {
		path     string
		skipped  bool
	}{
		{"vendor/somepkg/file.go", true},
		{".git/config", true},
		{"node_modules/express/index.js", true},
		{"dist/bundle.js", true},
		{"build/output.o", true},
		{"target/debug/lib.rs", true},
		{"internal/domain/user.go", false},
		{"cmd/main.go", false},
		{"pkg/util/helper.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSkippedDir(tt.path)
			if got != tt.skipped {
				t.Errorf("isSkippedDir(%q) = %v, want %v", tt.path, got, tt.skipped)
			}
		})
	}
}

func TestBuildLayerResolver(t *testing.T) {
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
		{Name: "application", Paths: []string{"internal/application/**"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
	}

	resolver := buildLayerResolver("/test", layers)
	if resolver == nil {
		t.Fatal("buildLayerResolver returned nil")
	}

	tests := []struct {
		path  string
		layer string
	}{
		{"internal/domain/user.go", "domain"},
		{"internal/domain/entity/order.go", "domain"},
		{"internal/application/service.go", "application"},
		{"internal/infrastructure/db/repo.go", "infrastructure"},
		{"cmd/main.go", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := resolver(tt.path)
			if got != tt.layer {
				t.Errorf("buildLayerResolver(%q) = %q, want %q", tt.path, got, tt.layer)
			}
		})
	}
}

func TestResolveImportLayer(t *testing.T) {
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
		{Name: "application", Paths: []string{"internal/application/**"}},
	}

	tests := []struct {
		name       string
		importPath string
		layer      string
	}{
		{"domain import", "github.com/pkg/internal/domain/user", "domain"},
		{"application import", "github.com/pkg/internal/application/service", "application"},
		{"infra import", "github.com/pkg/internal/infrastructure/db", "external"},
		{"stdlib import", "fmt", "external"},
		{"empty import", "", "external"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveImportLayer(tt.importPath, layers)
			if got != tt.layer {
				t.Errorf("resolveImportLayer(%q) = %q, want %q", tt.importPath, got, tt.layer)
			}
		})
	}
}

func TestImportSummary_ShortStats(t *testing.T) {
	t.Run("nil summary returns empty", func(t *testing.T) {
		var s *ImportSummary
		if got := s.ShortStats(); got != "" {
			t.Errorf("nil ShortStats = %q, want empty", got)
		}
	})

	t.Run("empty summary returns empty", func(t *testing.T) {
		s := &ImportSummary{}
		if got := s.ShortStats(); got != "" {
			t.Errorf("empty ShortStats = %q, want empty", got)
		}
	})

	t.Run("with data returns formatted string", func(t *testing.T) {
		s := &ImportSummary{
			ImportsFound: 42,
			FilesScanned: 10,
			ByLayer: map[string]map[string]int{
				"domain": {"application": 5},
			},
		}
		got := s.ShortStats()
		if !strings.Contains(got, "42") || !strings.Contains(got, "10") {
			t.Errorf("ShortStats = %q, want stats with 42 and 10", got)
		}
	})
}

func TestImportSummary_FormatSummary(t *testing.T) {
	t.Run("nil summary returns no deps", func(t *testing.T) {
		var s *ImportSummary
		got := s.FormatSummary()
		if !strings.Contains(got, "No dependencies") {
			t.Errorf("nil FormatSummary = %q, want 'No dependencies'", got)
		}
	})

	t.Run("empty byLayer returns no deps", func(t *testing.T) {
		s := &ImportSummary{}
		got := s.FormatSummary()
		if !strings.Contains(got, "No dependencies") {
			t.Errorf("empty FormatSummary = %q, want 'No dependencies'", got)
		}
	})

	t.Run("with dependencies shows formatted output", func(t *testing.T) {
		s := &ImportSummary{
			FilesScanned: 5,
			ImportsFound: 10,
			ByLayer: map[string]map[string]int{
				"domain":       {"application": 3, "infrastructure": 2},
				"application": {"domain": 5},
			},
		}
		got := s.FormatSummary()
		if !strings.Contains(got, "Files scanned: 5") {
			t.Errorf("FormatSummary missing file count: %s", got)
		}
		if !strings.Contains(got, "Imports found: 10") {
			t.Errorf("FormatSummary missing import count: %s", got)
		}
		if !strings.Contains(got, "domain") || !strings.Contains(got, "application") {
			t.Errorf("FormatSummary missing layer names: %s", got)
		}
	})
}

func TestScanGoImports_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

import "fmt"

func main() {}
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanGoImports(file, "app")
	if err != nil {
		t.Fatalf("scanGoImports() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 import, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Import != "fmt" {
		t.Errorf("expected import 'fmt', got %q", entries[0].Import)
	}
}

func TestScanGoImports_BlockImport(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {}
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanGoImports(file, "app")
	if err != nil {
		t.Fatalf("scanGoImports() error = %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 imports, got %d", len(entries))
	}
}

func TestScanGoImports_AliasedImport(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

import (
	"fmt"
	alias "os/exec"
)

func main() {}
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanGoImports(file, "app")
	if err != nil {
		t.Fatalf("scanGoImports() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 imports, got %d", len(entries))
	}
}

func TestScanGoImports_NoImport(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "main.go")
	content := []byte(`package main

func main() {}
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanGoImports(file, "app")
	if err != nil {
		t.Fatalf("scanGoImports() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 imports, got %d", len(entries))
	}
}

func TestScanGoImports_NonexistentFile(t *testing.T) {
	_, err := scanGoImports("/nonexistent/file.go", "app")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestScanTSImports_Variety(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "app.ts")
	content := []byte(`import { Component } from '@angular/core';
import * as _ from 'lodash';
const fs = require('fs');
import 'reflect-metadata';
import './local-module';
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanTSImports(file, "app")
	if err != nil {
		t.Fatalf("scanTSImports() error = %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 imports, got %d", len(entries))
	}
}

func TestScanTSImports_NonexistentFile(t *testing.T) {
	_, err := scanTSImports("/nonexistent/file.ts", "app")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestScanTSImports_NoImport(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty.ts")
	content := []byte(`const x = 42;
`)
	if err := os.WriteFile(file, content, 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := scanTSImports(file, "app")
	if err != nil {
		t.Fatalf("scanTSImports() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 imports, got %d", len(entries))
	}
}
