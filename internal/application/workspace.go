package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

// WorkspaceOptions holds configurable options for workspace execution.
type WorkspaceOptions struct {
	Verbose bool
	Jobs    int  // max workers for detector concurrency (0 = unlimited)
	Cache   ports.Cache // shared cache instance across projects (nil = no cache)
}

// WorkspaceService orchestrates the workspace audit flow:
// load config → resolve projects → merge configs → detect + evaluate → aggregate.
type WorkspaceService struct {
	detectors []ports.Detector
}

// NewWorkspaceService creates a new WorkspaceService with the given dependencies.
func NewWorkspaceService(detectors []ports.Detector) *WorkspaceService {
	return &WorkspaceService{
		detectors: detectors,
	}
}

// LoadWorkspace reads and parses a workspace config file.
// It accepts either a directory or a file path:
//   - Directory: looks for arx-workspace.yaml, then falls back to arx.yaml with workspace: field
//   - File: reads the file directly (must be arx-workspace.yaml or arx.yaml with workspace: field)
//
// Returns the parsed WorkspaceConfig, the root path (directory for relative paths), and any error.
func (s *WorkspaceService) LoadWorkspace(workspacePath string) (*domain.WorkspaceConfig, string, error) {
	info, err := os.Stat(workspacePath)
	if err != nil {
		return nil, "", fmt.Errorf("workspace config: %w", err)
	}

	if info.IsDir() {
		// Try arx-workspace.yaml first
		wsPath := filepath.Join(workspacePath, "arx-workspace.yaml")
		if data, err := os.ReadFile(wsPath); err == nil {
			config, err := parseWorkspaceConfig(data)
			if err != nil {
				return nil, "", fmt.Errorf("workspace config: %w", err)
			}
			return config, workspacePath, nil
		}

		// Fallback: try arx.yaml with workspace: field
		wsPath = filepath.Join(workspacePath, "arx.yaml")
		if data, err := os.ReadFile(wsPath); err == nil {
			var cfg domain.Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, "", fmt.Errorf("workspace config: parsing arx.yaml: %w", err)
			}
			if cfg.Workspace != nil {
				return cfg.Workspace, workspacePath, nil
			}
		}

		return nil, "", fmt.Errorf("workspace config: no arx-workspace.yaml or arx.yaml with workspace: field found in %s", workspacePath)
	}

	// It's a file — read directly
	data, err := os.ReadFile(workspacePath)
	if err != nil {
		return nil, "", fmt.Errorf("workspace config: %w", err)
	}

	// Try parsing as workspace config first
	config, err := parseWorkspaceConfig(data)
	if err == nil {
		rootPath := filepath.Dir(workspacePath)
		return config, rootPath, nil
	}

	// Try parsing as arx.yaml with workspace: field
	var cfg domain.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("workspace config: parsing %s: %w", workspacePath, err)
	}
	if cfg.Workspace != nil {
		rootPath := filepath.Dir(workspacePath)
		return cfg.Workspace, rootPath, nil
	}

	return nil, "", fmt.Errorf("workspace config: no workspace configuration found in %s", workspacePath)
}

// ResolveProjects resolves all project entries (globs + explicit paths) into
// concrete ResolvedProjects relative to the given root path.
func (s *WorkspaceService) ResolveProjects(wc *domain.WorkspaceConfig, rootPath string) ([]domain.ResolvedProject, error) {
	return wc.ResolveProjects(rootPath)
}

// Run executes the workspace audit: for each resolved project, merge config,
// detect dependencies, evaluate rules, and collect results.
// Error in one project does NOT fail others (error isolation).
func (s *WorkspaceService) Run(ctx context.Context, wc *domain.WorkspaceConfig, projects []domain.ResolvedProject, opts WorkspaceOptions) (*domain.WorkspaceReport, error) {
	if len(s.detectors) == 0 {
		return nil, fmt.Errorf("workspace: no detectors provided")
	}

	var projectReports []domain.ProjectReport

	for _, project := range projects {
		pr := s.runProject(ctx, wc, project, opts)
		projectReports = append(projectReports, pr)
	}

	report := domain.NewWorkspaceReport(wc.Version, projectReports)
	return &report, nil
}

// runProject runs the audit for a single project with error isolation.
func (s *WorkspaceService) runProject(ctx context.Context, wc *domain.WorkspaceConfig, project domain.ResolvedProject, opts WorkspaceOptions) domain.ProjectReport {
	start := time.Now()

	// Merge shared config with project override
	merged := wc.Merge(&project)

	// Run detectors with optional cache and worker pool
	var dependencies []domain.Dependency
	var err error
	if opts.Cache != nil {
		dependencies, err = RunDetectorsCached(ctx, project.Path, merged.Layers, s.detectors, opts.Cache)
	} else {
		dependencies, err = RunDetectors(ctx, project.Path, merged.Layers, s.detectors, opts.Jobs)
	}
	if err != nil {
		elapsed := time.Since(start)
		return domain.NewProjectReport(project.Path, nil, elapsed, err)
	}

	// Evaluate rules
	violations := EvaluateArchitecture(dependencies, merged.Rules, merged.Layers, merged.UserFunctions())

	elapsed := time.Since(start)
	return domain.NewProjectReport(project.Path, violations, elapsed, nil)
}

// parseWorkspaceConfig parses YAML data into a WorkspaceConfig.
func parseWorkspaceConfig(data []byte) (*domain.WorkspaceConfig, error) {
	var config domain.WorkspaceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
