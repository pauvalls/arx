package ruletest

import (
	"encoding/xml"
	"fmt"

	"github.com/pauvalls/arx/internal/ruletest"
)

// JUnitTestSuites is the root element for JUnit XML output
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a JUnit test suite
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single JUnit test case
type JUnitTestCase struct {
	Name    string        `xml:"name,attr"`
	Classname string      `xml:"classname,attr,omitempty"`
	Failure *JUnitFailure `xml:"failure,omitempty"`
	Error   *JUnitError   `xml:"error,omitempty"`
	Time    string        `xml:"time,attr,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
}

// JUnitError represents a test error (panic, unexpected error)
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
}

// JUnitReporter formats test results as JUnit XML
type JUnitReporter struct{}

// NewJUnitReporter creates a new JUnitReporter
func NewJUnitReporter() *JUnitReporter {
	return &JUnitReporter{}
}

// ReportJUnit generates JUnit XML for the given test results and returns it as a string.
// The time parameter is the total execution time (e.g., "0.05s").
func (r *JUnitReporter) ReportJUnit(results []ruletest.CaseResult, time string) string {
	if results == nil {
		results = []ruletest.CaseResult{}
	}

	tc := make([]JUnitTestCase, 0, len(results))
	var failures, errors int

	for _, cr := range results {
		testCase := JUnitTestCase{
			Name: cr.Name,
			Time: "0.00s",
		}

		if !cr.Passed {
			testCase.Failure = &JUnitFailure{
				Message: cr.Details,
				Type:    "AssertionError",
			}
			failures++
		}

		tc = append(tc, testCase)
	}

	suite := JUnitTestSuite{
		Name:      "arx.rule-test",
		Tests:     len(results),
		Failures:  failures,
		Errors:    errors,
		Time:      time,
		TestCases: tc,
	}

	suites := JUnitTestSuites{
		Suites: []JUnitTestSuite{suite},
	}

	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<error>%s</error>\n", err)
	}

	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n%s\n", string(data))
}
