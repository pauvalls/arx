package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestSARIFReporter_Report(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violations []domain.Violation
		wantSchema string
		wantLevel  string
	}{
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
			wantSchema: "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
			wantLevel:  "error",
		},
		{
			name: "warning severity",
			violations: []domain.Violation{
				{
					ID:          "D-01",
					RuleID:      "application-cannot-depend-on-infrastructure",
					Severity:    domain.SeverityWarning,
					File:        "internal/application/service.go",
					Line:        25,
					SourceLayer: "application",
					TargetLayer: "infrastructure",
					Import:      "github.com/example/app/internal/infrastructure/db",
					Message:     "Application should depend on ports",
				},
			},
			wantSchema: "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
			wantLevel:  "warning",
		},
		{
			name:       "no violations",
			violations: []domain.Violation{},
			wantSchema: "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewSARIFReporter()
			log := reporter.buildSARIFLog(tt.violations)

			// Verify schema
			if log.Schema != tt.wantSchema {
				t.Errorf("Schema mismatch: got %q, want %q", log.Schema, tt.wantSchema)
			}

			// Verify version
			if log.Version != "2.1.0" {
				t.Errorf("Version mismatch: got %q, want 2.1.0", log.Version)
			}

			// Verify results count
			if len(log.Runs) != 1 {
				t.Fatalf("Expected 1 run, got %d", len(log.Runs))
			}

			results := log.Runs[0].Results
			if len(results) != len(tt.violations) {
				t.Fatalf("Results count mismatch: got %d, want %d", len(results), len(tt.violations))
			}

			// Verify first result if violations exist
			if len(tt.violations) > 0 {
				result := results[0]
				if result.Level != tt.wantLevel {
					t.Errorf("Level mismatch: got %q, want %q", result.Level, tt.wantLevel)
				}

				if result.RuleID != tt.violations[0].RuleID {
					t.Errorf("RuleID mismatch: got %q, want %q", result.RuleID, tt.violations[0].RuleID)
				}

				if result.Locations[0].PhysicalLocation.ArtifactLocation.URI != tt.violations[0].File {
					t.Errorf("File mismatch: got %q, want %q", result.Locations[0].PhysicalLocation.ArtifactLocation.URI, tt.violations[0].File)
				}

				if result.Locations[0].PhysicalLocation.Region.StartLine != tt.violations[0].Line {
					t.Errorf("Line mismatch: got %d, want %d", result.Locations[0].PhysicalLocation.Region.StartLine, tt.violations[0].Line)
				}
			}
		})
	}
}

func TestSARIFReporter_Report_JSONSerialization(t *testing.T) {
	t.Parallel()

	reporter := NewSARIFReporter()
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

	log := reporter.buildSARIFLog(violations)

	// Verify JSON serialization
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal SARIF log: %v", err)
	}

	// Verify it's valid JSON
	var validated map[string]interface{}
	if err := json.Unmarshal(data, &validated); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// Verify required fields exist
	if validated["$schema"] == nil {
		t.Error("Missing $schema field")
	}
	if validated["version"] == nil {
		t.Error("Missing version field")
	}
	if validated["runs"] == nil {
		t.Error("Missing runs field")
	}
}

func TestMarkdownReporter_Report(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violations []domain.Violation
		wantContains []string
	}{
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
			wantContains: []string{
				"# Architecture Audit Report",
				"## Executive Summary",
				"Total Violations | 1",
				"## Detailed Violations",
				"D-01",
				"internal/domain/order.go:14",
			},
		},
		{
			name:       "no violations",
			violations: []domain.Violation{},
			wantContains: []string{
				"# Architecture Audit Report",
				"✅ **No violations found!**",
			},
		},
		{
			name: "multiple violations",
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
				{
					ID:          "D-02",
					RuleID:      "application-cannot-depend-on-infrastructure",
					Severity:    domain.SeverityWarning,
					File:        "internal/application/service.go",
					Line:        25,
					SourceLayer: "application",
					TargetLayer: "infrastructure",
					Import:      "github.com/example/app/internal/infrastructure/db",
					Message:     "Application should depend on ports",
				},
			},
			wantContains: []string{
				"Total Violations | 2",
				"D-01",
				"D-02",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewMarkdownReporter()
			report := reporter.buildMarkdownReport(tt.violations)

			for _, want := range tt.wantContains {
				if !strings.Contains(report, want) {
					t.Errorf("Report missing %q", want)
				}
			}
		})
	}
}

func TestMarkdownReporter_GenerateCommitMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		violations []domain.Violation
		want       string
	}{
		{
			name:       "no violations",
			violations: []domain.Violation{},
			want:       "docs: architecture audit - no violations found",
		},
		{
			name: "errors only",
			violations: []domain.Violation{
				{
					ID:       "D-01",
					Severity: domain.SeverityError,
				},
				{
					ID:       "D-02",
					Severity: domain.SeverityError,
				},
			},
			want: "docs: architecture audit - 2 errors, 0 warnings",
		},
		{
			name: "mixed severity",
			violations: []domain.Violation{
				{
					ID:       "D-01",
					Severity: domain.SeverityError,
				},
				{
					ID:       "D-02",
					Severity: domain.SeverityWarning,
				},
			},
			want: "docs: architecture audit - 1 errors, 1 warnings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GenerateCommitMessage(tt.violations)
			if got != tt.want {
				t.Errorf("GenerateCommitMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
