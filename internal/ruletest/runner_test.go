package ruletest

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

// mockConfigReader implements the ConfigReader interface for testing
type mockConfigReader struct {
	config *domain.Config
	err    error
}

func (m *mockConfigReader) Read(configPath string) (*domain.Config, error) {
	return m.config, m.err
}

// mockDetectFunc returns known dependencies
func mockDetectFunc(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return []domain.Dependency{
		{SourceFile: "internal/domain/a.go", SourceLine: 10, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
		{SourceFile: "internal/application/b.go", SourceLine: 20, ImportPath: "internal/infrastructure/db", ResolvedLayer: "infrastructure"},
	}, nil
}

func TestRuleTestRunner_Run_Pass(t *testing.T) {
	reader := &mockConfigReader{
		config: &domain.Config{
			Version: "1.0",
			Layers: []domain.Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "application", Paths: []string{"internal/application/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			Rules: []domain.Rule{
				{
					ID:       "D-01",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     domain.RuleTypeCannot,
					Severity: domain.SeverityError,
				},
			},
		},
	}

	runner := NewRuleTestRunner(reader, mockDetectFunc)
	suite := TestSuite{
		Name: "test-suite",
		Tests: []TestCase{
			{
				Name:   "domain should not depend on infra",
				Expect: Expectation{Violations: IntPtr(1)},
			},
		},
	}

	result, err := runner.Run(suite, "/fake/path/arx.yaml")
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected suite to pass, got cases: %+v", result.Cases)
	}
}

func TestRuleTestRunner_Run_FailCount(t *testing.T) {
	reader := &mockConfigReader{
		config: &domain.Config{
			Version: "1.0",
			Layers: []domain.Layer{
				{Name: "domain", Paths: []string{"internal/domain/"}},
				{Name: "application", Paths: []string{"internal/application/"}},
				{Name: "infrastructure", Paths: []string{"internal/infrastructure/"}},
			},
			Rules: []domain.Rule{
				{
					ID:       "D-01",
					From:     "domain",
					To:       []string{"infrastructure"},
					Type:     domain.RuleTypeCannot,
					Severity: domain.SeverityError,
				},
			},
		},
	}

	runner := NewRuleTestRunner(reader, mockDetectFunc)
	suite := TestSuite{
		Name: "test-suite",
		Tests: []TestCase{
			{
				Name:   "expect 0 violations (will fail)",
				Expect: Expectation{Violations: IntPtr(0)},
			},
		},
	}

	result, err := runner.Run(suite, "/fake/path")
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected suite to fail (count mismatch)")
	}
}

func TestRuleTestRunner_NonexistentPath(t *testing.T) {
	suite := TestSuite{
		Name:  "empty",
		Tests: []TestCase{},
	}

	// Reader should return nil, err when file doesn't exist
	readerFails := &mockConfigReader{
		err: assertError("config file not found"),
	}
	runner := NewRuleTestRunner(readerFails, nil)
	_, err := runner.Run(suite, "/nonexistent/arx.yaml")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestRuleTestRunner_PanicRecovery(t *testing.T) {
	reader := &mockConfigReader{
		config: &domain.Config{},
	}

	panicFunc := func(projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
		panic("test panic in detector")
	}

	runner := NewRuleTestRunner(reader, panicFunc)
	suite := TestSuite{
		Name: "panic-test",
		Tests: []TestCase{
			{
				Name:   "will panic",
				Expect: Expectation{Violations: IntPtr(0)},
			},
		},
	}

	result, err := runner.Run(suite, "/fake/path")
	if err != nil {
		t.Fatalf("Run returned unexpected error after panic recovery: %v", err)
	}
	if result.Passed {
		t.Error("expected result to not pass after panic")
	}
}

func TestRuleTestRunner_EmptySuite(t *testing.T) {
	reader := &mockConfigReader{
		config: &domain.Config{Version: "1.0"},
	}

	runner := NewRuleTestRunner(reader, mockDetectFunc)
	suite := TestSuite{
		Name:  "empty",
		Tests: []TestCase{},
	}

	result, err := runner.Run(suite, "/fake/path")
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("empty suite should not pass")
	}
}

// assertError returns an error with the given message
type assertError string

func (e assertError) Error() string { return string(e) }
