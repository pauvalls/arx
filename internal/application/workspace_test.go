package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	arxcache "github.com/pauvalls/arx/internal/infrastructure/cache"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

func TestLoadWorkspace_NotFound(t *testing.T) {
	svc := NewWorkspaceService(nil)

	tmpDir := t.TempDir()
	_, _, err := svc.LoadWorkspace(tmpDir)
	if err == nil {
		t.Fatal("LoadWorkspace() expected error for missing config, got nil")
	}
}

func TestLoadWorkspace_Invalid(t *testing.T) {
	svc := NewWorkspaceService(nil)

	tmpDir := t.TempDir()
	// Write invalid YAML
	badPath := filepath.Join(tmpDir, "arx-workspace.yaml")
	if err := os.WriteFile(badPath, []byte(": invalid yaml :"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := svc.LoadWorkspace(tmpDir)
	if err == nil {
		t.Fatal("LoadWorkspace() expected error for invalid YAML, got nil")
	}
}

func TestLoadWorkspace_Valid(t *testing.T) {
	svc := NewWorkspaceService(nil)

	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "arx-workspace.yaml")
	wsContent := `version: "1"
projects:
  - path: services/*
shared:
  layers:
    - name: domain
      paths: ["domain/**"]
`
	if err := os.WriteFile(wsPath, []byte(wsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create services/ subdir so glob doesn't fail
	os.MkdirAll(filepath.Join(tmpDir, "services"), 0755)

	config, rootPath, err := svc.LoadWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("LoadWorkspace() unexpected error = %v", err)
	}
	if config == nil {
		t.Fatal("LoadWorkspace() returned nil config")
	}
	if config.Version != "1" {
		t.Errorf("LoadWorkspace() Version = %q, want %q", config.Version, "1")
	}
	if rootPath != tmpDir {
		t.Errorf("LoadWorkspace() rootPath = %q, want %q", rootPath, tmpDir)
	}
}

func TestLoadWorkspace_AcceptsFilePath(t *testing.T) {
	svc := NewWorkspaceService(nil)

	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "arx-workspace.yaml")
	wsContent := `version: "1"
projects:
  - path: services/*
`
	if err := os.WriteFile(wsPath, []byte(wsContent), 0644); err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(filepath.Join(tmpDir, "services"), 0755)

	config, rootPath, err := svc.LoadWorkspace(wsPath)
	if err != nil {
		t.Fatalf("LoadWorkspace() with file path error = %v", err)
	}
	if config == nil {
		t.Fatal("LoadWorkspace() with file path returned nil config")
	}
	if config.Version != "1" {
		t.Errorf("LoadWorkspace() Version = %q, want %q", config.Version, "1")
	}
	if rootPath != tmpDir {
		t.Errorf("LoadWorkspace() rootPath = %q, want %q", rootPath, tmpDir)
	}
}

func TestLoadWorkspace_FallsBackToArxYaml(t *testing.T) {
	svc := NewWorkspaceService(nil)

	tmpDir := t.TempDir()
	wsPath := filepath.Join(tmpDir, "arx.yaml")
	wsContent := `version: "1"
layers:
  - name: domain
    paths: ["domain/**"]
rules: []
workspace:
  version: "1"
  projects:
    - path: services/*
`
	if err := os.WriteFile(wsPath, []byte(wsContent), 0644); err != nil {
		t.Fatal(err)
	}

	os.MkdirAll(filepath.Join(tmpDir, "services"), 0755)

	config, rootPath, err := svc.LoadWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("LoadWorkspace() fallback error = %v", err)
	}
	if config == nil {
		t.Fatal("LoadWorkspace() fallback returned nil config")
	}
	if rootPath != tmpDir {
		t.Errorf("LoadWorkspace() rootPath = %q, want %q", rootPath, tmpDir)
	}
}

func TestResolveProjects_Globs(t *testing.T) {
	tmpDir := t.TempDir()
	subdirs := []string{"services/auth", "services/api", "libs/shared"}
	for _, d := range subdirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Projects: []domain.WorkspaceProject{
			{Path: "services/*"},
		},
	}

	svc := NewWorkspaceService(nil)
	projects, err := svc.ResolveProjects(wc, tmpDir)
	if err != nil {
		t.Fatalf("ResolveProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("ResolveProjects() returned %d projects, want 2", len(projects))
	}
}

func TestResolveProjects_Dedup(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "services/auth"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services/api"), 0755)

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Projects: []domain.WorkspaceProject{
			{Path: "services/*"},
			{Path: "services/auth"},
		},
	}

	svc := NewWorkspaceService(nil)
	projects, err := svc.ResolveProjects(wc, tmpDir)
	if err != nil {
		t.Fatalf("ResolveProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("ResolveProjects() returned %d projects, want 2 (dedup overlapping)", len(projects))
	}
}

func TestResolveProjects_ZeroMatches(t *testing.T) {
	tmpDir := t.TempDir()

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Projects: []domain.WorkspaceProject{
			{Path: "void/*"},
		},
	}

	svc := NewWorkspaceService(nil)
	_, err := svc.ResolveProjects(wc, tmpDir)
	if err == nil {
		t.Fatal("ResolveProjects() expected error for zero matches, got nil")
	}
}

func TestResolveProjects_Mixed(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "services/auth"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "services/api"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "libs/shared"), 0755)

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Projects: []domain.WorkspaceProject{
			{Path: "services/*"},
			{Path: "libs/shared"},
		},
	}

	svc := NewWorkspaceService(nil)
	projects, err := svc.ResolveProjects(wc, tmpDir)
	if err != nil {
		t.Fatalf("ResolveProjects() error = %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("ResolveProjects() returned %d projects, want 3", len(projects))
	}
}

func TestConfig_WorkspaceField(t *testing.T) {
	yamlContent := `version: "1"
layers:
  - name: domain
    paths: ["domain/**"]
rules: []
workspace:
  version: "1"
  projects:
    - path: services/*
  shared:
    layers:
      - name: domain
        paths: ["domain/**"]
`

	var cfg domain.Config
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal error = %v", err)
	}

	if cfg.Workspace == nil {
		t.Fatal("Config.Workspace is nil after unmarshal")
	}
	if cfg.Workspace.Version != "1" {
		t.Errorf("Config.Workspace.Version = %q, want %q", cfg.Workspace.Version, "1")
	}
	if len(cfg.Workspace.Projects) != 1 {
		t.Errorf("Config.Workspace.Projects count = %d, want 1", len(cfg.Workspace.Projects))
	}
	if cfg.Workspace.Projects[0].Path != "services/*" {
		t.Errorf("Config.Workspace.Projects[0].Path = %q, want %q", cfg.Workspace.Projects[0].Path, "services/*")
	}
	if cfg.Workspace.Shared == nil {
		t.Fatal("Config.Workspace.Shared is nil")
	}
	if len(cfg.Workspace.Shared.Layers) != 1 {
		t.Errorf("Config.Workspace.Shared.Layers count = %d, want 1", len(cfg.Workspace.Shared.Layers))
	}
}

func TestRun_AllPass(t *testing.T) {
	ctx := context.Background()
	svc := NewWorkspaceService([]ports.Detector{
		&mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
	})

	wc := &domain.WorkspaceConfig{Version: "1"}
	projects := []domain.ResolvedProject{
		domain.NewResolvedProject("/tmp/p1", nil),
		domain.NewResolvedProject("/tmp/p2", nil),
	}

	report, err := svc.Run(ctx, wc, projects, WorkspaceOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report == nil {
		t.Fatal("Run() returned nil report")
	}
	if !report.Passed() {
		t.Errorf("Run() Passed = false, want true")
	}
	if report.Summary.FailedProjects != 0 {
		t.Errorf("Run() FailedProjects = %d, want 0", report.Summary.FailedProjects)
	}
	if len(report.Projects) != 2 {
		t.Errorf("Run() projects count = %d, want 2", len(report.Projects))
	}
}

func TestRun_OneFail(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	violatingDetector := &callTrackingDetector{
		mockDetector: mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
		extractFn: func() []domain.Dependency {
			callCount++
			if callCount == 2 { // second project gets violations
				return []domain.Dependency{
					{SourceFile: "domain/main.go", SourceLine: 1, ImportPath: "bad_import", ResolvedLayer: "infra"},
				}
			}
			return []domain.Dependency{}
		},
	}

	svc := NewWorkspaceService([]ports.Detector{violatingDetector})

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Shared: &domain.WorkspaceShared{
			Layers: []domain.Layer{
				{Name: "domain", Paths: []string{"domain/**"}},
				{Name: "infra", Paths: []string{"infra/**"}},
			},
			Rules: []domain.Rule{
				{ID: "no-domain-to-infra", From: "domain", To: []string{"infra"}, Type: domain.RuleTypeCannot, Severity: domain.SeverityError},
			},
		},
	}

	projects := []domain.ResolvedProject{
		domain.NewResolvedProject("/tmp/p1", nil),
		domain.NewResolvedProject("/tmp/p2", nil),
	}

	report, err := svc.Run(ctx, wc, projects, WorkspaceOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Passed() {
		t.Error("Run() Passed = true, want false (one project should fail)")
	}
	if report.Summary.FailedProjects != 1 {
		t.Errorf("Run() FailedProjects = %d, want 1", report.Summary.FailedProjects)
	}
	if report.Summary.TotalProjects != 2 {
		t.Errorf("Run() TotalProjects = %d, want 2", report.Summary.TotalProjects)
	}
}

func TestRun_ErrorIsolation(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	errorDetector := &callTrackingDetector{
		mockDetector: mockDetector{
			name:         "go",
			detectResult: true,
		},
		extractFn: func() []domain.Dependency {
			callCount++
			if callCount == 1 { // first project fails
				return nil
			}
			return []domain.Dependency{}
		},
		extractErr: func() error {
			if callCount == 1 {
				return errors.New("extraction failed for p1")
			}
			return nil
		},
	}

	svc := NewWorkspaceService([]ports.Detector{errorDetector})

	wc := &domain.WorkspaceConfig{Version: "1"}
	projects := []domain.ResolvedProject{
		domain.NewResolvedProject("/tmp/p1", nil),
		domain.NewResolvedProject("/tmp/p2", nil),
	}

	report, err := svc.Run(ctx, wc, projects, WorkspaceOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Both projects should have entries
	if len(report.Projects) != 2 {
		t.Fatalf("Run() projects count = %d, want 2", len(report.Projects))
	}

	// First project should have an error
	p1 := report.Projects[0]
	if p1.Error == "" {
		t.Error("Project 1 should have error, got empty")
	}

	// Second project should be OK
	p2 := report.Projects[1]
	if p2.Error != "" {
		t.Errorf("Project 2 should not have error, got %q", p2.Error)
	}
}

func TestRun_MergeApplied(t *testing.T) {
	ctx := context.Background()

	svc := NewWorkspaceService([]ports.Detector{
		&mockDetector{
			name:         "go",
			detectResult: true,
			extractDeps:  []domain.Dependency{},
		},
	})

	wc := &domain.WorkspaceConfig{
		Version: "1",
		Shared: &domain.WorkspaceShared{
			Layers: []domain.Layer{
				{Name: "shared-layer", Paths: []string{"shared/**"}},
			},
			Rules: []domain.Rule{
				{ID: "shared-rule", From: "shared-layer", To: []string{"other"}, Type: domain.RuleTypeCannot, Severity: domain.SeverityError},
			},
		},
	}

	override := &domain.ProjectOverride{
		Layers: []domain.Layer{
			{Name: "override-layer", Paths: []string{"override/**"}},
		},
	}

	projects := []domain.ResolvedProject{
		domain.NewResolvedProject("/tmp/p1", override),
	}

	report, err := svc.Run(ctx, wc, projects, WorkspaceOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(report.Projects) != 1 {
		t.Fatalf("Run() projects count = %d, want 1", len(report.Projects))
	}
}

func TestRun_SharedCache_SecondProjectHitsCache(t *testing.T) {
	ctx := context.Background()

	// Create two temp project dirs with identical Go-like files
	tmpDir := t.TempDir()
	p1Dir := filepath.Join(tmpDir, "project1")
	p2Dir := filepath.Join(tmpDir, "project2")
	os.MkdirAll(p1Dir, 0755)
	os.MkdirAll(p2Dir, 0755)

	// Write a Go file to each project (same content for deterministic hashing)
	goContent := []byte("package main\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(p1Dir, "main.go"), goContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p2Dir, "main.go"), goContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a shared FileCache
	cacheDir := filepath.Join(tmpDir, ".arx-cache")
	sharedCache := arxcache.NewFileCache(cacheDir)
	// Don't set config hash — cache works without it (empty hash matches)

	det := &mockDetector{
		name:         "go",
		detectResult: true,
		extractDeps:  []domain.Dependency{{SourceFile: "main.go", SourceLine: 1, ImportPath: "fmt"}},
	}
	svc := NewWorkspaceService([]ports.Detector{det})

	wc := &domain.WorkspaceConfig{Version: "1"}
	projects := []domain.ResolvedProject{
		domain.NewResolvedProject(p1Dir, nil),
		domain.NewResolvedProject(p2Dir, nil),
	}

	opts := WorkspaceOptions{
		Cache: sharedCache,
	}

	report, err := svc.Run(ctx, wc, projects, opts)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(report.Projects) != 2 {
		t.Fatalf("Run() projects count = %d, want 2", len(report.Projects))
	}

	// After workspace run, cache directory should exist with entries from both projects
	// The "go" detector should have cached entries
	cacheDirExists, _ := exists(cacheDir)
	if !cacheDirExists {
		t.Fatal("shared cache directory was not created")
	}

	// The go detector cache dir should have entries
	goCacheDir := filepath.Join(cacheDir, "go")
	entries, err := os.ReadDir(goCacheDir)
	if err != nil {
		t.Fatalf("reading go cache dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("go cache dir is empty — no caching happened")
	}

	// Re-run: should complete without errors (cache hit for both)
	report2, err := svc.Run(ctx, wc, projects, opts)
	if err != nil {
		t.Fatalf("Run() second pass error = %v", err)
	}
	if len(report2.Projects) != 2 {
		t.Fatalf("Run() second pass projects count = %d, want 2", len(report2.Projects))
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// callTrackingDetector wraps mockDetector with dynamic behavior per call.
type callTrackingDetector struct {
	mockDetector
	extractFn  func() []domain.Dependency
	extractErr func() error
	detectFn   func() (bool, error)
}

func (d *callTrackingDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	if d.extractFn != nil {
		deps := d.extractFn()
		var err error
		if d.extractErr != nil {
			err = d.extractErr()
		}
		return deps, err
	}
	return d.mockDetector.ExtractImports(ctx, projectRoot, layers)
}
