package ports

import (
	"context"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

// MockDetector implements Detector interface for testing
type MockDetector struct {
	name           string
	detectResult   bool
	detectErr      error
	extractDeps    []domain.Dependency
	extractErr     error
}

func (m *MockDetector) Name() string {
	return m.name
}

func (m *MockDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	return m.detectResult, m.detectErr
}

func (m *MockDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return m.extractDeps, m.extractErr
}

// TestDetectorInterface verifies the Detector interface can be implemented
func TestDetectorInterface(t *testing.T) {
	var _ Detector = (*MockDetector)(nil)

	detector := &MockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{
				SourceFile: "main.go",
				SourceLine: 5,
				ImportPath: "fmt",
			},
		},
	}

	ctx := context.Background()

	// Test Name
	if detector.Name() != "go" {
		t.Errorf("Name() = %q, want %q", detector.Name(), "go")
	}

	// Test Detect
	detected, err := detector.Detect(ctx, "/test/project")
	if err != nil {
		t.Errorf("Detect() error = %v", err)
	}
	if !detected {
		t.Errorf("Detect() = %v, want %v", detected, true)
	}

	// Test ExtractImports
	deps, err := detector.ExtractImports(ctx, "/test/project", []domain.Layer{})
	if err != nil {
		t.Errorf("ExtractImports() error = %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("ExtractImports() returned %d deps, want 1", len(deps))
	}
}

// MockConfigReader implements ConfigReader interface for testing
type MockConfigReader struct {
	config    *domain.Config
	readErr   error
	validateErr error
}

func (m *MockConfigReader) Read(configPath string) (*domain.Config, error) {
	return m.config, m.readErr
}

func (m *MockConfigReader) Validate(config *domain.Config) error {
	return m.validateErr
}

// TestConfigReaderInterface verifies the ConfigReader interface can be implemented
func TestConfigReaderInterface(t *testing.T) {
	var _ ConfigReader = (*MockConfigReader)(nil)

	reader := &MockConfigReader{
		config: &domain.Config{
			Version: "1.0.0",
			Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
			Rules:   []domain.Rule{},
		},
	}

	config, err := reader.Read("arx.yaml")
	if err != nil {
		t.Errorf("Read() error = %v", err)
	}
	if config == nil {
		t.Errorf("Read() returned nil config")
	}
	if config.Version != "1.0.0" {
		t.Errorf("Read() config.Version = %q, want %q", config.Version, "1.0.0")
	}
}

// MockReporter implements Reporter interface for testing
type MockReporter struct {
	reportErr error
}

func (m *MockReporter) Report(violations []domain.Violation, format OutputFormat) error {
	return m.reportErr
}

// TestReporterInterface verifies the Reporter interface can be implemented
func TestReporterInterface(t *testing.T) {
	var _ Reporter = (*MockReporter)(nil)

	reporter := &MockReporter{}

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

	// Test Terminal format
	err := reporter.Report(violations, OutputFormatTerminal)
	if err != nil {
		t.Errorf("Report(terminal) error = %v", err)
	}

	// Test JSON format
	err = reporter.Report(violations, OutputFormatJSON)
	if err != nil {
		t.Errorf("Report(json) error = %v", err)
	}
}

// TestOutputFormatValues verifies OutputFormat enum values
func TestOutputFormatValues(t *testing.T) {
	if OutputFormatTerminal != "terminal" {
		t.Errorf("OutputFormatTerminal = %q, want %q", OutputFormatTerminal, "terminal")
	}
	if OutputFormatJSON != "json" {
		t.Errorf("OutputFormatJSON = %q, want %q", OutputFormatJSON, "json")
	}
}

// MockFileWriter implements FileWriter interface for testing
type MockFileWriter struct {
	files  map[string][]byte
	exists map[string]bool
}

func (m *MockFileWriter) Write(path string, content []byte) error {
	if m.files == nil {
		m.files = make(map[string][]byte)
	}
	m.files[path] = content
	return nil
}

func (m *MockFileWriter) Exists(path string) bool {
	if m.exists == nil {
		return false
	}
	return m.exists[path]
}

// TestFileWriterInterface verifies the FileWriter interface can be implemented
func TestFileWriterInterface(t *testing.T) {
	var _ FileWriter = (*MockFileWriter)(nil)

	writer := &MockFileWriter{
		exists: map[string]bool{
			"existing.txt": true,
		},
	}

	// Test Write
	err := writer.Write("test.txt", []byte("content"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	// Test Exists - existing file
	if !writer.Exists("existing.txt") {
		t.Errorf("Exists(existing.txt) = %v, want %v", writer.Exists("existing.txt"), true)
	}

	// Test Exists - non-existing file
	if writer.Exists("nonexistent.txt") {
		t.Errorf("Exists(nonexistent.txt) = %v, want %v", writer.Exists("nonexistent.txt"), false)
	}
}
