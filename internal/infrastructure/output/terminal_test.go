package output_test

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/output"
)

func TestExitCode_WithThreshold(t *testing.T) {
	tests := []struct {
		name          string
		violations    []domain.Violation
		maxViolations int
		wantExitCode  int
	}{
		{
			name:          "0 violations, threshold 0 → exit 0",
			violations:    []domain.Violation{},
			maxViolations: 0,
			wantExitCode:  0,
		},
		{
			name:          "0 violations, threshold 5 → exit 0",
			violations:    []domain.Violation{},
			maxViolations: 5,
			wantExitCode:  0,
		},
		{
			name: "3 violations, threshold 5 → exit 0",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError},
				{ID: "V2", Severity: domain.SeverityError},
				{ID: "V3", Severity: domain.SeverityError},
			},
			maxViolations: 5,
			wantExitCode:  0,
		},
		{
			name: "5 violations, threshold 5 → exit 0",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError},
				{ID: "V2", Severity: domain.SeverityError},
				{ID: "V3", Severity: domain.SeverityError},
				{ID: "V4", Severity: domain.SeverityError},
				{ID: "V5", Severity: domain.SeverityError},
			},
			maxViolations: 5,
			wantExitCode:  0,
		},
		{
			name: "6 violations, threshold 5 → exit 1",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError},
				{ID: "V2", Severity: domain.SeverityError},
				{ID: "V3", Severity: domain.SeverityError},
				{ID: "V4", Severity: domain.SeverityError},
				{ID: "V5", Severity: domain.SeverityError},
				{ID: "V6", Severity: domain.SeverityError},
			},
			maxViolations: 5,
			wantExitCode:  1,
		},
		{
			name: "3 violations (all overridden), threshold 0 → exit 0",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError, Overridden: true},
				{ID: "V2", Severity: domain.SeverityError, Overridden: true},
				{ID: "V3", Severity: domain.SeverityError, Overridden: true},
			},
			maxViolations: 0,
			wantExitCode:  0,
		},
		{
			name: "3 violations (2 overridden), threshold 0 → exit 1",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError, Overridden: true},
				{ID: "V2", Severity: domain.SeverityError, Overridden: true},
				{ID: "V3", Severity: domain.SeverityError, Overridden: false},
			},
			maxViolations: 0,
			wantExitCode:  1,
		},
		{
			name: "3 violations (2 overridden), threshold 5 → exit 0",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError, Overridden: true},
				{ID: "V2", Severity: domain.SeverityError, Overridden: true},
				{ID: "V3", Severity: domain.SeverityError, Overridden: false},
			},
			maxViolations: 5,
			wantExitCode:  0,
		},
		{
			name: "10 violations (all non-overridden), threshold 5 → exit 1",
			violations: []domain.Violation{
				{ID: "V1", Severity: domain.SeverityError},
				{ID: "V2", Severity: domain.SeverityError},
				{ID: "V3", Severity: domain.SeverityError},
				{ID: "V4", Severity: domain.SeverityError},
				{ID: "V5", Severity: domain.SeverityError},
				{ID: "V6", Severity: domain.SeverityError},
				{ID: "V7", Severity: domain.SeverityError},
				{ID: "V8", Severity: domain.SeverityError},
				{ID: "V9", Severity: domain.SeverityError},
				{ID: "V10", Severity: domain.SeverityError},
			},
			maxViolations: 5,
			wantExitCode:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := output.ExitCode(tt.violations, tt.maxViolations)
			if got != tt.wantExitCode {
				t.Errorf("ExitCode() = %d, want %d", got, tt.wantExitCode)
			}
		})
	}
}
