package domain

import (
	"path/filepath"
	"time"
)

// ProjectSummary holds violation count breakdown for a single project.
type ProjectSummary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

// ProjectReport holds the result of auditing a single project in the workspace.
type ProjectReport struct {
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Status     string         `json:"status"` // "pass" or "fail"
	Violations []Violation    `json:"violations"`
	Summary    ProjectSummary `json:"summary"`
	Detectors  []interface{}  `json:"detectors,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Error      string         `json:"error,omitempty"`
}

// WorkspaceSummary aggregates results across all workspace projects.
type WorkspaceSummary struct {
	TotalProjects   int  `json:"total_projects"`
	FailedProjects  int  `json:"failed_projects"`
	TotalViolations int  `json:"total_violations"`
	Errors          int  `json:"errors"`
	Warnings        int  `json:"warnings"`
	Info            int  `json:"info"`
	Passed          bool `json:"passed"`
}

// WorkspaceReport is the top-level aggregated report for the workspace run.
type WorkspaceReport struct {
	Version  string            `json:"version"`
	Projects []ProjectReport   `json:"projects"`
	Summary  WorkspaceSummary  `json:"summary"`
}

// ProjectSummaryFromViolations computes severity counts from violations.
func ProjectSummaryFromViolations(violations []Violation) ProjectSummary {
	s := ProjectSummary{}
	for _, v := range violations {
		s.Total++
		switch v.Severity {
		case SeverityWarning:
			s.Warnings++
		case SeverityInfo:
			s.Info++
		default:
			s.Errors++
		}
	}
	return s
}

// NewProjectReport creates a ProjectReport from project path, violations, duration, and error.
// Name is derived from the path basename. Status is "pass" if no violations and no error.
func NewProjectReport(path string, violations []Violation, duration time.Duration, err error) ProjectReport {
	summary := ProjectSummaryFromViolations(violations)

	status := "pass"
	if len(violations) > 0 || err != nil {
		status = "fail"
	}

	errorStr := ""
	if err != nil {
		errorStr = err.Error()
	}

	return ProjectReport{
		Name:       filepath.Base(path),
		Path:       path,
		Status:     status,
		Violations: violations,
		Summary:    summary,
		DurationMs: duration.Milliseconds(),
		Error:      errorStr,
	}
}

// NewWorkspaceReport aggregates ProjectReports into a WorkspaceReport.
// Computes summary counts across all projects.
func NewWorkspaceReport(version string, projects []ProjectReport) WorkspaceReport {
	summary := WorkspaceSummary{
		TotalProjects:  len(projects),
		Passed:         true,
	}

	for _, p := range projects {
		if p.Status == "fail" || p.Error != "" {
			summary.FailedProjects++
			summary.Passed = false
		}
		summary.TotalViolations += p.Summary.Total
		summary.Errors += p.Summary.Errors
		summary.Warnings += p.Summary.Warnings
		summary.Info += p.Summary.Info
	}

	return WorkspaceReport{
		Version:  version,
		Projects: projects,
		Summary:  summary,
	}
}

// Passed returns true if all projects pass.
func (r *WorkspaceReport) Passed() bool {
	return r.Summary.Passed
}
