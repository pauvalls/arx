package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ruletest"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	infraruletest "github.com/pauvalls/arx/internal/infrastructure/ruletest"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test [path]",
	Short: "Run architecture rule tests",
	Long: `Run architecture rule tests defined in YAML files.

Test files define test cases with expectations (violation count, file matches,
layer matches, pattern matches) that are validated against real fixtures.

If no path is provided, the current directory is scanned for test YAML files.

The test runner:
  1. Discovers and parses YAML test definitions
  2. For each test case, loads the fixture's arx.yaml configuration
  3. Detects dependencies in the fixture directory
  4. Evaluates architectural rules
  5. Compares results against expectations

Exit codes:
  0 - All tests pass
  1 - Some tests fail
  2 - Internal error (parse failure, I/O error)

Examples:
  arx test                           # Run all tests in current directory
  arx test tests/                    # Run all tests in tests/ directory
  arx test tests/my_test.yaml        # Run a specific test file
  arx test --ci                      # CI mode with minimal output
  arx test --junit results.xml       # Write JUnit XML output
  arx test --verbose                 # Show detailed match info`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

var (
	testFixture string
	testRule    string
	testVerbose bool
	testCI      bool
	testJUnit   string
)

func init() {
	testCmd.Flags().StringVar(&testFixture, "fixture", "", "Override fixture path for all test cases")
	testCmd.Flags().StringVar(&testRule, "rule", "", "Filter tests by rule ID (glob)")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Show detailed match info")
	testCmd.Flags().BoolVar(&testCI, "ci", false, "CI mode: exit code reflects pass/fail (minimal output)")
	testCmd.Flags().StringVar(&testJUnit, "junit", "", "Write JUnit XML to file")
	rootCmd.AddCommand(testCmd)
}

// cliTestReporter adapts infrastructure reporters to the application TestReporter interface
type cliTestReporter struct {
	table *infraruletest.TableReporter
	junit *infraruletest.JUnitReporter
}

func (r *cliTestReporter) Report(results []ruletest.CaseResult, verbose bool) string {
	return r.table.Report(results, verbose)
}

func (r *cliTestReporter) ReportJUnit(results []ruletest.CaseResult, time string) string {
	return r.junit.ReportJUnit(results, time)
}

// runTest executes the test command
func runTest(cmd *cobra.Command, args []string) error {
	// Determine test path
	testPath := "."
	if len(args) > 0 {
		testPath = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(testPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path %q: %v\n", testPath, err)
		os.Exit(2)
		return nil
	}

	// Create test service with real dependencies
	service := newTestService()

	// In CI mode, auto-generate JUnit XML if no explicit JUnit path given
	junitPath := testJUnit
	if testCI && junitPath == "" {
		junitPath = "arx-test-results.xml"
	}

	opts := application.TestOptions{
		FixtureOverride: testFixture,
		RuleFilter:      testRule,
		Verbose:         testVerbose,
		JUnitPath:       junitPath,
		CI:              testCI,
	}

	result, err := service.RunTests(absPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
		return nil
	}

	// Print table output
	fmt.Print(result.TableOutput)

	// Exit with appropriate code
	if !result.Passed {
		os.Exit(1)
	}

	return nil
}

// newTestService creates a RuleTestService with real infrastructure dependencies.
func newTestService() *application.RuleTestService {
	reader := config.NewYAMLReader()
	detectors := detector.GetDetectors()

	detectFunc := ruletest.DependenciesFunc(func(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
		ctx := context.Background()
		return application.RunDetectors(ctx, projectRoot, layers, detectors)
	})

	runner := ruletest.NewRuleTestRunner(reader, detectFunc)
	parser := infraruletest.NewParser()
	reporter := &cliTestReporter{
		table: infraruletest.NewTableReporter(),
		junit: infraruletest.NewJUnitReporter(),
	}

	return application.NewRuleTestService(parser, runner, reporter)
}
