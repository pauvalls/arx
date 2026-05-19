package ruletest

import (
	"fmt"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
)

// CaseResult holds the evaluation result for a single test case
type CaseResult struct {
	Name    string
	Passed  bool
	Details string
}

// EvalResult holds the overall evaluation result for a test suite
type EvalResult struct {
	Passed bool
	Cases  []CaseResult
}

// buildMatchers creates ViolationMatchers from an Expectation
func buildMatchers(exp Expectation) []ViolationMatcher {
	var matchers []ViolationMatcher

	// Add count matcher if violations count was explicitly set
	if exp.Violations != nil {
		matchers = append(matchers, &CountMatcher{Expected: *exp.Violations})
	}
	// Add files matcher if files were specified
	if len(exp.Files) > 0 {
		matchers = append(matchers, &FilesMatcher{Patterns: exp.Files})
	}
	// Add layers matcher if layers were specified
	if len(exp.Layers) > 0 {
		matchers = append(matchers, &LayersMatcher{Expectations: exp.Layers})
	}
	// Add patterns matcher if patterns were specified
	if len(exp.Patterns) > 0 {
		matchers = append(matchers, &PatternsMatcher{Patterns: exp.Patterns})
	}

	return matchers
}

// filterByRule filters violations to only those matching the given rule ID
func filterByRule(violations []domain.Violation, ruleID string) []domain.Violation {
	if ruleID == "" {
		return violations
	}
	var result []domain.Violation
	for _, v := range violations {
		if v.RuleID == ruleID {
			result = append(result, v)
		}
	}
	return result
}

// EvalSuite evaluates a TestSuite against violations and returns EvalResult
func EvalSuite(suite TestSuite, violations []domain.Violation) EvalResult {
	if len(suite.Tests) == 0 {
		return EvalResult{Passed: false, Cases: []CaseResult{}}
	}

	allPassed := true
	cases := make([]CaseResult, 0, len(suite.Tests))

	for _, tc := range suite.Tests {
		// Filter violations by rule ID if specified
		filtered := filterByRule(violations, tc.RuleID)
		matchers := buildMatchers(tc.Expect)
		casePassed := true
		var failDetails []string

		for _, matcher := range matchers {
			matched, detail := matcher.Match(filtered)
			if !matched {
				casePassed = false
				failDetails = append(failDetails, detail)
			}
		}

		detail := ""
		if !casePassed {
			allPassed = false
			detail = strings.Join(failDetails, "; ")
		} else {
			detail = "all expectations met"
		}

		cases = append(cases, CaseResult{
			Name:    tc.Name,
			Passed:  casePassed,
			Details: detail,
		})
	}

	return EvalResult{
		Passed: allPassed,
		Cases:  cases,
	}
}

// EvalSuiteSummary returns a human-readable summary of the evaluation
func EvalSuiteSummary(result EvalResult) string {
	var passed, failed int
	for _, c := range result.Cases {
		if c.Passed {
			passed++
		} else {
			failed++
		}
	}
	total := passed + failed
	if result.Passed {
		return fmt.Sprintf("%d/%d tests passed", passed, total)
	}
	return fmt.Sprintf("%d/%d tests passed, %d failed", passed, total, failed)
}
