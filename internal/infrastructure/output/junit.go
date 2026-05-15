package output

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// JUnitReporter implements ports.Reporter for JUnit XML output
type JUnitReporter struct {
	tool    string
	version string
}

// NewJUnitReporter creates a new JUnit reporter
func NewJUnitReporter() *JUnitReporter {
	return &JUnitReporter{
		tool:    "arx",
		version: "1.0",
	}
}

// JUnitTestSuite represents a JUnit test suite
type JUnitTestSuite struct {
	XMLName   xml.Name      `xml:"testsuite"`
	Name      string        `xml:"name,attr"`
	Tests     int           `xml:"tests,attr"`
	Failures  int           `xml:"failures,attr"`
	Skipped   int           `xml:"skipped,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test
type JUnitSkipped struct {
	Message string `xml:"message,attr"`
}

// Report implements ports.Reporter interface
func (r *JUnitReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	testSuite := r.buildTestSuite(violations)

	// Wrap in testsuites root element
	type TestSuites struct {
		XMLName xml.Name       `xml:"testsuites"`
		Suites  []JUnitTestSuite `xml:"testsuite"`
	}

	suites := TestSuites{
		Suites: []JUnitTestSuite{testSuite},
	}

	// Marshal to XML
	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	// Write XML declaration and content
	fmt.Fprintln(os.Stdout, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintln(os.Stdout, string(data))

	return nil
}

// buildTestSuite constructs a JUnit test suite from violations
func (r *JUnitReporter) buildTestSuite(violations []domain.Violation) JUnitTestSuite {
	testCases := make([]JUnitTestCase, 0, len(violations))
	failures := 0
	skipped := 0

	for _, v := range violations {
		testCase := JUnitTestCase{
			Name:      v.ID,
			Classname: v.File,
		}

		switch v.Severity {
		case domain.SeverityError:
			testCase.Failure = &JUnitFailure{
				Message: fmt.Sprintf("%s → %s", v.SourceLayer, v.TargetLayer),
				Type:    "error",
				Text:    v.Message,
			}
			failures++
		case domain.SeverityWarning:
			testCase.Skipped = &JUnitSkipped{
				Message: fmt.Sprintf("%s → %s", v.SourceLayer, v.TargetLayer),
			}
			skipped++
		default:
			// Info severity: just a passing test case
		}

		testCases = append(testCases, testCase)
	}

	return JUnitTestSuite{
		Name:      r.tool,
		Tests:     len(violations),
		Failures:  failures,
		Skipped:   skipped,
		TestCases: testCases,
	}
}
