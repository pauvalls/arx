package output

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestJUnitReporter_Report(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violations []domain.Violation
		wantTests  int
		wantFailures int
		wantSkipped int
	}{
		{
			name:       "no violations",
			violations: []domain.Violation{},
			wantTests:  0,
			wantFailures: 0,
			wantSkipped:  0,
		},
		{
			name: "single error",
			violations: []domain.Violation{
				{
					ID:          "D-01",
					RuleID:      "domain-cannot-depend-on-infrastructure",
					Severity:    domain.SeverityError,
					File:        "internal/domain/order.go",
					Line:        14,
					SourceLayer: "domain",
					TargetLayer: "infrastructure",
					Import:      "github.com/example/app/internal/infrastructure/postgres",
					Message:     "Domain should not depend on infrastructure",
				},
			},
			wantTests:    1,
			wantFailures: 1,
			wantSkipped:  0,
		},
		{
			name: "single warning",
			violations: []domain.Violation{
				{
					ID:          "W-01",
					RuleID:      "application-should-depend-on-ports",
					Severity:    domain.SeverityWarning,
					File:        "internal/application/service.go",
					Line:        25,
					SourceLayer: "application",
					TargetLayer: "infrastructure",
					Import:      "github.com/example/app/internal/infrastructure/db",
					Message:     "Application should depend on ports",
				},
			},
			wantTests:    1,
			wantFailures: 0,
			wantSkipped:  1,
		},
		{
			name: "mixed severities",
			violations: []domain.Violation{
				{
					ID:          "D-01",
					Severity:    domain.SeverityError,
					File:        "domain/order.go",
					Line:        14,
					SourceLayer: "domain",
					TargetLayer: "infrastructure",
					Message:     "Domain should not depend on infrastructure",
				},
				{
					ID:          "D-02",
					Severity:    domain.SeverityError,
					File:        "application/service.go",
					Line:        25,
					SourceLayer: "application",
					TargetLayer: "infrastructure",
					Message:     "Application should not depend on infrastructure",
				},
				{
					ID:          "W-01",
					Severity:    domain.SeverityWarning,
					File:        "presentation/handler.go",
					Line:        10,
					SourceLayer: "presentation",
					TargetLayer: "infrastructure",
					Message:     "Presentation should not depend on infrastructure",
				},
			},
			wantTests:    3,
			wantFailures: 2,
			wantSkipped:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewJUnitReporter()
			suite := reporter.buildTestSuite(tt.violations)

			if suite.Tests != tt.wantTests {
				t.Errorf("Tests count mismatch: got %d, want %d", suite.Tests, tt.wantTests)
			}
			if suite.Failures != tt.wantFailures {
				t.Errorf("Failures count mismatch: got %d, want %d", suite.Failures, tt.wantFailures)
			}
			if suite.Skipped != tt.wantSkipped {
				t.Errorf("Skipped count mismatch: got %d, want %d", suite.Skipped, tt.wantSkipped)
			}
		})
	}
}

func TestJUnitReporter_XMLStructure(t *testing.T) {
	t.Parallel()

	reporter := NewJUnitReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    domain.SeverityError,
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "Domain should not depend on infrastructure",
		},
	}

	suite := reporter.buildTestSuite(violations)

	// Marshal to verify XML structure
	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test suite: %v", err)
	}

	xmlStr := string(data)

	// Verify required attributes
	if !strings.Contains(xmlStr, `name="arx"`) {
		t.Errorf("Missing suite name attribute: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `tests="1"`) {
		t.Errorf("Missing tests count: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `failures="1"`) {
		t.Errorf("Missing failures count: %s", xmlStr)
	}
}

func TestJUnitReporter_TestCaseDetails(t *testing.T) {
	t.Parallel()

	reporter := NewJUnitReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    domain.SeverityError,
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "Domain should not depend on infrastructure",
		},
		{
			ID:          "W-01",
			RuleID:      "application-should-depend-on-ports",
			Severity:    domain.SeverityWarning,
			File:        "internal/application/service.go",
			Line:        25,
			SourceLayer: "application",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/db",
			Message:     "Application should depend on ports",
		},
	}

	suite := reporter.buildTestSuite(violations)

	// Verify error test case
	if len(suite.TestCases) != 2 {
		t.Fatalf("Expected 2 test cases, got %d", len(suite.TestCases))
	}

	errorCase := suite.TestCases[0]
	if errorCase.Name != "D-01" {
		t.Errorf("Test case name mismatch: got %q, want %q", errorCase.Name, "D-01")
	}
	if errorCase.Classname != "internal/domain/order.go" {
		t.Errorf("Test case classname mismatch: got %q, want %q", errorCase.Classname, "internal/domain/order.go")
	}
	if errorCase.Failure == nil {
		t.Error("Expected failure element for error violation")
	} else {
		if errorCase.Failure.Message != "domain → infrastructure" {
			t.Errorf("Failure message mismatch: got %q, want %q", errorCase.Failure.Message, "domain → infrastructure")
		}
		if errorCase.Failure.Type != "error" {
			t.Errorf("Failure type mismatch: got %q, want %q", errorCase.Failure.Type, "error")
		}
	}

	// Verify warning test case
	warningCase := suite.TestCases[1]
	if warningCase.Skipped == nil {
		t.Error("Expected skipped element for warning violation")
	} else {
		if warningCase.Skipped.Message != "application → infrastructure" {
			t.Errorf("Skipped message mismatch: got %q, want %q", warningCase.Skipped.Message, "application → infrastructure")
		}
	}
}

func TestJUnitReporter_XMLEscaping(t *testing.T) {
	t.Parallel()

	reporter := NewJUnitReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			Severity:    domain.SeverityError,
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Message:     "Message with <special> & \"characters\"",
		},
	}

	suite := reporter.buildTestSuite(violations)

	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	xmlStr := string(data)

	// Verify special characters are escaped
	if strings.Contains(xmlStr, "<special>") {
		t.Error("Special characters not escaped in XML")
	}
	if !strings.Contains(xmlStr, "&lt;special&gt;") {
		t.Errorf("Expected escaped special characters in: %s", xmlStr)
	}
}

func TestJUnitReporter_Report_Integration(t *testing.T) {
	t.Parallel()

	reporter := NewJUnitReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			Severity:    domain.SeverityError,
			File:        "domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Message:     "Domain should not depend on infrastructure",
		},
	}

	// Test that Report doesn't error
	err := reporter.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Errorf("Report failed: %v", err)
	}
}
