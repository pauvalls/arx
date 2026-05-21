package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestInitService_Scan(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)

	writer := newMockFileWriter()
	service := NewInitService(writer)

	info, err := service.Scan(tmpDir)
	if err != nil {
		t.Fatalf("InitService.Scan() error = %v", err)
	}

	found := false
	for _, lang := range info.Languages {
		if lang == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("InitService.Scan() did not detect Go")
	}
}

func TestInitService_Generate(t *testing.T) {
	writer := newMockFileWriter()
	service := NewInitService(writer)

	info := &ProjectInfo{
		Root:      "/test",
		Languages: []string{"go"},
		SuggestedLayers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain/**"}},
			{Name: "application", Paths: []string{"internal/application/**"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure/**"}},
		},
	}

	config, err := service.Generate(info)
	if err != nil {
		t.Fatalf("InitService.Generate() error = %v", err)
	}

	if len(config.Rules) < 5 {
		t.Errorf("InitService.Generate() created %d rules, want at least 5", len(config.Rules))
	}
}

func TestInitService_Write(t *testing.T) {
	writer := newMockFileWriter()
	service := NewInitService(writer)

	config := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	}

	err := service.Write(config, "arx.yaml")
	if err != nil {
		t.Fatalf("InitService.Write() error = %v", err)
	}

	if _, ok := writer.files["arx.yaml"]; !ok {
		t.Errorf("InitService.Write() did not write to arx.yaml")
	}
}

func TestInitService_Init(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "internal", "domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "internal", "application"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "internal", "infrastructure"), 0755)

	writer := newMockFileWriter()
	service := NewInitService(writer)

	config, err := service.Init(tmpDir, "arx.yaml")
	if err != nil {
		t.Fatalf("InitService.Init() error = %v", err)
	}

	if config.Version.String() != "1.0" {
		t.Errorf("InitService.Init() config.Version = %q, want %q", config.Version.String(), "1.0")
	}

	if len(config.Rules) < 5 {
		t.Errorf("InitService.Init() created %d rules, want at least 5", len(config.Rules))
	}

	if _, ok := writer.files["arx.yaml"]; !ok {
		t.Errorf("InitService.Init() did not write config file")
	}
}

func TestCheckService_Load(t *testing.T) {
	expectedConfig := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain/**"}}},
		Rules:   []domain.Rule{},
	}
	reader := &mockConfigReader{config: expectedConfig}
	reporter := &mockReporter{}

	service := NewCheckService(reader, []ports.Detector{}, reporter)

	config, err := service.Load("arx.yaml")
	if err != nil {
		t.Fatalf("CheckService.Load() error = %v", err)
	}

	if config.Version.String() != "1.0" {
		t.Errorf("CheckService.Load() config.Version = %q, want %q", config.Version.String(), "1.0")
	}
}

func TestCheckService_Detect(t *testing.T) {
	ctx := context.Background()
	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"},
		},
	}

	reader := &mockConfigReader{}
	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{goDetector}, reporter)

	deps, err := service.Detect(ctx, "/test", []domain.Layer{})
	if err != nil {
		t.Fatalf("CheckService.Detect() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("CheckService.Detect() returned %d deps, want 1", len(deps))
	}
}

func TestCheckService_Evaluate(t *testing.T) {
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
		},
	}

	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain"}},
		{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
	}

	reader := &mockConfigReader{}
	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{}, reporter)

	violations := service.Evaluate(dependencies, rules, layers)

	if len(violations) != 1 {
		t.Errorf("CheckService.Evaluate() returned %d violations, want 1", len(violations))
	}
}

func TestCheckService_Report(t *testing.T) {
	violations := []domain.Violation{
		{ID: "D-01", RuleID: "R1", Message: "test violation"},
	}

	reader := &mockConfigReader{}
	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{}, reporter)

	err := service.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("CheckService.Report() error = %v", err)
	}

	if len(reporter.reportedViolations) != 1 {
		t.Errorf("CheckService.Report() reported %d violations, want 1", len(reporter.reportedViolations))
	}
}

func TestCheckService_Check(t *testing.T) {
	ctx := context.Background()

	config := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "infrastructure", Paths: []string{"internal/infrastructure"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "domain-imports-infrastructure",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     domain.RuleTypeCannot,
				Severity: domain.SeverityError,
			},
		},
	}

	reader := &mockConfigReader{config: config}

	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{
				SourceFile:    "internal/domain/user.go",
				SourceLine:    10,
				ImportPath:    "github.com/example/arx/internal/infrastructure/db",
				ResolvedLayer: "infrastructure",
			},
		},
	}

	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{goDetector}, reporter)

	err := service.Check(ctx, "arx.yaml", "/test", ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("CheckService.Check() error = %v", err)
	}

	if len(reporter.reportedViolations) != 1 {
		t.Errorf("CheckService.Check() reported %d violations, want 1", len(reporter.reportedViolations))
	}
}

func TestCheckService_Check_LoadError(t *testing.T) {
	ctx := context.Background()
	reader := &mockConfigReader{readErr: errors.New("config not found")}
	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{}, reporter)

	err := service.Check(ctx, "arx.yaml", "/test", ports.OutputFormatTerminal)
	if err == nil {
		t.Errorf("CheckService.Check() with load error should return error")
	}
}

func TestCheckService_Check_DetectError(t *testing.T) {
	ctx := context.Background()
	config := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
		Rules:   []domain.Rule{},
	}
	reader := &mockConfigReader{config: config}

	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractErr:   errors.New("extraction failed"),
	}

	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{goDetector}, reporter)

	err := service.Check(ctx, "arx.yaml", "/test", ports.OutputFormatTerminal)
	if err == nil {
		t.Errorf("CheckService.Check() with detect error should return error")
	}
}

func TestCheckService_Check_ReportError(t *testing.T) {
	ctx := context.Background()
	config := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers:  []domain.Layer{{Name: "domain", Paths: []string{"internal/domain"}}},
		Rules:   []domain.Rule{},
	}
	reader := &mockConfigReader{config: config}
	reporter := &mockReporter{reportErr: errors.New("report failed")}
	service := NewCheckService(reader, []ports.Detector{}, reporter)

	err := service.Check(ctx, "arx.yaml", "/test", ports.OutputFormatTerminal)
	if err == nil {
		t.Errorf("CheckService.Check() with report error should return error")
	}
}

func TestCheckService_Check_NoViolations(t *testing.T) {
	ctx := context.Background()

	config := &domain.Config{
		Version: domain.SchemaVersion{Major: 1, Minor: 0},
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
			{Name: "application", Paths: []string{"internal/application"}},
		},
		Rules: []domain.Rule{
			{
				ID:       "domain-imports-infrastructure",
				From:     "domain",
				To:       []string{"infrastructure"},
				Type:     domain.RuleTypeCannot,
				Severity: domain.SeverityError,
			},
		},
	}

	reader := &mockConfigReader{config: config}

	// Detector returns a dependency that does NOT violate any rule
	goDetector := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps: []domain.Dependency{
			{
				SourceFile:    "internal/application/service.go",
				SourceLine:    5,
				ImportPath:    "github.com/example/arx/internal/domain/user",
				ResolvedLayer: "domain",
			},
		},
	}

	reporter := &mockReporter{}
	service := NewCheckService(reader, []ports.Detector{goDetector}, reporter)

	err := service.Check(ctx, "arx.yaml", "/test", ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("CheckService.Check() error = %v", err)
	}

	if len(reporter.reportedViolations) != 0 {
		t.Errorf("CheckService.Check() reported %d violations, want 0", len(reporter.reportedViolations))
	}
}
