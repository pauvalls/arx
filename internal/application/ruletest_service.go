package application

import (
	"fmt"
	"os"
	"time"

	"github.com/pauvalls/arx/internal/ruletest"
)

// TestParser defines the interface for parsing test definition files
type TestParser interface {
	Parse(path string) ([]ruletest.TestSuite, error)
}

// TestRunner defines the interface for running test suites
type TestRunner interface {
	Run(suite ruletest.TestSuite, fixturePath string) (ruletest.EvalResult, error)
}

// TestReporter defines the interface for reporting test results
type TestReporter interface {
	// Report formats results as a human-readable string
	Report(results []ruletest.CaseResult, verbose bool) string
	// ReportJUnit formats results as JUnit XML string
	ReportJUnit(results []ruletest.CaseResult, time string) string
}

// TestOptions configures how tests are run and reported
type TestOptions struct {
	FixtureOverride string // Override fixture path for all test cases
	RuleFilter      string // Filter tests by rule ID (glob)
	Verbose         bool   // Show detailed match info
	JUnitPath       string // Write JUnit XML to this file
	CI              bool   // CI mode: exit code reflects pass/fail/error
}

// TestRunResult holds the aggregated results of a test run
type TestRunResult struct {
	Passed      bool
	ExitCode    int
	Results     []ruletest.CaseResult
	SuiteCount  int
	PassCount   int
	FailCount   int
	Summary     string
	TableOutput string
}

// RuleTestService orchestrates the test flow: parse → run → report
type RuleTestService struct {
	parser   TestParser
	runner   TestRunner
	reporter TestReporter
}

// NewRuleTestService creates a new RuleTestService
func NewRuleTestService(parser TestParser, runner TestRunner, reporter TestReporter) *RuleTestService {
	return &RuleTestService{
		parser:   parser,
		runner:   runner,
		reporter: reporter,
	}
}

// RunTests parses test files in the given path, runs all test suites,
// collects results, and reports them.
func (s *RuleTestService) RunTests(path string, opts TestOptions) (*TestRunResult, error) {
	start := time.Now()

	// Parse test definitions
	suites, err := s.parser.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing test definitions: %w", err)
	}

	// Run each test suite
	var allResults []ruletest.CaseResult
	totalPass := 0
	totalFail := 0

	for _, suite := range suites {
		fixturePath := ""
		if len(suite.Tests) > 0 {
			fixturePath = suite.Tests[0].Fixture
		}

		if opts.FixtureOverride != "" {
			fixturePath = opts.FixtureOverride
		}

		result, err := s.runner.Run(suite, fixturePath)
		if err != nil {
			return nil, fmt.Errorf("running suite %q: %w", suite.Name, err)
		}

		for _, cr := range result.Cases {
			allResults = append(allResults, cr)
			if cr.Passed {
				totalPass++
			} else {
				totalFail++
			}
		}
	}

	// Generate table output
	tableOutput := s.reporter.Report(allResults, opts.Verbose)

	// Generate JUnit output if requested
	if opts.JUnitPath != "" {
		elapsed := time.Since(start)
		timeStr := fmt.Sprintf("%.3fs", elapsed.Seconds())
		junitXML := s.reporter.ReportJUnit(allResults, timeStr)

		if err := os.WriteFile(opts.JUnitPath, []byte(junitXML), 0644); err != nil {
			return nil, fmt.Errorf("writing JUnit XML to %s: %w", opts.JUnitPath, err)
		}
	}

	// Build result
	passed := totalFail == 0
	exitCode := 0
	if !passed {
		exitCode = 1
	}

	result := &TestRunResult{
		Passed:      passed,
		ExitCode:    exitCode,
		Results:     allResults,
		SuiteCount:  len(suites),
		PassCount:   totalPass,
		FailCount:   totalFail,
		Summary:     fmt.Sprintf("%d/%d tests passed", totalPass, totalPass+totalFail),
		TableOutput: tableOutput,
	}

	return result, nil
}
