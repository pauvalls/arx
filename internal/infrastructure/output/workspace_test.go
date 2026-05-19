package output

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestWorkspaceTerminal_Mixed(t *testing.T) {
	// Disable colors for test
	SetNoColor(true)
	defer SetNoColor(false)

	report := createMixedReport()
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "FAIL") {
		t.Errorf("Terminal output should contain FAIL for failing project")
	}
	if !strings.Contains(output, "PASS") {
		t.Errorf("Terminal output should contain PASS for passing project")
	}
	if !strings.Contains(output, "1 of 2") {
		t.Errorf("Terminal output should contain summary line with '1 of 2'")
	}
}

func TestWorkspaceTerminal_AllPass(t *testing.T) {
	SetNoColor(true)
	defer SetNoColor(false)

	report := createAllPassReport()
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "FAIL") {
		t.Errorf("All-pass output should not contain FAIL")
	}
	if !strings.Contains(output, "PASS") {
		t.Errorf("All-pass output should contain 'PASS', got: %s", output)
	}
	if !strings.Contains(output, "0") {
		t.Errorf("All-pass output should show 0 values, got: %s", output)
	}
}

func TestWorkspaceTerminal_Empty(t *testing.T) {
	SetNoColor(true)
	defer SetNoColor(false)

	report := domain.NewWorkspaceReport("1", []domain.ProjectReport{})
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No projects") {
		t.Errorf("Empty output should contain 'No projects', got: %s", output)
	}
}

func TestWorkspaceTerminal_SingleProject(t *testing.T) {
	SetNoColor(true)
	defer SetNoColor(false)

	report := createSingleProjectReport()
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "p1") {
		t.Errorf("Output should contain project name 'p1'")
	}
}

func TestWorkspaceTerminal_Verbose(t *testing.T) {
	SetNoColor(true)
	defer SetNoColor(false)

	report := createMixedReport()
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.RenderVerbose(&report, &buf)
	if err != nil {
		t.Fatalf("RenderVerbose() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ERROR") && !strings.Contains(output, "WARN") {
		t.Errorf("Verbose output should contain violation details with severity labels, got: %s", output)
	}
	if !strings.Contains(output, "no-domain-to-infra") {
		t.Errorf("Verbose output should contain rule IDs, got: %s", output)
	}
}

func TestWorkspaceJSON_Mixed(t *testing.T) {
	report := createMixedReport()
	reporter := NewWorkspaceJSONReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v\nOutput: %s", err, output)
	}

	// Verify structure
	if _, ok := parsed["version"]; !ok {
		t.Error("JSON output missing 'version'")
	}
	projects, ok := parsed["projects"].([]interface{})
	if !ok {
		t.Fatal("JSON output 'projects' is not an array")
	}
	if len(projects) != 2 {
		t.Errorf("JSON projects count = %d, want 2", len(projects))
	}

	summary, ok := parsed["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON output missing 'summary' object")
	}
	if summary["passed"] != false {
		t.Errorf("JSON summary.passed = %v, want false", summary["passed"])
	}
	if summary["failed_projects"] != float64(1) {
		t.Errorf("JSON summary.failed_projects = %v, want 1", summary["failed_projects"])
	}

	// Check first project has violations
	p0 := projects[0].(map[string]interface{})
	if p0["status"] != "fail" {
		t.Errorf("First project status = %v, want 'fail'", p0["status"])
	}
}

func TestWorkspaceJSON_AllPass(t *testing.T) {
	report := createAllPassReport()
	reporter := NewWorkspaceJSONReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	summary := parsed["summary"].(map[string]interface{})
	if summary["passed"] != true {
		t.Errorf("JSON summary.passed = %v, want true", summary["passed"])
	}
	if summary["failed_projects"] != float64(0) {
		t.Errorf("JSON summary.failed_projects = %v, want 0", summary["failed_projects"])
	}
}

func TestWorkspaceJSON_Violations(t *testing.T) {
	report := createMixedReport()
	reporter := NewWorkspaceJSONReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Parse first project's violations
	var parsed map[string]interface{}
	json.Unmarshal([]byte(buf.String()), &parsed)
	projects := parsed["projects"].([]interface{})
	p0 := projects[0].(map[string]interface{})

	violations, ok := p0["violations"].([]interface{})
	if !ok {
		t.Fatal("Project violations is not an array")
	}
	if len(violations) == 0 {
		t.Fatal("Failing project should have violations")
	}

	v0 := violations[0].(map[string]interface{})
	if v0["severity"] != "error" {
		t.Errorf("Violation severity = %v, want 'error'", v0["severity"])
	}
	if v0["rule_id"] == "" {
		t.Error("Violation rule_id should not be empty")
	}
	if v0["file"] == "" {
		t.Error("Violation file should not be empty")
	}
}

func TestWorkspaceJSON_Empty(t *testing.T) {
	report := domain.NewWorkspaceReport("1", []domain.ProjectReport{})
	reporter := NewWorkspaceJSONReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	projects := parsed["projects"].([]interface{})
	if len(projects) != 0 {
		t.Errorf("Empty report should have 0 projects, got %d", len(projects))
	}

	summary := parsed["summary"].(map[string]interface{})
	if summary["total_projects"] != float64(0) {
		t.Errorf("summary.total_projects = %v, want 0", summary["total_projects"])
	}
}

func TestWorkspaceTerminal_NoColor(t *testing.T) {
	// Save original noColor state and restore after test
	origNoColor := GetNoColor()
	defer SetNoColor(origNoColor)

	// Set NO_COLOR via env
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	// Re-init noColor (called automatically in init())
	// We need to reset it manually since init() already ran
	SetNoColor(true)

	report := createMixedReport()
	reporter := NewWorkspaceTerminalReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Should not contain ANSI escape codes
	if strings.Contains(output, "\033[") {
		t.Errorf("NO_COLOR output should not contain ANSI escape codes")
	}
}

func TestWorkspaceJSON_OutputFormat(t *testing.T) {
	report := createMixedReport()
	reporter := NewWorkspaceJSONReporter()

	var buf strings.Builder
	err := reporter.Render(&report, &buf)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	// Check that it's pretty-printed (contains newlines and indentation)
	if !strings.Contains(output, "\n") {
		t.Errorf("JSON output should be pretty-printed with newlines")
	}
	if !strings.Contains(output, "  ") {
		t.Errorf("JSON output should be indented with spaces")
	}
}

// Test helpers

func createMixedReport() domain.WorkspaceReport {
	dur := 100 * time.Millisecond
	p1 := domain.NewProjectReport("/projects/services/auth", []domain.Violation{
		{ID: "V1", RuleID: "no-domain-to-infra", Severity: domain.SeverityError, File: "domain/main.go", Line: 10, SourceLayer: "domain", TargetLayer: "infra"},
		{ID: "V2", RuleID: "app-cannot-infra", Severity: domain.SeverityWarning, File: "app/service.go", Line: 20, SourceLayer: "application", TargetLayer: "infrastructure"},
		{ID: "V3", RuleID: "some-rule", Severity: domain.SeverityError, File: "domain/model.go", Line: 5, SourceLayer: "domain", TargetLayer: "infra"},
	}, dur, nil)
	p2 := domain.NewProjectReport("/projects/libs/shared", []domain.Violation{}, dur/2, nil)

	return domain.NewWorkspaceReport("1", []domain.ProjectReport{p1, p2})
}

func createAllPassReport() domain.WorkspaceReport {
	p1 := domain.NewProjectReport("/projects/services/auth", []domain.Violation{}, 0, nil)
	p2 := domain.NewProjectReport("/projects/libs/shared", []domain.Violation{}, 0, nil)
	return domain.NewWorkspaceReport("1", []domain.ProjectReport{p1, p2})
}

func createSingleProjectReport() domain.WorkspaceReport {
	p1 := domain.NewProjectReport("/projects/p1", []domain.Violation{}, 0, nil)
	return domain.NewWorkspaceReport("1", []domain.ProjectReport{p1})
}
