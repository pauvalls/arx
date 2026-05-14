package application

import (
	"context"
	"errors"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// mockConfigReader implements ports.ConfigReader for testing
type mockConfigReader struct {
	config      *domain.Config
	readErr     error
	validateErr error
}

func (m *mockConfigReader) Read(configPath string) (*domain.Config, error) {
	return m.config, m.readErr
}

func (m *mockConfigReader) Validate(config *domain.Config) error {
	return m.validateErr
}

// mockDetector implements ports.Detector for testing
type mockDetector struct {
	name         string
	detectResult bool
	detectErr    error
	extractDeps  []domain.Dependency
	extractErr   error
}

func (m *mockDetector) Name() string {
	return m.name
}

func (m *mockDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	return m.detectResult, m.detectErr
}

func (m *mockDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return m.extractDeps, m.extractErr
}

// mockReporter implements ports.Reporter for testing
type mockReporter struct {
	reportedViolations []domain.Violation
	reportedFormat     ports.OutputFormat
	reportErr          error
}

func (m *mockReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	m.reportedViolations = violations
	m.reportedFormat = format
	return m.reportErr
}

func TestLoadConfig_Success(t *testing.T) {
	expectedConfig := &domain.Config{
		Version: "1.0",
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	}
	reader := &mockConfigReader{config: expectedConfig}

	config, err := LoadConfig("arx.yaml", reader)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("LoadConfig() config.Version = %q, want %q", config.Version, "1.0")
	}
}

func TestLoadConfig_ReadError(t *testing.T) {
	reader := &mockConfigReader{readErr: errors.New("file not found")}

	_, err := LoadConfig("arx.yaml", reader)
	if err == nil {
		t.Errorf("LoadConfig() with read error should return error")
	}
}

func TestLoadConfig_ValidationError(t *testing.T) {
	reader := &mockConfigReader{
		config:      &domain.Config{Version: "1.0", Layers: []domain.Layer{}, Rules: []domain.Rule{}},
		validateErr: errors.New("invalid config"),
	}

	_, err := LoadConfig("arx.yaml", reader)
	if err == nil {
		t.Errorf("LoadConfig() with validation error should return error")
	}
}

func TestLoadConfig_NilReader(t *testing.T) {
	_, err := LoadConfig("arx.yaml", nil)
	if err == nil {
		t.Errorf("LoadConfig(nil reader) should return error")
	}
}

func TestLoadConfig_UsesPortInterface(t *testing.T) {
	var _ ports.ConfigReader = (*mockConfigReader)(nil)
}

func TestRunDetectors_Success(t *testing.T) {
	ctx := context.Background()
	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "main.go", SourceLine: 5, ImportPath: "fmt"},
		},
	}
	tsDetector := &mockDetector{
		name:         "typescript",
		detectResult: false,
	}

	deps, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{goDetector, tsDetector})
	if err != nil {
		t.Fatalf("RunDetectors() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("RunDetectors() returned %d deps, want 1", len(deps))
	}
}

func TestRunDetectors_MultipleApplicable(t *testing.T) {
	ctx := context.Background()
	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
		},
	}
	tsDetector := &mockDetector{
		name:         "typescript",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "app.ts", SourceLine: 1, ImportPath: "react"},
		},
	}

	deps, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{goDetector, tsDetector})
	if err != nil {
		t.Fatalf("RunDetectors() error = %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("RunDetectors() returned %d deps, want 2", len(deps))
	}
}

func TestRunDetectors_NoDetectors(t *testing.T) {
	ctx := context.Background()
	_, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{})
	if err == nil {
		t.Errorf("RunDetectors() with no detectors should return error")
	}
}

func TestRunDetectors_DetectError(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: false,
		detectErr:    errors.New("detection failed"),
	}

	_, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{detector})
	if err == nil {
		t.Errorf("RunDetectors() with detect error should return error")
	}
}

func TestRunDetectors_ExtractError(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractErr:   errors.New("extraction failed"),
	}

	_, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{detector})
	if err == nil {
		t.Errorf("RunDetectors() with extract error should return error")
	}
}

func TestRunDetectors_SkipsNilDetector(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps:  []domain.Dependency{},
	}

	deps, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{nil, detector})
	if err != nil {
		t.Fatalf("RunDetectors() error = %v", err)
	}

	if len(deps) != 0 {
		t.Errorf("RunDetectors() returned %d deps, want 0", len(deps))
	}
}

func TestRunDetectors_UsesPortInterface(t *testing.T) {
	var _ ports.Detector = (*mockDetector)(nil)
}

func TestEvaluateArchitecture_WithViolations(t *testing.T) {
	dependencies := []domain.Dependency{
		{
			SourceFile:    "internal/domain/user.go",
			SourceLine:    10,
			ImportPath:    "github.com/example/arx/internal/infrastructure/db",
			ResolvedLayer: "infrastructure",
		},
	}

	rules := []domain.Rule{
		{
			ID:          "domain-imports-infrastructure",
			From:        "domain",
			To:          []string{"infrastructure"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityError,
			Explanation: "Domain should not import infrastructure",
		},
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	violations := EvaluateArchitecture(dependencies, rules, layers)

	if len(violations) != 1 {
		t.Fatalf("EvaluateArchitecture() returned %d violations, want 1", len(violations))
	}

	if violations[0].RuleID != "domain-imports-infrastructure" {
		t.Errorf("EvaluateArchitecture() violation.RuleID = %q, want %q", violations[0].RuleID, "domain-imports-infrastructure")
	}

	// Message should be enriched with the explanation
	if violations[0].Message != "Domain should not import infrastructure" {
		t.Errorf("EvaluateArchitecture() violation.Message = %q, want %q", violations[0].Message, "Domain should not import infrastructure")
	}
}

func TestEvaluateArchitecture_NoViolations(t *testing.T) {
	dependencies := []domain.Dependency{
		{
			SourceFile:    "internal/application/service.go",
			SourceLine:    15,
			ImportPath:    "github.com/example/arx/internal/domain/user",
			ResolvedLayer: "domain",
		},
	}

	rules := []domain.Rule{
		{
			ID:       "domain-imports-infrastructure",
			From:     "domain",
			To:       []string{"infrastructure"},
			Type:     domain.RuleTypeCannot,
			Severity: domain.SeverityError,
		},
	}

	layers := []domain.Layer{
		{Name: "application", Paths: []string{"internal/application"}},
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	violations := EvaluateArchitecture(dependencies, rules, layers)

	if len(violations) != 0 {
		t.Errorf("EvaluateArchitecture() returned %d violations, want 0", len(violations))
	}
}

func TestEvaluateArchitecture_EnrichesWithBuiltinExplanation(t *testing.T) {
	dependencies := []domain.Dependency{
		{
			SourceFile:    "internal/domain/user.go",
			SourceLine:    10,
			ImportPath:    "github.com/example/arx/internal/infrastructure/db",
			ResolvedLayer: "infrastructure",
		},
	}

	rules := []domain.Rule{
		{
			ID:       "domain-imports-infrastructure",
			From:     "domain",
			To:       []string{"infrastructure"},
			Type:     domain.RuleTypeCannot,
			Severity: domain.SeverityError,
			// No Explanation set — should fall back to built-in
		},
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	violations := EvaluateArchitecture(dependencies, rules, layers)

	if len(violations) != 1 {
		t.Fatalf("EvaluateArchitecture() returned %d violations, want 1", len(violations))
	}

	// Should have the built-in explanation
	if violations[0].Message == "" {
		t.Errorf("EvaluateArchitecture() did not enrich violation message")
	}
}

func TestGenerateReport_Success(t *testing.T) {
	reporter := &mockReporter{}
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "R1",
			File:        "test.go",
			Line:        10,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Message:     "domain cannot depend on infrastructure",
		},
	}

	err := GenerateReport(violations, ports.OutputFormatTerminal, reporter)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	if len(reporter.reportedViolations) != 1 {
		t.Errorf("GenerateReport() reported %d violations, want 1", len(reporter.reportedViolations))
	}

	if reporter.reportedFormat != ports.OutputFormatTerminal {
		t.Errorf("GenerateReport() format = %q, want %q", reporter.reportedFormat, ports.OutputFormatTerminal)
	}
}

func TestGenerateReport_NilReporter(t *testing.T) {
	violations := []domain.Violation{}
	err := GenerateReport(violations, ports.OutputFormatJSON, nil)
	if err == nil {
		t.Errorf("GenerateReport(nil reporter) should return error")
	}
}

func TestGenerateReport_ReportError(t *testing.T) {
	reporter := &mockReporter{reportErr: errors.New("report failed")}
	violations := []domain.Violation{{ID: "D-01"}}

	err := GenerateReport(violations, ports.OutputFormatTerminal, reporter)
	if err == nil {
		t.Errorf("GenerateReport() with report error should return error")
	}
}

func TestGenerateReport_UsesPortInterface(t *testing.T) {
	var _ ports.Reporter = (*mockReporter)(nil)
}
