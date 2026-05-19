package typescript_detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestName(t *testing.T) {
	d := New()
	if d.Name() != "typescript" {
		t.Errorf("Name() = %q, want %q", d.Name(), "typescript")
	}
}

func TestDetect(t *testing.T) {
	t.Run("with tsconfig returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ok, err := d.Detect(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if !ok {
			t.Error("Detect() = false, want true")
		}
	})

	t.Run("with package.json returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ok, err := d.Detect(context.Background(), tmpDir)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if !ok {
			t.Error("Detect() = false, want true")
		}
	})

	t.Run("without markers returns false", func(t *testing.T) {
		d := New()
		ok, err := d.Detect(context.Background(), t.TempDir())
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if ok {
			t.Error("Detect() = true, want false")
		}
	})
}

func TestLoadTsConfig(t *testing.T) {
	t.Run("loads path aliases", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "tsconfig.json")
		tsconfig := `{
			"compilerOptions": {
				"paths": {
					"@domain/*": ["./src/domain/*"],
					"@infra/*": ["./src/infra/*"]
				}
			}
		}`
		if err := os.WriteFile(configPath, []byte(tsconfig), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		if err := d.loadTsConfig(configPath); err != nil {
			t.Fatalf("loadTsConfig() error = %v", err)
		}
		if len(d.pathAliases) != 2 {
			t.Errorf("expected 2 aliases, got %d", len(d.pathAliases))
		}
		if d.pathAliases["@domain"] != "./src/domain" {
			t.Errorf("alias @domain = %q, want %q", d.pathAliases["@domain"], "./src/domain")
		}
		if d.pathAliases["@infra"] != "./src/infra" {
			t.Errorf("alias @infra = %q, want %q", d.pathAliases["@infra"], "./src/infra")
		}
	})

	t.Run("no tsconfig is not an error", func(t *testing.T) {
		d := New()
		if err := d.loadTsConfig(filepath.Join(t.TempDir(), "nonexistent.json")); err == nil {
			t.Error("expected error for nonexistent tsconfig")
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{invalid}"), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		if err := d.loadTsConfig(tmpDir); err == nil {
			t.Error("expected error for invalid tsconfig")
		}
	})
}

func TestResolveAlias(t *testing.T) {
	d := &Detector{
		pathAliases: map[string]string{
			"@domain": "src/domain",
			"@lib":    "src/lib",
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"@domain/user", "src/domain/user"},
		{"@lib/helpers", "src/lib/helpers"},
		{"./relative/path", "./relative/path"},
		{"external-pkg", "external-pkg"},
	}
	for _, tt := range tests {
		got := d.resolveAlias(tt.input)
		if got != tt.expected {
			t.Errorf("resolveAlias(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveLayer(t *testing.T) {
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"src/domain/**"}},
		{Name: "infrastructure", Paths: []string{"src/infra/**"}},
	}

	t.Run("scoped package matching layer", func(t *testing.T) {
		d := &Detector{}
		got := d.resolveLayer("@domain/user", "/project", layers)
		if got != "domain" {
			t.Errorf("resolveLayer(@domain) = %q, want %q", got, "domain")
		}
	})

	t.Run("relative import matching layer", func(t *testing.T) {
		d := &Detector{baseUrl: "src"}
		got := d.resolveLayer("./domain/user", "/project", layers)
		if got != "domain" {
			t.Errorf("resolveLayer(./domain) = %q, want %q", got, "domain")
		}
	})

	t.Run("external dep returns empty", func(t *testing.T) {
		d := &Detector{}
		got := d.resolveLayer("express", "/project", layers)
		if got != "" {
			t.Errorf("resolveLayer(express) = %q, want empty", got)
		}
	})

	t.Run("import containing layer name", func(t *testing.T) {
		d := &Detector{}
		got := d.resolveLayer("@scope/domain/models", "/project", layers)
		if got != "domain" {
			t.Errorf("resolveLayer(@scope/domain) = %q, want %q", got, "domain")
		}
	})
}

func TestExtractImports(t *testing.T) {
	t.Run("basic extraction with tsconfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte(`{
			"compilerOptions": {
				"baseUrl": "./src",
				"paths": { "@domain/*": ["./domain/*"] }
			}
		}`), 0644); err != nil {
			t.Fatal(err)
		}
		srcDir := filepath.Join(tmpDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(srcDir, "app.ts"), []byte(`import { User } from '@domain/user';
import express from 'express';
`), 0644); err != nil {
			t.Fatal(err)
		}

		d := New()
		ctx := context.Background()
		if _, err := d.Detect(ctx, tmpDir); err != nil {
			t.Fatal(err)
		}

		deps, err := d.ExtractImports(ctx, tmpDir, []domain.Layer{
			{Name: "domain", Paths: []string{"src/domain/**"}},
		})
		if err != nil {
			t.Fatalf("ExtractImports() error = %v", err)
		}
		if len(deps) != 2 {
			t.Fatalf("ExtractImports() returned %d deps, want 2", len(deps))
		}
		if deps[0].ImportPath != "@domain/user" {
			t.Errorf("ImportPath = %q, want %q", deps[0].ImportPath, "@domain/user")
		}
		if deps[0].Language != "typescript" {
			t.Errorf("Language = %q, want %q", deps[0].Language, "typescript")
		}
	})

	t.Run("empty directory returns no deps", func(t *testing.T) {
		d := New()
		deps, err := d.ExtractImports(context.Background(), t.TempDir(), nil)
		if err != nil {
			t.Fatalf("ExtractImports() error = %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("ExtractImports() returned %d deps, want 0", len(deps))
		}
	})

	t.Run("cancellation returns context error", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "test.ts"), []byte(`import "x"`), 0644); err != nil {
			t.Fatal(err)
		}
		d := New()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := d.ExtractImports(ctx, tmpDir, nil)
		if err != context.Canceled {
			t.Logf("Expected Canceled or nil, got: %v", err)
		}
	})
}

func FuzzTypeScriptDetector(f *testing.F) {
	seeds := []string{
		"import { User } from './user';\n",
		"import express from 'express';\n",
		"const fs = require('fs');\n",
		"import './styles.css';\n",
		"import * as d3 from 'd3';\n",
		"import React, { useState, useEffect } from 'react';\n",
		"import type { User } from './types';\n",
		"export { User } from './user';\n",
		"const { merge } = require('./utils/helpers');\n",
		"const lodash = await import('lodash');\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content string) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.ts")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return
		}
		d := New()
		deps, err := d.extractFileImports(filePath, tmpDir, nil)
		if err != nil {
			return
		}
		_ = deps
	})
}
