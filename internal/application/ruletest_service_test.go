package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain/ruletest"
)

// mockTestParser implements TestParser for unit testing
type mockTestParser struct {
	suites []ruletest.TestSuite
	err    error
}

func (m *mockTestParser) Parse(path string) ([]ruletest.TestSuite, error) {
	return m.suites, m.err
}

// mockTestRunner implements TestRunner for unit testing
type mockTestRunner struct {
	results map[string]ruletest.EvalResult
	err     error
}

func (m *mockTestRunner) Run(suite ruletest.TestSuite, fixturePath string) (ruletest.EvalResult, error) {
	if m.err != nil {
		return ruletest.EvalResult{}, m.err
	}
	if res, ok := m.results[suite.Name]; ok {
		return res, nil
	}
	return ruletest.EvalResult{Passed: true}, nil
}

// mockTestReporter implements TestReporter for unit testing
type mockTestReporter struct {
	lastResults []ruletest.CaseResult
}

func (m *mockTestReporter) Report(results []ruletest.CaseResult, verbose bool) string {
	m.lastResults = results
	return "mock table output"
}

func (m *mockTestReporter) ReportJUnit(results []ruletest.CaseResult, time string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<testsuites><testsuite name="arx.rule-test" tests="1" failures="0" errors="0" time="0s"><testcase name="mock" time="0s"/></testsuite></testsuites>`
}

// assertError is a simple error type for testing
type assertError string

func (e assertError) Error() string { return string(e) }

func TestRuleTestService_RunTests_AllPass(t *testing.T) {
	parser := &mockTestParser{
		suites: []ruletest.TestSuite{
			{
				Name: "suite1",
				Tests: []ruletest.TestCase{
					{Name: "test1", Fixture: "/tmp/fake", Expect: ruletest.Expectation{Violations: ruletest.IntPtr(0)}},
				},
			},
		},
	}

	runner := &mockTestRunner{
		results: map[string]ruletest.EvalResult{
			"suite1": {
				Passed: true,
				Cases: []ruletest.CaseResult{
					{Name: "test1", Passed: true, Details: "ok"},
				},
			},
		},
	}

	reporter := &mockTestReporter{}
	service := NewRuleTestService(parser, runner, reporter)
	result, err := service.RunTests("/fake/path", TestOptions{})
	if err != nil {
		t.Fatalf("RunTests returned error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected all pass, got failed")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRuleTestService_RunTests_Fail(t *testing.T) {
	parser := &mockTestParser{
		suites: []ruletest.TestSuite{
			{
				Name: "suite1",
				Tests: []ruletest.TestCase{
					{Name: "test1", Expect: ruletest.Expectation{Violations: ruletest.IntPtr(0)}},
				},
			},
		},
	}

	runner := &mockTestRunner{
		results: map[string]ruletest.EvalResult{
			"suite1": {
				Passed: false,
				Cases: []ruletest.CaseResult{
					{Name: "test1", Passed: false, Details: "expected 0 violations, got 2"},
				},
			},
		},
	}

	reporter := &mockTestReporter{}
	service := NewRuleTestService(parser, runner, reporter)
	result, err := service.RunTests("/fake/path", TestOptions{CI: true})
	if err != nil {
		t.Fatalf("RunTests returned error: %v", err)
	}
	if result.Passed {
		t.Errorf("expected failure")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1 (CI mode with failures), got %d", result.ExitCode)
	}
}

func TestRuleTestService_RunTests_ParseError(t *testing.T) {
	parser := &mockTestParser{
		err: assertError("parse error: invalid YAML"),
	}

	runner := &mockTestRunner{}
	reporter := &mockTestReporter{}
	service := NewRuleTestService(parser, runner, reporter)
	_, err := service.RunTests("/bad/path", TestOptions{})
	if err == nil {
		t.Fatal("expected error for parse failure")
	}
}

func TestRuleTestService_RunTests_CIExitCode(t *testing.T) {
	tests := []struct {
		name     string
		passed   bool
		exitCode int
	}{
		{"all pass", true, 0},
		{"some fail", false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &mockTestParser{
				suites: []ruletest.TestSuite{
					{
						Name: "suite1",
						Tests: []ruletest.TestCase{
							{Name: tt.name, Expect: ruletest.Expectation{Violations: ruletest.IntPtr(1)}},
						},
					},
				},
			}

			runner := &mockTestRunner{
				results: map[string]ruletest.EvalResult{
					"suite1": {
						Passed: tt.passed,
						Cases: []ruletest.CaseResult{
							{Name: tt.name, Passed: tt.passed, Details: "result"},
						},
					},
				},
			}

			reporter := &mockTestReporter{}
			service := NewRuleTestService(parser, runner, reporter)
			result, err := service.RunTests("/fake/path", TestOptions{})
			if err != nil {
				t.Fatalf("RunTests error: %v", err)
			}
			if result.Passed != tt.passed {
				t.Errorf("expected passed=%v, got %v", tt.passed, result.Passed)
			}
		})
	}
}

func TestRuleTestService_RunTests_JUnitOutput(t *testing.T) {
	parser := &mockTestParser{
		suites: []ruletest.TestSuite{
			{
				Name: "suite1",
				Tests: []ruletest.TestCase{
					{Name: "passing", Expect: ruletest.Expectation{Violations: ruletest.IntPtr(0)}},
				},
			},
		},
	}

	runner := &mockTestRunner{
		results: map[string]ruletest.EvalResult{
			"suite1": {
				Passed: true,
				Cases: []ruletest.CaseResult{
					{Name: "passing", Passed: true, Details: "ok"},
				},
			},
		},
	}

	dir := t.TempDir()
	junitPath := filepath.Join(dir, "results.xml")

	reporter := &mockTestReporter{}
	service := NewRuleTestService(parser, runner, reporter)
	result, err := service.RunTests("/fake/path", TestOptions{JUnitPath: junitPath})
	if err != nil {
		t.Fatalf("RunTests error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass")
	}

	// Verify JUnit file was written
	data, err := os.ReadFile(junitPath)
	if err != nil {
		t.Fatalf("failed to read JUnit file: %v", err)
	}
	if !strings.Contains(string(data), "testsuite") {
		t.Errorf("JUnit file should contain testsuite element")
	}
}

func TestRuleTestService_EmptyPath(t *testing.T) {
	parser := &mockTestParser{
		suites: nil,
		err:    assertError("no test files found"),
	}

	runner := &mockTestRunner{}
	reporter := &mockTestReporter{}
	service := NewRuleTestService(parser, runner, reporter)
	_, err := service.RunTests("/empty/path", TestOptions{})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
