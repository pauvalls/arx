package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveIncludes_Basic(t *testing.T) {
	dir := t.TempDir()

	// Create the included file
	includePath := filepath.Join(dir, "shared.yaml")
	if err := os.WriteFile(includePath, []byte("path: ./shared\n"), 0644); err != nil {
		t.Fatalf("failed to write include file: %v", err)
	}

	// Create the main file with !include
	mainContent := []byte("version: \"1.0\"\nlayers: !include shared.yaml\nrules: []\n")
	result, err := ResolveIncludes(dir, mainContent)
	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v", err)
	}

	// The result should have the included content replacing the !include node
	// It should still be valid YAML and contain the resolved content
	resultStr := string(result)
	if !strings.Contains(resultStr, "path: ./shared") {
		t.Errorf("ResolveIncludes() missing include content.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "version:") {
		t.Errorf("ResolveIncludes() missing original content.\n got: %s", resultStr)
	}
}

func TestResolveIncludes_Nested(t *testing.T) {
	dir := t.TempDir()

	// Create innermost file
	innerPath := filepath.Join(dir, "inner.yaml")
	if err := os.WriteFile(innerPath, []byte("setting: deep\n"), 0644); err != nil {
		t.Fatalf("failed to write inner file: %v", err)
	}

	// Create middle file that includes inner
	middlePath := filepath.Join(dir, "middle.yaml")
	if err := os.WriteFile(middlePath, []byte("middle: !include inner.yaml\n"), 0644); err != nil {
		t.Fatalf("failed to write middle file: %v", err)
	}

	// Create main file that includes middle
	mainContent := []byte("top: !include middle.yaml\n")
	result, err := ResolveIncludes(dir, mainContent)
	if err != nil {
		t.Fatalf("ResolveIncludes() nested error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "setting: deep") {
		t.Errorf("ResolveIncludes() nested include missing deep content.\n got: %s", resultStr)
	}
}

func TestResolveIncludes_Circular(t *testing.T) {
	dir := t.TempDir()

	// Create a.yaml that includes b.yaml
	aPath := filepath.Join(dir, "a.yaml")
	if err := os.WriteFile(aPath, []byte("a: !include b.yaml\n"), 0644); err != nil {
		t.Fatalf("failed to write a.yaml: %v", err)
	}

	// Create b.yaml that includes a.yaml (circular)
	bPath := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(bPath, []byte("b: !include a.yaml\n"), 0644); err != nil {
		t.Fatalf("failed to write b.yaml: %v", err)
	}

	mainContent := []byte("root: !include a.yaml\n")
	_, err := ResolveIncludes(dir, mainContent)
	if err == nil {
		t.Fatal("ResolveIncludes() expected error for circular include, got nil")
	}
	if !strings.Contains(err.Error(), "circular") && !strings.Contains(err.Error(), "cycle") {
		t.Errorf("ResolveIncludes() error should mention circular/cycle, got: %v", err)
	}
}

func TestResolveIncludes_MissingFile(t *testing.T) {
	dir := t.TempDir()

	mainContent := []byte("data: !include nonexistent.yaml\n")
	_, err := ResolveIncludes(dir, mainContent)
	if err == nil {
		t.Fatal("ResolveIncludes() expected error for missing file, got nil")
	}
}

func TestResolveIncludes_RelativePath(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create included file in sub directory
	includePath := filepath.Join(subDir, "dep.yaml")
	if err := os.WriteFile(includePath, []byte("dep: true\n"), 0644); err != nil {
		t.Fatalf("failed to write dep.yaml: %v", err)
	}

	// Main file references it with relative path from root
	mainContent := []byte("data: !include sub/dep.yaml\n")
	result, err := ResolveIncludes(dir, mainContent)
	if err != nil {
		t.Fatalf("ResolveIncludes() error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "dep: true") {
		t.Errorf("ResolveIncludes() missing relative include content.\n got: %s", resultStr)
	}
}

func TestResolveIncludes_DiamondDependency(t *testing.T) {
	dir := t.TempDir()

	// shared.yaml included by both a.yaml and main
	sharedPath := filepath.Join(dir, "shared.yaml")
	if err := os.WriteFile(sharedPath, []byte("shared: true\n"), 0644); err != nil {
		t.Fatalf("failed to write shared.yaml: %v", err)
	}

	// a.yaml includes shared
	aPath := filepath.Join(dir, "a.yaml")
	if err := os.WriteFile(aPath, []byte("a: !include shared.yaml\n"), 0644); err != nil {
		t.Fatalf("failed to write a.yaml: %v", err)
	}

	// Main includes both a.yaml and shared.yaml (diamond)
	mainContent := []byte("first: !include a.yaml\nsecond: !include shared.yaml\n")
	result, err := ResolveIncludes(dir, mainContent)
	if err != nil {
		t.Fatalf("ResolveIncludes() diamond error = %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "shared: true") {
		t.Errorf("ResolveIncludes() diamond: missing shared content.\n got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "a:") {
		t.Errorf("ResolveIncludes() diamond: missing a content.\n got: %s", resultStr)
	}
}

func TestResolveIncludes_NoInclude(t *testing.T) {
	dir := t.TempDir()

	input := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\n")
	result, err := ResolveIncludes(dir, input)
	if err != nil {
		t.Fatalf("ResolveIncludes() no-include error = %v", err)
	}

	// Round-trip through YAML marshal/unmarshal changes indentation
	// (yaml.Marshal uses 2-space indent). Verify semantic equivalence instead.
	resultStr := string(result)
	if !strings.Contains(resultStr, "version:") {
		t.Errorf("ResolveIncludes() missing version key")
	}
	if !strings.Contains(resultStr, "name: domain") {
		t.Errorf("ResolveIncludes() missing layer name")
	}
	if !strings.Contains(resultStr, "paths:") {
		t.Errorf("ResolveIncludes() missing paths key")
	}
}
