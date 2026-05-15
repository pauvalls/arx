package output

import (
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestGitHubAnnotationsReporter_Report(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violations []domain.Violation
		wantCount  int
	}{
		{
			name:       "no violations",
			violations: []domain.Violation{},
			wantCount:  0,
		},
		{
			name: "single violation",
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
			wantCount: 1,
		},
		{
			name: "multiple violations",
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
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewGitHubAnnotationsReporter()
			annotations := reporter.buildAnnotations(tt.violations)

			if len(annotations) != tt.wantCount {
				t.Errorf("Annotation count mismatch: got %d, want %d", len(annotations), tt.wantCount)
			}
		})
	}
}

func TestGitHubAnnotationsReporter_FormatViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violation  domain.Violation
		wantPrefix string
		wantTitle  string
	}{
		{
			name: "error severity",
			violation: domain.Violation{
				ID:          "D-01",
				RuleID:      "domain-cannot-depend-on-infrastructure",
				Severity:    domain.SeverityError,
				File:        "internal/domain/order.go",
				Line:        14,
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Message:     "Domain should not depend on infrastructure",
			},
			wantPrefix: "::error",
			wantTitle:  "domain-cannot-depend-on-infrastructure",
		},
		{
			name: "warning severity",
			violation: domain.Violation{
				ID:          "W-01",
				RuleID:      "application-should-depend-on-ports",
				Severity:    domain.SeverityWarning,
				File:        "internal/application/service.go",
				Line:        25,
				SourceLayer: "application",
				TargetLayer: "infrastructure",
				Message:     "Application should depend on ports",
			},
			wantPrefix: "::warning",
			wantTitle:  "application-should-depend-on-ports",
		},
		{
			name: "info severity",
			violation: domain.Violation{
				ID:          "I-01",
				RuleID:      "suggestion",
				Severity:    domain.SeverityInfo,
				File:        "presentation/handler.go",
				Line:        10,
				SourceLayer: "presentation",
				TargetLayer: "infrastructure",
				Message:     "Consider using ports",
			},
			wantPrefix: "::notice",
			wantTitle:  "suggestion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewGitHubAnnotationsReporter()
			result := reporter.formatViolation(tt.violation)

			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("Expected prefix %q, got %q", tt.wantPrefix, result)
			}

			// Verify file path is present
			if !strings.Contains(result, "file="+tt.violation.File) {
				t.Errorf("Missing file path in annotation: %s", result)
			}

			// Verify line number is present
			if !strings.Contains(result, "line=14") && !strings.Contains(result, "line=25") && !strings.Contains(result, "line=10") {
				t.Errorf("Missing line number in annotation: %s", result)
			}

			// Verify title is present
			if !strings.Contains(result, "title="+tt.wantTitle) {
				t.Errorf("Missing title in annotation: %s", result)
			}

			// Verify message is present after ::
			parts := strings.SplitN(result, "::", 2)
			if len(parts) != 2 {
				t.Errorf("Invalid annotation format: %s", result)
			}
		})
	}
}

func TestGitHubAnnotationsReporter_TitleTruncation(t *testing.T) {
	t.Parallel()

	reporter := NewGitHubAnnotationsReporter()
	violation := domain.Violation{
		ID:          "D-01",
		RuleID:      "this-is-a-very-long-rule-id-that-exceeds-fifty-characters-and-should-be-truncated",
		Severity:    domain.SeverityError,
		File:        "internal/domain/order.go",
		Line:        14,
		SourceLayer: "domain",
		TargetLayer: "infrastructure",
		Message:     "Test message",
	}

	result := reporter.formatViolation(violation)

	// Title should be truncated to 50 chars
	if strings.Contains(result, "should-be-truncated") {
		t.Errorf("Title not truncated: %s", result)
	}
}

func TestGitHubAnnotationsReporter_Escaping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		message   string
		wantEscaped string
	}{
		{
			name:      "percent sign",
			message:   "Message with % percent",
			wantEscaped: "Message with %25 percent",
		},
		{
			name:      "newline",
			message:   "Message\nwith newline",
			wantEscaped: "Message%0Awith newline",
		},
		{
			name:      "carriage return",
			message:   "Message\rwith CR",
			wantEscaped: "Message%0Dwith CR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := escapeWorkflowMessage(tt.message)
			if result != tt.wantEscaped {
				t.Errorf("escapeWorkflowMessage() = %q, want %q", result, tt.wantEscaped)
			}
		})
	}
}

func TestGitHubAnnotationsReporter_FilePathEscaping(t *testing.T) {
	t.Parallel()

	reporter := NewGitHubAnnotationsReporter()
	violation := domain.Violation{
		ID:          "D-01",
		RuleID:      "test-rule",
		Severity:    domain.SeverityError,
		File:        "path/with%percent/file.go",
		Line:        14,
		SourceLayer: "domain",
		TargetLayer: "infrastructure",
		Message:     "Test",
	}

	result := reporter.formatViolation(violation)

	// Percent in file path should be escaped
	if strings.Contains(result, "file=path/with%percent/file.go") {
		t.Errorf("File path percent not escaped: %s", result)
	}
	if !strings.Contains(result, "file=path/with%25percent/file.go") {
		t.Errorf("File path percent not properly escaped: %s", result)
	}
}

func TestGitHubAnnotationsReporter_Report_Integration(t *testing.T) {
	t.Parallel()

	reporter := NewGitHubAnnotationsReporter()
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
