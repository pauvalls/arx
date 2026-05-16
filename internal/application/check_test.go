package application

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestRunDetectorsWithStatus_ReturnsStatusForAllDetectors(t *testing.T) {
	ctx := context.Background()
	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "main.go", SourceLine: 5, ImportPath: "fmt"},
			{SourceFile: "main.go", SourceLine: 6, ImportPath: "os"},
		},
	}
	tsDetector := &mockDetector{
		name:         "typescript",
		detectResult: false,
	}

	result, err := RunDetectorsWithStatus(ctx, "/test", []domain.Layer{}, []ports.Detector{goDetector, tsDetector})
	if err != nil {
		t.Fatalf("RunDetectorsWithStatus() error = %v", err)
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("RunDetectorsWithStatus() returned %d deps, want 2", len(result.Dependencies))
	}

	if len(result.Statuses) != 2 {
		t.Fatalf("RunDetectorsWithStatus() returned %d statuses, want 2", len(result.Statuses))
	}

	// Go detector should be applicable with 2 deps
	if result.Statuses[0].Name != "go" {
		t.Errorf("Status[0].Name = %q, want %q", result.Statuses[0].Name, "go")
	}
	if !result.Statuses[0].Applicable {
		t.Error("Status[0].Applicable should be true")
	}
	if result.Statuses[0].DepCount != 2 {
		t.Errorf("Status[0].DepCount = %d, want 2", result.Statuses[0].DepCount)
	}
	if result.Statuses[0].Error != "" {
		t.Errorf("Status[0].Error = %q, want empty", result.Statuses[0].Error)
	}

	// TypeScript detector should not be applicable
	if result.Statuses[1].Name != "typescript" {
		t.Errorf("Status[1].Name = %q, want %q", result.Statuses[1].Name, "typescript")
	}
	if result.Statuses[1].Applicable {
		t.Error("Status[1].Applicable should be false")
	}
	if result.Statuses[1].DepCount != 0 {
		t.Errorf("Status[1].DepCount = %d, want 0", result.Statuses[1].DepCount)
	}
}

func TestRunDetectorsWithStatus_CaptureDetectError(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: false,
		detectErr:    errors.New("no go.mod found"),
	}

	result, err := RunDetectorsWithStatus(ctx, "/test", []domain.Layer{}, []ports.Detector{detector})
	if err == nil {
		t.Fatal("RunDetectorsWithStatus() should return error for detect failure")
	}

	if len(result.Statuses) != 1 {
		t.Fatalf("RunDetectorsWithStatus() returned %d statuses, want 1", len(result.Statuses))
	}

	if result.Statuses[0].Error == "" {
		t.Error("Status[0].Error should contain the detection error")
	}
	if result.Statuses[0].Applicable {
		t.Error("Status[0].Applicable should be false on detect error")
	}
}

func TestRunDetectorsWithStatus_CaptureExtractError(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractErr:   errors.New("parse error"),
	}

	result, err := RunDetectorsWithStatus(ctx, "/test", []domain.Layer{}, []ports.Detector{detector})
	if err == nil {
		t.Fatal("RunDetectorsWithStatus() should return error for extract failure")
	}

	if len(result.Statuses) != 1 {
		t.Fatalf("RunDetectorsWithStatus() returned %d statuses, want 1", len(result.Statuses))
	}

	if result.Statuses[0].Error == "" {
		t.Error("Status[0].Error should contain the extraction error")
	}
	if result.Statuses[0].Applicable {
		// Applicable is set before extraction, so it should be true even if extraction fails
		t.Log("Note: Applicable is set before extraction runs")
	}
}

func TestRunDetectorsWithStatus_NoDetectors(t *testing.T) {
	ctx := context.Background()
	_, err := RunDetectorsWithStatus(ctx, "/test", []domain.Layer{}, []ports.Detector{})
	if err == nil {
		t.Errorf("RunDetectorsWithStatus() with no detectors should return error")
	}
}

func TestRunDetectorsWithStatus_SkipsNilDetector(t *testing.T) {
	ctx := context.Background()
	detector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps:  []domain.Dependency{},
	}

	result, err := RunDetectorsWithStatus(ctx, "/test", []domain.Layer{}, []ports.Detector{nil, detector})
	if err != nil {
		t.Fatalf("RunDetectorsWithStatus() error = %v", err)
	}

	// Nil detector slot should be empty (zero-value status)
	if result.Statuses[0].Name != "" {
		t.Errorf("Status[0].Name = %q, want empty for nil detector", result.Statuses[0].Name)
	}

	if result.Statuses[1].Name != "go" {
		t.Errorf("Status[1].Name = %q, want %q", result.Statuses[1].Name, "go")
	}
}

// mockBlockingDetector is a detector that blocks until a channel is closed.
// Used to test concurrent execution.
type mockBlockingDetector struct {
	mockDetector
	startSignal  chan struct{}
	finishSignal chan struct{}
}

func (m *mockBlockingDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	close(m.startSignal)
	select {
	case <-m.finishSignal:
		return m.detectResult, m.detectErr
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func TestRunDetectors_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()

	finish1 := make(chan struct{})
	finish2 := make(chan struct{})
	started1 := make(chan struct{})
	started2 := make(chan struct{})

	detector1 := &mockBlockingDetector{
		mockDetector: mockDetector{
			name:         "d1",
			detectResult: true,
			extractDeps: []domain.Dependency{
				{SourceFile: "a.go", SourceLine: 1, ImportPath: "fmt"},
			},
		},
		startSignal:  started1,
		finishSignal: finish1,
	}
	detector2 := &mockBlockingDetector{
		mockDetector: mockDetector{
			name:         "d2",
			detectResult: true,
			extractDeps: []domain.Dependency{
				{SourceFile: "b.go", SourceLine: 1, ImportPath: "os"},
			},
		},
		startSignal:  started2,
		finishSignal: finish2,
	}

	// Run in background
	resultCh := make(chan struct {
		deps []domain.Dependency
		err  error
	}, 1)
	go func() {
		deps, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{detector1, detector2})
		resultCh <- struct {
			deps []domain.Dependency
			err  error
		}{deps, err}
	}()

	// Wait for BOTH detectors to start — proves they run concurrently
	select {
	case <-started1:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("detector 1 did not start")
	}
	select {
	case <-started2:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("detector 2 did not start — detectors may not be running concurrently")
	}

	// Now let both finish
	close(finish1)
	close(finish2)

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("RunDetectors() error = %v", result.err)
		}
		if len(result.deps) != 2 {
			t.Errorf("RunDetectors() returned %d deps, want 2", len(result.deps))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunDetectors() did not complete")
	}
}

func TestRunDetectors_ErrorCancelsOthers(t *testing.T) {
	ctx := context.Background()

	finish := make(chan struct{})
	started := make(chan struct{})

	slowDetector := &mockBlockingDetector{
		mockDetector: mockDetector{name: "slow"},
		startSignal:  started,
		finishSignal: finish,
	}
	failingDetector := &mockDetector{
		name:         "failing",
		detectResult: false,
		detectErr:    errors.New("detection failed"),
	}

	// Run in background
	resultCh := make(chan struct {
		deps []domain.Dependency
		err  error
	}, 1)
	go func() {
		deps, err := RunDetectors(ctx, "/test", []domain.Layer{}, []ports.Detector{slowDetector, failingDetector})
		resultCh <- struct {
			deps []domain.Dependency
			err  error
		}{deps, err}
	}()

	// Wait for slow detector to start
	select {
	case <-started:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("slow detector did not start")
	}

	// The failing detector should cause overall error
	select {
	case result := <-resultCh:
		if result.err == nil {
			t.Fatal("RunDetectors() with failing detector should return error")
		}
		if len(result.deps) != 0 {
			t.Errorf("RunDetectors() returned %d deps, want 0", len(result.deps))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunDetectors() did not return error in time")
	}

	close(finish)
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
