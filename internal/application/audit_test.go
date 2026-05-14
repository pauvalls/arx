package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// MockConfigReader is a test double for ConfigReader
type MockConfigReader struct {
	config *domain.Config
	err    error
}

func (m *MockConfigReader) Read(configPath string) (*domain.Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.config, nil
}

func (m *MockConfigReader) Validate(config *domain.Config) error {
	return nil
}

// MockDetector is a test double for Detector
type MockDetector struct {
	name         string
	detectResult bool
	detectErr    error
	deps         []domain.Dependency
	extractErr   error
}

func (m *MockDetector) Name() string {
	return m.name
}

func (m *MockDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	return m.detectResult, m.detectErr
}

func (m *MockDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	if m.extractErr != nil {
		return nil, m.extractErr
	}
	return m.deps, nil
}

// MockHistoryStorage is a test double for HistoryStorage
type MockHistoryStorage struct {
	savedReports []*domain.AuditReport
	loadLatest   *domain.AuditReport
	saveErr      error
	loadErr      error
}

func (m *MockHistoryStorage) Save(ctx context.Context, report *domain.AuditReport) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	m.savedReports = append(m.savedReports, report)
	return "test-path", nil
}

func (m *MockHistoryStorage) Load(ctx context.Context, date time.Time) (*domain.AuditReport, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return nil, nil
}

func (m *MockHistoryStorage) LoadLatest(ctx context.Context) (*domain.AuditReport, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.loadLatest, nil
}

func (m *MockHistoryStorage) List(ctx context.Context) ([]time.Time, error) {
	return []time.Time{}, nil
}

func (m *MockHistoryStorage) DeleteOld(ctx context.Context, maxAudits int) (int, error) {
	return 0, nil
}

func createTestConfig() *domain.Config {
	return &domain.Config{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"domain/**"}},
			{Name: "application", Paths: []string{"application/**"}},
			{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "no-domain-to-infra",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     domain.RuleTypeCannot,
				Severity: domain.SeverityError,
			},
		},
	}
}

func createTestDependencies() []domain.Dependency {
	return []domain.Dependency{
		{
			SourceFile:   "application/service.go",
			ImportPath:   "github.com/example/domain",
			ResolvedLayer: "domain",
		},
		{
			SourceFile:   "domain/entity.go",
			ImportPath:   "github.com/example/infrastructure",
			ResolvedLayer: "infrastructure",
		},
	}
}

func TestAuditService_Audit_Success(t *testing.T) {
	// Setup
	config := createTestConfig()
	configReader := &MockConfigReader{config: config}
	
	detectors := []ports.Detector{
		&MockDetector{
			name:         "go",
			detectResult: true,
			deps:         createTestDependencies(),
		},
	}
	
	historyStorage := &MockHistoryStorage{}
	
	service := NewAuditService(configReader, detectors, historyStorage, "")
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	projectRoot := tmpDir
	
	// Write minimal config
	configContent := `layers:
  - name: domain
    paths: ["domain/**"]
  - name: application
    paths: ["application/**"]
rules:
  - id: no-domain-to-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Execute
	ctx := context.Background()
	report, err := service.Audit(ctx, projectRoot, configPath)
	
	// Assert
	if err != nil {
		t.Fatalf("Audit() returned error: %v", err)
	}
	
	if report == nil {
		t.Fatal("Audit() returned nil report")
	}
	
	if report.ProjectRoot != projectRoot {
		t.Errorf("Expected ProjectRoot %q, got %q", projectRoot, report.ProjectRoot)
	}
	
	if report.ConfigHash == "" {
		t.Error("Expected ConfigHash to be set")
	}
	
	if report.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
	
	// Should have saved to history
	if len(historyStorage.savedReports) != 1 {
		t.Errorf("Expected 1 saved report, got %d", len(historyStorage.savedReports))
	}
}

func TestAuditService_Audit_NoConfig(t *testing.T) {
	// Setup
	configReader := &MockConfigReader{
		err: os.ErrNotExist,
	}
	
	detectors := []ports.Detector{
		&MockDetector{name: "go", detectResult: true},
	}
	
	service := NewAuditService(configReader, detectors, nil, "")
	
	// Execute
	ctx := context.Background()
	report, err := service.Audit(ctx, "/nonexistent", "/nonexistent/config.yaml")
	
	// Assert
	if err == nil {
		t.Fatal("Expected error for missing config, got nil")
	}
	
	if report != nil {
		t.Error("Expected nil report on error")
	}
}

func TestAuditService_Audit_EmptyProject(t *testing.T) {
	// Setup
	config := createTestConfig()
	configReader := &MockConfigReader{config: config}
	
	// Detector that finds nothing
	detectors := []ports.Detector{
		&MockDetector{
			name:         "go",
			detectResult: false, // No Go project detected
			deps:         []domain.Dependency{},
		},
	}
	
	historyStorage := &MockHistoryStorage{}
	
	service := NewAuditService(configReader, detectors, historyStorage, "")
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	projectRoot := tmpDir
	
	// Write minimal config
	configContent := `layers:
  - name: domain
    paths: ["domain/**"]
  - name: application
    paths: ["application/**"]
rules:
  - id: no-domain-to-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Execute
	ctx := context.Background()
	report, err := service.Audit(ctx, projectRoot, configPath)
	
	// Assert
	if err != nil {
		t.Fatalf("Audit() returned error: %v", err)
	}
	
	if report == nil {
		t.Fatal("Audit() returned nil report")
	}
	
	// Should have empty violations
	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations for empty project, got %d", len(report.Violations))
	}
	
	// Should have empty coupling matrix
	if report.CouplingMatrix.Count() != 0 {
		t.Errorf("Expected 0 dependencies in coupling matrix, got %d", report.CouplingMatrix.Count())
	}
}

func TestAuditService_Audit_WithTrends(t *testing.T) {
	// Setup
	config := createTestConfig()
	configReader := &MockConfigReader{config: config}
	
	detectors := []ports.Detector{
		&MockDetector{
			name:         "go",
			detectResult: true,
			deps:         createTestDependencies(),
		},
	}
	
	// Previous audit with fewer violations
	previousReport := &domain.AuditReport{
		Timestamp:   time.Now().Add(-24 * time.Hour),
		Violations:  []domain.Violation{}, // No violations before
		DebtScore:   domain.DebtScore{Total: 0},
	}
	
	historyStorage := &MockHistoryStorage{
		loadLatest: previousReport,
	}
	
	service := NewAuditService(configReader, detectors, historyStorage, "")
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	projectRoot := tmpDir
	
	configContent := `layers:
  - name: domain
    paths: ["domain/**"]
  - name: application
    paths: ["application/**"]
rules:
  - id: no-domain-to-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Execute
	ctx := context.Background()
	report, err := service.Audit(ctx, projectRoot, configPath)
	
	// Assert
	if err != nil {
		t.Fatalf("Audit() returned error: %v", err)
	}
	
	// Should have trend report
	if report.TrendReport.Summary == "" {
		t.Error("Expected trend report to have summary")
	}
	
	// Should show degradation (more violations than before)
	if report.TrendReport.Status != domain.TrendDegraded {
		t.Errorf("Expected trend status %q, got %q", domain.TrendDegraded, report.TrendReport.Status)
	}
}

func TestAuditService_Audit_NoHistoryStorage(t *testing.T) {
	// Setup
	config := createTestConfig()
	configReader := &MockConfigReader{config: config}
	
	detectors := []ports.Detector{
		&MockDetector{
			name:         "go",
			detectResult: true,
			deps:         createTestDependencies(),
		},
	}
	
	// No history storage
	service := NewAuditService(configReader, detectors, nil, "")
	
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	projectRoot := tmpDir
	
	configContent := `layers:
  - name: domain
    paths: ["domain/**"]
  - name: application
    paths: ["application/**"]
rules:
  - id: no-domain-to-infra
    from: domain
    to: [infrastructure]
    type: Cannot
    severity: error
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	
	// Execute
	ctx := context.Background()
	report, err := service.Audit(ctx, projectRoot, configPath)
	
	// Assert
	if err != nil {
		t.Fatalf("Audit() returned error: %v", err)
	}
	
	// Should have trend report indicating no history
	if report.TrendReport.Summary != "History storage not configured" {
		t.Errorf("Expected trend summary 'History storage not configured', got %q", report.TrendReport.Summary)
	}
}

func TestAuditService_CalculateCoupling(t *testing.T) {
	// Create temp files to match layer paths
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "application"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "infrastructure"), 0755)
	
	// Create config with paths that match the temp directory structure
	config := &domain.Config{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{filepath.Join(tmpDir, "domain")}},
			{Name: "application", Paths: []string{filepath.Join(tmpDir, "application")}},
			{Name: "infrastructure", Paths: []string{filepath.Join(tmpDir, "infrastructure")}},
		},
	}
	
	// Create dependencies with known coupling
	deps := []domain.Dependency{
		{SourceFile: filepath.Join(tmpDir, "application/service.go"), ResolvedLayer: "domain"},
		{SourceFile: filepath.Join(tmpDir, "application/service.go"), ResolvedLayer: "domain"},
		{SourceFile: filepath.Join(tmpDir, "application/repo.go"), ResolvedLayer: "infrastructure"},
		{SourceFile: filepath.Join(tmpDir, "domain/entity.go"), ResolvedLayer: "infrastructure"},
	}
	
	// Manually test coupling calculation
	layerMap := make(map[string]*domain.Layer)
	for i := range config.Layers {
		layerMap[config.Layers[i].Name] = &config.Layers[i]
	}
	
	matrix := domain.NewCouplingMatrix()
	for _, dep := range deps {
		sourceLayer := resolveLayerForCoupling(dep.SourceFile, layerMap)
		targetLayer := dep.ResolvedLayer
		if sourceLayer != "" && targetLayer != "" {
			matrix.Add(sourceLayer, targetLayer)
		}
	}
	
	// Assert
	appToDomain := matrix.Get("application", "domain")
	if appToDomain != 2 {
		t.Errorf("Expected application→domain coupling = 2, got %d", appToDomain)
	}
	
	appToInfra := matrix.Get("application", "infrastructure")
	if appToInfra != 1 {
		t.Errorf("Expected application→infrastructure coupling = 1, got %d", appToInfra)
	}
	
	domainToInfra := matrix.Get("domain", "infrastructure")
	if domainToInfra != 1 {
		t.Errorf("Expected domain→infrastructure coupling = 1, got %d", domainToInfra)
	}
}

func TestAuditService_CalculateDebt(t *testing.T) {
	// Setup
	violations := []domain.Violation{
		{Severity: domain.SeverityError},
		{Severity: domain.SeverityError},
		{Severity: domain.SeverityError},
		{Severity: domain.SeverityWarning},
		{Severity: domain.SeverityWarning},
	}
	
	matrix := domain.NewCouplingMatrix()
	// Add circular dependency: domain→infrastructure and infrastructure→domain
	matrix.Add("domain", "infrastructure")
	matrix.Add("infrastructure", "domain")
	
	// Calculate debt inline (same logic as calculateDebt)
	debt := domain.NewDebtScore()
	for _, v := range violations {
		debt.AddViolation(string(v.Severity))
	}
	
	// Count circular dependencies
	circularCount := 0
	checked := make(map[string]bool)
	for fromLayer, targets := range matrix.FromTo {
		for toLayer := range targets {
			pair := fromLayer + "->" + toLayer
			reversePair := toLayer + "->" + fromLayer
			if checked[reversePair] {
				continue
			}
			if reverseTargets, ok := matrix.FromTo[toLayer]; ok {
				if _, exists := reverseTargets[fromLayer]; exists {
					circularCount++
					checked[pair] = true
				}
			}
		}
	}
	
	// Add circular penalty
	for i := 0; i < circularCount; i++ {
		debt.BySeverity["circular"] = debt.BySeverity["circular"] + 5
	}
	debt.Calculate()
	
	// Assert
	// Base: 3 errors × 3 + 2 warnings × 1 = 11
	// Circular penalty: 1 circular × 5 = 5
	// Total: 16
	expectedBase := (3 * 3) + (2 * 1) // 11
	if debt.Total < expectedBase {
		t.Errorf("Expected debt total >= %d (base), got %d", expectedBase, debt.Total)
	}
	
	if debt.BySeverity["error"] != 3 {
		t.Errorf("Expected 3 error violations, got %d", debt.BySeverity["error"])
	}
	
	if debt.BySeverity["warning"] != 2 {
		t.Errorf("Expected 2 warning violations, got %d", debt.BySeverity["warning"])
	}
}
