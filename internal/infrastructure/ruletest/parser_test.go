package ruletest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_Valid(t *testing.T) {
	content := `
tests:
  - name: "domain should not depend on infra"
    fixture: "test/fixtures/go-project"
    rule: "D-01"
    expect:
      violations: 2
      files:
        - "internal/domain/**"
      layers:
        - source: "domain"
          target: "infrastructure"
      patterns:
        - "depends on"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test_valid.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	suites, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}
	if len(suites[0].Tests) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(suites[0].Tests))
	}

	tc := suites[0].Tests[0]
	if tc.Name != "domain should not depend on infra" {
		t.Errorf("test name = %q, want %q", tc.Name, "domain should not depend on infra")
	}
	if tc.Fixture != "test/fixtures/go-project" {
		t.Errorf("fixture = %q, want %q", tc.Fixture, "test/fixtures/go-project")
	}
	if tc.RuleID != "D-01" {
		t.Errorf("rule = %q, want %q", tc.RuleID, "D-01")
	}
	if tc.Expect.Violations == nil || *tc.Expect.Violations != 2 {
		t.Errorf("expect.violations = %v, want 2", tc.Expect.Violations)
	}
	if len(tc.Expect.Files) != 1 || tc.Expect.Files[0] != "internal/domain/**" {
		t.Errorf("expect.files = %v, want [internal/domain/**]", tc.Expect.Files)
	}
	if len(tc.Expect.Layers) != 1 {
		t.Fatalf("expected 1 layer expectation, got %d", len(tc.Expect.Layers))
	}
	if tc.Expect.Layers[0].Source != "domain" || tc.Expect.Layers[0].Target != "infrastructure" {
		t.Errorf("layer expectation = %+v, want {Source:domain Target:infrastructure}", tc.Expect.Layers[0])
	}
	if len(tc.Expect.Patterns) != 1 || tc.Expect.Patterns[0] != "depends on" {
		t.Errorf("expect.patterns = %v, want [depends on]", tc.Expect.Patterns)
	}
}

func TestParseFile_MissingExpect(t *testing.T) {
	content := `
tests:
  - name: "no expectations"
    fixture: "test/fixtures/go-project"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "missing_expect.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err := parser.ParseFile(path)
	if err == nil {
		t.Fatal("expected error for missing expectations")
	}
}

func TestParseFile_DuplicateNames(t *testing.T) {
	content := `
tests:
  - name: "same name"
    expect:
      violations: 1
  - name: "same name"
    expect:
      violations: 2
`
	dir := t.TempDir()
	path := filepath.Join(dir, "duplicates.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err := parser.ParseFile(path)
	if err == nil {
		t.Fatal("expected error for duplicate test names")
	}
}

func TestParseFile_InvalidYAML(t *testing.T) {
	content := `
tests:
  - name: "bad yaml
    expect:
      violations: 1
`
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err := parser.ParseFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseDir(t *testing.T) {
	dir := t.TempDir()

	// Create two test files
	file1 := `
tests:
  - name: "test1"
    expect:
      violations: 1
`
	file2 := `
tests:
  - name: "test2"
    expect:
      files:
        - "internal/**"
`
	if err := os.WriteFile(filepath.Join(dir, "test1.yaml"), []byte(file1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test2.yaml"), []byte(file2), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a non-test file that should be ignored
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	suites, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir returned error: %v", err)
	}
	if len(suites) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(suites))
	}

	// Verify both test names are present
	names := make(map[string]bool)
	for _, s := range suites {
		for _, tc := range s.Tests {
			names[tc.Name] = true
		}
	}
	if !names["test1"] {
		t.Error("test1 not found")
	}
	if !names["test2"] {
		t.Error("test2 not found")
	}
}

func TestParseDir_NoYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	// Create only non-yaml files
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	suites, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir returned error: %v", err)
	}
	if len(suites) != 0 {
		t.Fatalf("expected 0 suites, got %d", len(suites))
	}
}

func TestParseFile_EmptyName(t *testing.T) {
	content := `
tests:
  - name: ""
    expect:
      violations: 1
`
	dir := t.TempDir()
	path := filepath.Join(dir, "empty_name.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	_, err := parser.ParseFile(path)
	if err == nil {
		t.Fatal("expected error for empty test name")
	}
}

func TestParse_FilePath(t *testing.T) {
	// Test that Parse works with file path (delegates to ParseFile)
	content := `
tests:
  - name: "single test"
    expect:
      violations: 3
`
	dir := t.TempDir()
	path := filepath.Join(dir, "single.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	suites, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}
	if suites[0].Tests[0].Expect.Violations == nil || *suites[0].Tests[0].Expect.Violations != 3 {
		t.Errorf("expected violations=3, got %v", suites[0].Tests[0].Expect.Violations)
	}
}

func TestParseFile_AllExpectFields(t *testing.T) {
	content := `
tests:
  - name: "all fields"
    fixture: "test/fixtures/go-project"
    rule: "D-01"
    expect:
      violations: 5
      files:
        - "internal/**"
        - "cmd/**"
      layers:
        - source: "domain"
          target: "infrastructure"
        - source: "application"
          target: "infrastructure"
      patterns:
        - "depends"
        - "import"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "all_fields.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	suites, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}

	tc := suites[0].Tests[0]
	if tc.Expect.Violations == nil || *tc.Expect.Violations != 5 {
		t.Errorf("violations = %v, want 5", tc.Expect.Violations)
	}
	if len(tc.Expect.Files) != 2 {
		t.Errorf("expected 2 file patterns, got %d", len(tc.Expect.Files))
	}
	if len(tc.Expect.Layers) != 2 {
		t.Errorf("expected 2 layer expectations, got %d", len(tc.Expect.Layers))
	}
	if len(tc.Expect.Patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(tc.Expect.Patterns))
	}

	// Verify the suite validates correctly
	if err := suites[0].Validate(); err != nil {
		t.Errorf("suite validation failed: %v", err)
	}
}
