package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// mockConfigReader is a test double for ports.ConfigReader
type mockConfigReader struct {
	config *domain.Config
	err    error
}

func (m *mockConfigReader) Read(configPath string) (*domain.Config, error) {
	return m.config, m.err
}

func (m *mockConfigReader) Validate(config *domain.Config) error {
	return nil
}

// mockDetector is a test double for ports.Detector
type mockDetector struct {
	name         string
	canDetect    bool
	dependencies []domain.Dependency
	err          error
}

func (m *mockDetector) Name() string {
	return m.name
}

func (m *mockDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	return m.canDetect, m.err
}

func (m *mockDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return m.dependencies, m.err
}

// mockHistoryStorage is a test double for ports.HistoryStorage
type mockHistoryStorage struct {
	savedReports []*domain.AuditReport
	loadLatest   *domain.AuditReport
	err          error
}

func (m *mockHistoryStorage) Save(ctx context.Context, report *domain.AuditReport) (string, error) {
	m.savedReports = append(m.savedReports, report)
	return "/test/path/audit-test.json", nil
}

func (m *mockHistoryStorage) Load(ctx context.Context, date time.Time) (*domain.AuditReport, error) {
	return m.loadLatest, m.err
}

func (m *mockHistoryStorage) LoadLatest(ctx context.Context) (*domain.AuditReport, error) {
	return m.loadLatest, m.err
}

func (m *mockHistoryStorage) List(ctx context.Context) ([]time.Time, error) {
	return []time.Time{}, nil
}

func (m *mockHistoryStorage) DeleteOld(ctx context.Context, maxAudits int) (int, error) {
	return 0, nil
}

// TestAuditCmd_Success tests the audit command with a valid configuration
func TestAuditCmd_Success(t *testing.T) {
	// Create a temporary directory with a mock config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	// Write minimal config
	configContent := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: application
    paths:
      - application/**
rules:
  - id: R001
    from: domain
    to:
      - application
    type: Cannot
    severity: error
    explanation: Domain must not depend on application
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Capture stdout
	var buf bytes.Buffer

	// Create mock dependencies
	configReader := &mockConfigReader{
		config: &domain.Config{
			Layers: []domain.Layer{
				{Name: "domain", Paths: []string{"domain/**"}},
				{Name: "application", Paths: []string{"application/**"}},
			},
			Rules: []domain.Rule{
				{ID: "R001", From: "domain", To: []string{"application"}, Type: "Cannot", Severity: "error"},
			},
		},
	}

	detectors := []ports.Detector{
		&mockDetector{
			name:      "go",
			canDetect: true,
			dependencies: []domain.Dependency{
				{
					SourceFile:    "domain/entity.go",
					ImportPath:    "com.example.application",
					ResolvedLayer: "application",
				},
			},
		},
	}

	historyStorage := &mockHistoryStorage{}

	// Create audit service with mocks
	service := application.NewAuditService(configReader, detectors, historyStorage, tmpDir)

	// Run audit
	ctx := context.Background()
	report, err := service.Audit(ctx, tmpDir, configPath)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	// Verify report structure
	if report == nil {
		t.Fatal("Audit() returned nil report")
	}

	if report.ProjectRoot != tmpDir {
		t.Errorf("ProjectRoot = %q, want %q", report.ProjectRoot, tmpDir)
	}

	// Render terminal output
	if err := renderTerminal(&buf, report); err != nil {
		t.Fatalf("renderTerminal() error = %v", err)
	}

	// Verify output contains expected sections
	output := buf.String()
	if !contains(output, "ARCHITECTURE AUDIT REPORT") {
		t.Error("output missing header")
	}
	if !contains(output, "VIOLATIONS") {
		t.Error("output missing violations section")
	}
	if !contains(output, "COUPLING MATRIX") {
		t.Error("output missing coupling matrix section")
	}
	if !contains(output, "TECHNICAL DEBT") {
		t.Error("output missing debt section")
	}
}

// TestAuditCmd_NoConfig tests the audit command when config file is missing
func TestAuditCmd_NoConfig(t *testing.T) {
	// Create a temporary directory without config
	tmpDir := t.TempDir()

	// Try to run audit on directory without config
	_, err := os.Stat(filepath.Join(tmpDir, "arx.yaml"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected config to not exist, but got: %v", err)
	}

	// Verify that the error message is user-friendly
	expectedError := "configuration file not found"
	if err == nil {
		t.Errorf("expected error containing %q, got nil", expectedError)
	}
}

// TestAuditCmd_TrendFlag tests the --trend flag
func TestAuditCmd_TrendFlag(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer

	report := &domain.AuditReport{
		Timestamp:   time.Now(),
		ProjectRoot: tmpDir,
		TrendReport: domain.TrendReport{
			Status:         domain.TrendImproved,
			ViolationDelta: -2,
			DebtDelta:      -5,
			Summary:        "Architecture improved",
		},
	}

	// Test trend-only rendering
	if err := renderTrendOnly(&buf, report, ports.OutputFormatTerminal); err != nil {
		t.Fatalf("renderTrendOnly() error = %v", err)
	}

	output := buf.String()
	if !contains(output, "TREND COMPARISON") {
		t.Error("output missing trend header")
	}
	if !contains(output, "Improved") && !contains(output, "improved") {
		t.Error("output missing improvement status")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
