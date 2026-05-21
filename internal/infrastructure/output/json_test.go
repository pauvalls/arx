package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func captureJSONReporterOutput(t *testing.T, f func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestJSONReporter_ReportAudit_IncludesCouplingMatrix(t *testing.T) {
	reporter := NewJSONReporter()
	report := &domain.AuditReport{
		ProjectRoot: "/test",
		Violations: []domain.Violation{
			{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
		},
		CouplingMatrix: domain.CouplingMatrix{
			FromTo: map[string]map[string]int{
				"domain": {"infrastructure": 5},
			},
		},
		DebtScore: domain.NewDebtScore(),
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportAudit(report); err != nil {
			t.Fatalf("ReportAudit failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["coupling_matrix"]; !ok {
		t.Error("Missing 'coupling_matrix' field in JSON output")
	}

	cm, ok := result["coupling_matrix"].(map[string]interface{})
	if !ok {
		t.Fatal("coupling_matrix is not an object")
	}

	fromTo, ok := cm["from_to"].(map[string]interface{})
	if !ok {
		t.Fatal("from_to is not an object")
	}

	domainMap, ok := fromTo["domain"].(map[string]interface{})
	if !ok {
		t.Fatal("domain entry is not an object")
	}

	if domainMap["infrastructure"] != float64(5) {
		t.Errorf("Expected infrastructure count 5, got %v", domainMap["infrastructure"])
	}
}

func TestJSONReporter_ReportAudit_IncludesDebtScore(t *testing.T) {
	reporter := NewJSONReporter()
	debt := domain.NewDebtScore()
	debt.BySeverity["error"] = 3
	debt.BySeverity["warning"] = 2
	debt.Calculate()

	report := &domain.AuditReport{
		ProjectRoot: "/test",
		Violations:  []domain.Violation{},
		DebtScore:   debt,
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportAudit(report); err != nil {
			t.Fatalf("ReportAudit failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["debt_score"]; !ok {
		t.Error("Missing 'debt_score' field in JSON output")
	}

	ds, ok := result["debt_score"].(map[string]interface{})
	if !ok {
		t.Fatal("debt_score is not an object")
	}

	if ds["total"] != float64(11) {
		t.Errorf("Expected debt total 11, got %v", ds["total"])
	}
}

func TestJSONReporter_ReportAudit_IncludesTrendReport(t *testing.T) {
	reporter := NewJSONReporter()
	report := &domain.AuditReport{
		ProjectRoot: "/test",
		Violations:  []domain.Violation{},
		TrendReport: domain.TrendReport{
			Status:         domain.TrendImproved,
			ViolationDelta: -3,
			DebtDelta:      -5,
			Summary:        "Architecture improved",
		},
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportAudit(report); err != nil {
			t.Fatalf("ReportAudit failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["trend_report"]; !ok {
		t.Error("Missing 'trend_report' field in JSON output")
	}

	tr, ok := result["trend_report"].(map[string]interface{})
	if !ok {
		t.Fatal("trend_report is not an object")
	}

	if tr["status"] != "improved" {
		t.Errorf("Expected status 'improved', got %v", tr["status"])
	}
}

func TestJSONReporter_ReportWithContext_IncludesDetectors(t *testing.T) {
	reporter := NewJSONReporter()
	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
	}
	detectors := []application.DetectorStatus{
		{Name: "go", Applicable: true, DepCount: 42},
		{Name: "typescript", Applicable: false, DepCount: 0, Error: "no tsconfig.json"},
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportWithContext(violations, detectors); err != nil {
			t.Fatalf("ReportWithContext failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["detectors"]; !ok {
		t.Error("Missing 'detectors' field in JSON output")
	}

	dets, ok := result["detectors"].([]interface{})
	if !ok {
		t.Fatal("detectors is not an array")
	}

	if len(dets) != 2 {
		t.Fatalf("Expected 2 detectors, got %d", len(dets))
	}

	first := dets[0].(map[string]interface{})
	if first["name"] != "go" {
		t.Errorf("Expected first detector name 'go', got %v", first["name"])
	}
	if first["applicable"] != true {
		t.Error("Expected first detector applicable=true")
	}
	if first["dep_count"] != float64(42) {
		t.Errorf("Expected first detector dep_count 42, got %v", first["dep_count"])
	}

	second := dets[1].(map[string]interface{})
	if second["name"] != "typescript" {
		t.Errorf("Expected second detector name 'typescript', got %v", second["name"])
	}
	if second["error"] != "no tsconfig.json" {
		t.Errorf("Expected second detector error, got %v", second["error"])
	}
}

func TestJSONReporter_ReportWithContext_EmptyDetectors(t *testing.T) {
	reporter := NewJSONReporter()
	violations := []domain.Violation{}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportWithContext(violations, nil); err != nil {
			t.Fatalf("ReportWithContext failed: %v", err)
		}
	})

	// detectors should be omitted when empty (omitempty)
	if strings.Contains(output, `"detectors"`) {
		t.Error("Empty detectors should be omitted from JSON output")
	}
}

func TestJSONReporter_Report_WithPerformance_IncludesField(t *testing.T) {
	reporter := NewJSONReporter()
	perf := &domain.PerformanceReport{
		Total: 69300000, // 69.3ms
		Phases: []domain.PhaseTiming{
			{Name: "Go", Duration: 45200000},
			{Name: "Python", Duration: 12100000},
		},
	}
	reporter.SetPerformance(perf)

	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.Report(violations, ports.OutputFormatJSON); err != nil {
			t.Fatalf("Report failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["performance"]; !ok {
		t.Fatal("Missing 'performance' field in JSON output when SetPerformance was called")
	}

	perfField, ok := result["performance"].(map[string]interface{})
	if !ok {
		t.Fatal("performance is not an object")
	}

	totalNs, ok := perfField["total_duration_ns"].(float64)
	if !ok {
		t.Fatal("total_duration_ns is not a number")
	}
	if totalNs != 69300000 {
		t.Errorf("total_duration_ns = %v, want 69300000", totalNs)
	}

	phases, ok := perfField["phases"].([]interface{})
	if !ok {
		t.Fatal("phases is not an array")
	}
	if len(phases) != 2 {
		t.Fatalf("phases count = %d, want 2", len(phases))
	}

	first := phases[0].(map[string]interface{})
	if first["name"] != "Go" {
		t.Errorf("first phase name = %v, want 'Go'", first["name"])
	}
	durationNs, ok := first["duration_ns"].(float64)
	if !ok {
		t.Fatal("phase duration_ns is not a number")
	}
	if durationNs != 45200000 {
		t.Errorf("first phase duration_ns = %v, want 45200000", durationNs)
	}
}

func TestJSONReporter_Report_WithoutPerformance_OmitsField(t *testing.T) {
	reporter := NewJSONReporter()

	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.Report(violations, ports.OutputFormatJSON); err != nil {
			t.Fatalf("Report failed: %v", err)
		}
	})

	if strings.Contains(output, `"performance"`) {
		t.Error("'performance' field should be omitted when not set")
	}
}

func TestJSONReporter_ReportWithContext_WithPerformance_IncludesField(t *testing.T) {
	reporter := NewJSONReporter()
	perf := &domain.PerformanceReport{
		Total: 50000000,
		Phases: []domain.PhaseTiming{
			{Name: "Go", Duration: 30000000},
		},
	}
	reporter.SetPerformance(perf)

	violations := []domain.Violation{
		{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportWithContext(violations, nil); err != nil {
			t.Fatalf("ReportWithContext failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if _, ok := result["performance"]; !ok {
		t.Fatal("Missing 'performance' field in ReportWithContext output when SetPerformance was called")
	}

	phases, ok := result["performance"].(map[string]interface{})["phases"].([]interface{})
	if !ok {
		t.Fatal("performance.phases is not an array")
	}
	if len(phases) != 1 {
		t.Fatalf("performance.phases count = %d, want 1", len(phases))
	}
}

func TestJSONReporter_ReportAudit_BackwardCompatible(t *testing.T) {
	reporter := NewJSONReporter()
	report := &domain.AuditReport{
		ProjectRoot: "/test",
		Violations: []domain.Violation{
			{ID: "v1", RuleID: "r1", File: "a.go", Line: 1, Severity: domain.SeverityError},
		},
		CouplingMatrix: domain.NewCouplingMatrix(),
		DebtScore:      domain.NewDebtScore(),
	}

	output := captureJSONReporterOutput(t, func() {
		if err := reporter.ReportAudit(report); err != nil {
			t.Fatalf("ReportAudit failed: %v", err)
		}
	})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Existing fields must still be present
	if _, ok := result["version"]; !ok {
		t.Error("Missing 'version' field")
	}
	if _, ok := result["tool"]; !ok {
		t.Error("Missing 'tool' field")
	}
	if _, ok := result["violations"]; !ok {
		t.Error("Missing 'violations' field")
	}
	if _, ok := result["summary"]; !ok {
		t.Error("Missing 'summary' field")
	}
}
