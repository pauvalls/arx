package ruletest

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestEvalSuite_AllPass(t *testing.T) {
	suite := TestSuite{
		Name: "all-pass",
		Tests: []TestCase{
			{
				Name:   "expect 2 violations",
				Expect: Expectation{Violations: IntPtr(2)},
			},
		},
	}

	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/a.go", SourceLayer: "domain", TargetLayer: "infra", Message: "dep"},
		{ID: "D-02", File: "internal/app/b.go", SourceLayer: "app", TargetLayer: "infra", Message: "dep2"},
	}

	result := EvalSuite(suite, violations)

	if !result.Passed {
		t.Error("EvalSuite should return Passed=true for all passing tests")
	}
	if len(result.Cases) != 1 {
		t.Errorf("expected 1 case result, got %d", len(result.Cases))
	}
	if !result.Cases[0].Passed {
		t.Errorf("case should be passed, got details: %s", result.Cases[0].Details)
	}
}

func TestEvalSuite_MixedPassFail(t *testing.T) {
	suite := TestSuite{
		Name: "mixed",
		Tests: []TestCase{
			{
				Name:   "should pass (2 violations)",
				Expect: Expectation{Violations: IntPtr(2)},
			},
			{
				Name:   "should fail (expect 99)",
				Expect: Expectation{Violations: IntPtr(99)},
			},
		},
	}

	violations := []domain.Violation{
		{ID: "D-01", File: "a.go"},
		{ID: "D-02", File: "b.go"},
	}

	result := EvalSuite(suite, violations)

	if result.Passed {
		t.Error("EvalSuite should return Passed=false when any case fails")
	}
	if len(result.Cases) != 2 {
		t.Fatalf("expected 2 case results, got %d", len(result.Cases))
	}
	if !result.Cases[0].Passed {
		t.Errorf("first case should pass: %s", result.Cases[0].Details)
	}
	if result.Cases[1].Passed {
		t.Errorf("second case should fail: %s", result.Cases[1].Details)
	}
}

func TestEvalSuite_EmptyViolations(t *testing.T) {
	suite := TestSuite{
		Name: "empty-violations",
		Tests: []TestCase{
			{
				Name:   "expect 0 violations",
				Expect: Expectation{Violations: IntPtr(0)},
			},
			{
				Name:   "expect file match on nil",
				Expect: Expectation{Files: []string{"internal/**"}},
			},
		},
	}

	result := EvalSuite(suite, nil)

	if len(result.Cases) != 2 {
		t.Fatalf("expected 2 case results, got %d", len(result.Cases))
	}
	if !result.Cases[0].Passed {
		t.Errorf("case 'expect 0 violations' should pass with nil violations")
	}
	if result.Cases[1].Passed {
		t.Errorf("case 'expect file match on nil' should fail with nil violations")
	}
}

func TestEvalSuite_MultipleExpectations(t *testing.T) {
	// Test case with count AND files AND layers AND patterns expectations (AND logic)
	suite := TestSuite{
		Name: "multi-expect",
		Tests: []TestCase{
			{
				Name: "all expectations met",
				Expect: Expectation{
					Violations: IntPtr(2),
					Files:      []string{"internal/domain/*.go"},
					Layers:     []LayerExpectation{{Source: "domain", Target: "infra"}},
					Patterns:   []string{"dep"},
				},
			},
		},
	}

	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/a.go", SourceLayer: "domain", TargetLayer: "infra", Message: "dep from domain to infra"},
		{ID: "D-02", File: "internal/domain/b.go", SourceLayer: "domain", TargetLayer: "infra", Message: "another dep"},
	}

	result := EvalSuite(suite, violations)

	if !result.Passed {
		t.Errorf("all expectations should be met: %s", result.Cases[0].Details)
	}
}

func TestEvalSuite_MultipleExpectationsFail(t *testing.T) {
	suite := TestSuite{
		Name: "multi-expect-fail",
		Tests: []TestCase{
			{
				Name: "count passes but files fail",
				Expect: Expectation{
					Violations: IntPtr(1),
					Files:      []string{"cmd/**"},
				},
			},
		},
	}

	violations := []domain.Violation{
		{ID: "D-01", File: "internal/domain/a.go", Message: "dep"},
	}

	result := EvalSuite(suite, violations)

	if result.Passed {
		t.Error("should fail when one expectation is not met")
	}
}

func TestEvalSuite_EmptySuite(t *testing.T) {
	suite := TestSuite{
		Name:  "empty",
		Tests: []TestCase{},
	}

	result := EvalSuite(suite, []domain.Violation{})

	if result.Passed {
		t.Error("empty suite should not pass")
	}
	if len(result.Cases) != 0 {
		t.Errorf("empty suite should have 0 cases, got %d", len(result.Cases))
	}
}
