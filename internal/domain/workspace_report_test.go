package domain

import (
	"errors"
	"testing"
	"time"
)

func TestProjectSummaryFromViolations(t *testing.T) {
	tests := []struct {
		name       string
		violations []Violation
		want       ProjectSummary
	}{
		{
			name:       "no violations",
			violations: []Violation{},
			want:       ProjectSummary{Total: 0, Errors: 0, Warnings: 0, Info: 0},
		},
		{
			name: "mixed severities",
			violations: []Violation{
				{ID: "V1", Severity: SeverityError},
				{ID: "V2", Severity: SeverityWarning},
				{ID: "V3", Severity: SeverityInfo},
				{ID: "V4", Severity: SeverityError},
			},
			want: ProjectSummary{Total: 4, Errors: 2, Warnings: 1, Info: 1},
		},
		{
			name: "all errors",
			violations: []Violation{
				{ID: "V1", Severity: SeverityError},
				{ID: "V2", Severity: SeverityError},
				{ID: "V3", Severity: SeverityError},
			},
			want: ProjectSummary{Total: 3, Errors: 3, Warnings: 0, Info: 0},
		},
		{
			name: "all warnings",
			violations: []Violation{
				{ID: "V1", Severity: SeverityWarning},
			},
			want: ProjectSummary{Total: 1, Errors: 0, Warnings: 1, Info: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectSummaryFromViolations(tt.violations)
			if got != tt.want {
				t.Errorf("ProjectSummaryFromViolations() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNewProjectReport(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		path       string
		violations []Violation
		duration   time.Duration
		err        error
		wantName   string
		wantStatus string
		wantError  string
		wantTotal  int
	}{
		{
			name:       "pass — no violations",
			path:       "/home/user/projects/services/auth",
			violations: []Violation{},
			duration:   now.Sub(now),
			err:        nil,
			wantName:   "auth",
			wantStatus: "pass",
			wantTotal:  0,
		},
		{
			name: "fail — has violations",
			path: "/home/user/projects/services/api",
			violations: []Violation{
				{ID: "V1", Severity: SeverityError},
				{ID: "V2", Severity: SeverityWarning},
			},
			duration:   now.Sub(now),
			err:        nil,
			wantName:   "api",
			wantStatus: "fail",
			wantTotal:  2,
		},
		{
			name:       "error — has error even without violations",
			path:       "/home/user/projects/libs/shared",
			violations: []Violation{},
			duration:   now.Sub(now),
			err:        errors.New("detection failed"),
			wantName:   "shared",
			wantStatus: "fail",
			wantError:  "detection failed",
			wantTotal:  0,
		},
		{
			name:     "error with violations",
			path:     "/home/user/projects/broken",
			violations: []Violation{
				{ID: "V1", Severity: SeverityError},
			},
			duration:   now.Sub(now),
			err:        errors.New("something went wrong"),
			wantName:   "broken",
			wantStatus: "fail",
			wantError:  "something went wrong",
			wantTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := NewProjectReport(tt.path, tt.violations, tt.duration, tt.err)

			if pr.Name != tt.wantName {
				t.Errorf("NewProjectReport().Name = %q, want %q", pr.Name, tt.wantName)
			}
			if pr.Path != tt.path {
				t.Errorf("NewProjectReport().Path = %q, want %q", pr.Path, tt.path)
			}
			if pr.Status != tt.wantStatus {
				t.Errorf("NewProjectReport().Status = %q, want %q", pr.Status, tt.wantStatus)
			}
			if tt.wantError != "" && pr.Error != tt.wantError {
				t.Errorf("NewProjectReport().Error = %q, want %q", pr.Error, tt.wantError)
			}
			if tt.wantError == "" && pr.Error != "" {
				t.Errorf("NewProjectReport().Error = %q, want empty", pr.Error)
			}
			if pr.Summary.Total != tt.wantTotal {
				t.Errorf("NewProjectReport().Summary.Total = %d, want %d", pr.Summary.Total, tt.wantTotal)
			}
			wantMs := tt.duration.Milliseconds()
			if pr.DurationMs != wantMs {
				t.Errorf("NewProjectReport().DurationMs = %d, want %d", pr.DurationMs, wantMs)
			}
		})
	}
}

func TestNewWorkspaceReport(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		version  string
		projects []ProjectReport
		want     WorkspaceReport
	}{
		{
			name:    "all projects pass",
			version: "1",
			projects: []ProjectReport{
				NewProjectReport("/p1", []Violation{}, now.Sub(now), nil),
				NewProjectReport("/p2", []Violation{}, now.Sub(now), nil),
				NewProjectReport("/p3", []Violation{}, now.Sub(now), nil),
			},
			want: WorkspaceReport{
				Version: "1",
				Projects: []ProjectReport{
					{Name: "p1", Path: "/p1", Status: "pass", Summary: ProjectSummary{Total: 0, Errors: 0, Warnings: 0, Info: 0}},
					{Name: "p2", Path: "/p2", Status: "pass", Summary: ProjectSummary{Total: 0, Errors: 0, Warnings: 0, Info: 0}},
					{Name: "p3", Path: "/p3", Status: "pass", Summary: ProjectSummary{Total: 0, Errors: 0, Warnings: 0, Info: 0}},
				},
				Summary: WorkspaceSummary{
					TotalProjects:   3,
					FailedProjects:  0,
					TotalViolations: 0,
					Errors:          0,
					Warnings:        0,
					Info:            0,
					Passed:          true,
				},
			},
		},
		{
			name:    "one project fails",
			version: "1",
			projects: []ProjectReport{
				NewProjectReport("/p1", []Violation{}, now.Sub(now), nil),
				NewProjectReport("/p2", []Violation{{ID: "V1", Severity: SeverityError}}, now.Sub(now), nil),
				NewProjectReport("/p3", []Violation{}, now.Sub(now), nil),
			},
			want: WorkspaceReport{
				Version: "1",
				Summary: WorkspaceSummary{
					TotalProjects:   3,
					FailedProjects:  1,
					TotalViolations: 1,
					Errors:          1,
					Warnings:        0,
					Info:            0,
					Passed:          false,
				},
			},
		},
		{
			name:    "mixed violations across projects",
			version: "1",
			projects: []ProjectReport{
				NewProjectReport("/p1", []Violation{
					{ID: "V1", Severity: SeverityError},
					{ID: "V2", Severity: SeverityWarning},
				}, now.Sub(now), nil),
				NewProjectReport("/p2", []Violation{
					{ID: "V3", Severity: SeverityInfo},
				}, now.Sub(now), nil),
				NewProjectReport("/p3", []Violation{}, now.Sub(now), nil),
			},
			want: WorkspaceReport{
				Version: "1",
				Summary: WorkspaceSummary{
					TotalProjects:   3,
					FailedProjects:  2,
					TotalViolations: 3,
					Errors:          1,
					Warnings:        1,
					Info:            1,
					Passed:          false,
				},
			},
		},
		{
			name:     "empty projects list",
			version:  "1",
			projects: []ProjectReport{},
			want: WorkspaceReport{
				Version:  "1",
				Projects: []ProjectReport{},
				Summary: WorkspaceSummary{
					TotalProjects:   0,
					FailedProjects:  0,
					TotalViolations: 0,
					Errors:          0,
					Warnings:        0,
					Info:            0,
					Passed:          true,
				},
			},
		},
		{
			name:    "failing project with error counts as failing",
			version: "1",
			projects: []ProjectReport{
				NewProjectReport("/p1", []Violation{}, now.Sub(now), nil),
				NewProjectReport("/p2", []Violation{}, now.Sub(now), errors.New("oops")),
				NewProjectReport("/p3", []Violation{}, now.Sub(now), nil),
			},
			want: WorkspaceReport{
				Version: "1",
				Summary: WorkspaceSummary{
					TotalProjects:   3,
					FailedProjects:  1,
					TotalViolations: 0,
					Errors:          0,
					Warnings:        0,
					Info:            0,
					Passed:          false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewWorkspaceReport(tt.version, tt.projects)

			if got.Version != tt.want.Version {
				t.Errorf("NewWorkspaceReport().Version = %q, want %q", got.Version, tt.want.Version)
			}

			if len(got.Projects) != len(tt.want.Projects) {
				// For cases where we don't check every project detail, just check counts
				if len(got.Projects) != len(tt.projects) {
					t.Errorf("NewWorkspaceReport() projects count = %d, want %d", len(got.Projects), len(tt.projects))
				}
			}

			if got.Summary != tt.want.Summary {
				t.Errorf("NewWorkspaceReport().Summary = %+v, want %+v", got.Summary, tt.want.Summary)
			}
		})
	}
}

func TestWorkspaceReport_Passed(t *testing.T) {
	tests := []struct {
		name     string
		report   WorkspaceReport
		wantPass bool
	}{
		{
			name: "all pass",
			report: WorkspaceReport{
				Summary: WorkspaceSummary{Passed: true},
			},
			wantPass: true,
		},
		{
			name: "some fail",
			report: WorkspaceReport{
				Summary: WorkspaceSummary{Passed: false},
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.report.Passed(); got != tt.wantPass {
				t.Errorf("WorkspaceReport.Passed() = %v, want %v", got, tt.wantPass)
			}
		})
	}
}
