package ruletest

import (
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/ruletest"
)

func TestTableReporter_AllPass(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "test1", Passed: true, Details: "all expectations met"},
		{Name: "test2", Passed: true, Details: "all expectations met"},
	}

	reporter := NewTableReporter()
	output := reporter.Report(results, false)

	if !strings.Contains(output, "2/2 tests passed") {
		t.Errorf("expected summary '2/2 tests passed', got: %s", output)
	}
	if strings.Contains(output, "FAIL") {
		t.Errorf("unexpected FAIL in all-pass output: %s", output)
	}
}

func TestTableReporter_MixedResults(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "passing-test", Passed: true, Details: "all expectations met"},
		{Name: "failing-test", Passed: false, Details: "expected 0 violations, got 2"},
	}

	reporter := NewTableReporter()
	output := reporter.Report(results, false)

	if !strings.Contains(output, "1/2 tests passed") {
		t.Errorf("expected summary '1/2 tests passed', got: %s", output)
	}
	if !strings.Contains(output, "failing-test") {
		t.Errorf("expected failing test name in output: %s", output)
	}
	if !strings.Contains(output, "expected 0 violations, got 2") {
		t.Errorf("expected failure detail in output: %s", output)
	}
}

func TestTableReporter_Verbose(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "passing-test", Passed: true, Details: "all expectations met"},
		{Name: "failing-test", Passed: false, Details: "expected 1, got 0"},
	}

	reporter := NewTableReporter()
	output := reporter.Report(results, true)

	// Verbose should include passing tests
	if !strings.Contains(output, "passing-test") {
		t.Errorf("expected passing test in verbose output: %s", output)
	}
	if !strings.Contains(output, "1/2 tests passed") {
		t.Errorf("expected correct summary '1/2 tests passed': %s", output)
	}
}

func TestTableReporter_EmptySuite(t *testing.T) {
	reporter := NewTableReporter()
	output := reporter.Report(nil, false)

	if !strings.Contains(output, "0/0 tests passed") {
		t.Errorf("expected '0/0 tests passed' summary: %s", output)
	}
}

func TestTableReporter_VerboseShowsPASS(t *testing.T) {
	results := []ruletest.CaseResult{
		{Name: "only-pass", Passed: true, Details: "all expectations met"},
	}

	// Non-verbose should still show PASS for all-pass
	reporter := NewTableReporter()
	output := reporter.Report(results, false)

	if !strings.Contains(output, "PASS") {
		t.Errorf("expected PASS status in all-pass output: %s", output)
	}
}

func TestTableReporter_Header(t *testing.T) {
	reporter := NewTableReporter()
	results := []ruletest.CaseResult{
		{Name: "a-test", Passed: true, Details: "ok"},
	}

	output := reporter.Report(results, false)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (header, separator, data), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "TEST") {
		t.Errorf("header should contain TEST: %s", lines[0])
	}
	if !strings.Contains(lines[0], "STATUS") {
		t.Errorf("header should contain STATUS: %s", lines[0])
	}
}
