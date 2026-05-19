package ruletest

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/ruletest"
)

func TestJUnitReporter_AllPass(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "test1", Passed: true, Details: "all expectations met"},
		{Name: "test2", Passed: true, Details: "all expectations met"},
	}

	reporter := NewJUnitReporter()
	xmlData := reporter.ReportJUnit(results, "0.05s")

	var suites JUnitTestSuites
	if err := xml.Unmarshal([]byte(xmlData), &suites); err != nil {
		t.Fatalf("failed to parse JUnit XML: %v", err)
	}

	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Name != "arx.rule-test" {
		t.Errorf("suite name = %q, want %q", suite.Name, "arx.rule-test")
	}
	if suite.Tests != 2 {
		t.Errorf("tests = %d, want 2", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("failures = %d, want 0", suite.Failures)
	}
	if suite.Errors != 0 {
		t.Errorf("errors = %d, want 0", suite.Errors)
	}
	if suite.Time != "0.05s" {
		t.Errorf("time = %q, want %q", suite.Time, "0.05s")
	}
}

func TestJUnitReporter_WithFailures(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "passing", Passed: true, Details: "ok"},
		{Name: "failing", Passed: false, Details: "expected 0 violations, got 2"},
	}

	reporter := NewJUnitReporter()
	xmlData := reporter.ReportJUnit(results, "0.1s")

	var suites JUnitTestSuites
	if err := xml.Unmarshal([]byte(xmlData), &suites); err != nil {
		t.Fatalf("failed to parse JUnit XML: %v", err)
	}

	suite := suites.Suites[0]
	if suite.Tests != 2 {
		t.Errorf("tests = %d, want 2", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("failures = %d, want 1", suite.Failures)
	}
	if suite.Errors != 0 {
		t.Errorf("errors = %d, want 0", suite.Errors)
	}

	// Check the failing test case has a failure element
	if len(suite.TestCases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(suite.TestCases))
	}

	passCase := suite.TestCases[0]
	if passCase.Name != "passing" {
		t.Errorf("first test case name = %q", passCase.Name)
	}
	if passCase.Failure != nil {
		t.Error("passing test should not have failure element")
	}

	failCase := suite.TestCases[1]
	if failCase.Name != "failing" {
		t.Errorf("second test case name = %q", failCase.Name)
	}
	if failCase.Failure == nil {
		t.Fatal("failing test should have failure element")
	}
	if failCase.Failure.Message != "expected 0 violations, got 2" {
		t.Errorf("failure message = %q", failCase.Failure.Message)
	}
}

func TestJUnitReporter_WriteFile(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "test1", Passed: true, Details: "ok"},
	}

	reporter := NewJUnitReporter()
	xmlData := reporter.ReportJUnit(results, "0.01s")

	dir := t.TempDir()
	path := filepath.Join(dir, "result.xml")
	if err := os.WriteFile(path, []byte(xmlData), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("failed to parse written XML: %v", err)
	}

	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.Suites))
	}
}

func TestJUnitReporter_EmptyResults(t *testing.T) {
	reporter := NewJUnitReporter()
	xmlData := reporter.ReportJUnit(nil, "0s")

	var suites JUnitTestSuites
	if err := xml.Unmarshal([]byte(xmlData), &suites); err != nil {
		t.Fatalf("failed to parse JUnit XML: %v", err)
	}

	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.Suites))
	}

	suite := suites.Suites[0]
	if suite.Tests != 0 {
		t.Errorf("tests = %d, want 0", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("failures = %d, want 0", suite.Failures)
	}
}

func TestJUnitReporter_XMLDeclaration(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "test", Passed: true, Details: "ok"},
	}

	reporter := NewJUnitReporter()
	xmlData := reporter.ReportJUnit(results, "0.01s")

	if !strings.HasPrefix(xmlData, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Errorf("XML should start with declaration, got: %s", xmlData[:50])
	}
}
