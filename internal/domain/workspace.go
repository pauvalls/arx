package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WorkspaceShared holds shared layers, rules, and exclude patterns
// that are inherited by all projects unless overridden.
type WorkspaceShared struct {
	Layers  []Layer  `yaml:"layers,omitempty" json:"layers,omitempty"`
	Rules   []Rule   `yaml:"rules,omitempty" json:"rules,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// ProjectOverride holds per-project overrides that REPLACE shared fields
// entirely (shallow merge — not deep merge).
type ProjectOverride struct {
	Layers        []Layer  `yaml:"layers,omitempty" json:"layers,omitempty"`
	Rules         []Rule   `yaml:"rules,omitempty" json:"rules,omitempty"`
	Exclude       []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	MaxViolations *int     `yaml:"max_violations,omitempty" json:"max_violations,omitempty"`
}

// WorkspaceProject defines a single project entry in the workspace.
// Path can be a glob pattern or an explicit relative path.
type WorkspaceProject struct {
	Path     string           `yaml:"path" json:"path"`
	Override *ProjectOverride `yaml:"override,omitempty" json:"override,omitempty"`
}

// WorkspaceConfig is the root workspace configuration.
// It can be loaded from arx-workspace.yaml or from the workspace: field in arx.yaml.
type WorkspaceConfig struct {
	Version  string             `yaml:"version" json:"version"`
	Projects []WorkspaceProject `yaml:"projects" json:"projects"`
	Shared   *WorkspaceShared   `yaml:"shared,omitempty" json:"shared,omitempty"`
}

// ResolvedProject represents a concrete project after glob resolution.
type ResolvedProject struct {
	Name     string           // directory basename
	Path     string           // absolute path
	Override *ProjectOverride // nil if no per-project override
}

// NewResolvedProject creates a ResolvedProject with Name derived from path basename.
func NewResolvedProject(absPath string, override *ProjectOverride) ResolvedProject {
	return ResolvedProject{
		Name:     filepath.Base(absPath),
		Path:     absPath,
		Override: override,
	}
}

// Validate validates the workspace config structure.
func (wc *WorkspaceConfig) Validate() error {
	if wc.Version == "" {
		return fmt.Errorf("workspace config: version is required")
	}

	if len(wc.Projects) == 0 {
		return fmt.Errorf("workspace config: at least one project must be defined")
	}

	seenPaths := make(map[string]bool)
	for i, p := range wc.Projects {
		if p.Path == "" {
			return fmt.Errorf("workspace config: project[%d] path is required", i)
		}
		if seenPaths[p.Path] {
			return fmt.Errorf("workspace config: duplicate project path %q", p.Path)
		}
		seenPaths[p.Path] = true
	}

	return nil
}

// ResolveProjects resolves all project entries (globs + explicit paths) into
// concrete ResolvedProjects with absolute paths. Globs are resolved relative
// to baseDir. Duplicate paths are silently deduplicated. Unmatched globs
// cause an error listing all unmatched patterns.
func (wc *WorkspaceConfig) ResolveProjects(baseDir string) ([]ResolvedProject, error) {
	if err := wc.Validate(); err != nil {
		return nil, err
	}

	resolved := make(map[string]ResolvedProject)
	var unmatchedGlobs []string

	for _, proj := range wc.Projects {
		fullPattern := filepath.Join(baseDir, proj.Path)

		// Check if the path contains glob characters
		if containsGlob(proj.Path) {
			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return nil, fmt.Errorf("workspace config: invalid glob pattern %q: %w", proj.Path, err)
			}
			if len(matches) == 0 {
				unmatchedGlobs = append(unmatchedGlobs, proj.Path)
				continue
			}
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil || !info.IsDir() {
					continue
				}
				absPath, err := filepath.Abs(match)
				if err != nil {
					return nil, fmt.Errorf("workspace config: resolving path %q: %w", match, err)
				}
				resolved[absPath] = NewResolvedProject(absPath, proj.Override)
			}
		} else {
			// Explicit path: resolve to absolute
			// Try joining with baseDir first, if not absolute
			resolvedPath := proj.Path
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(baseDir, resolvedPath)
			}
			absPath, err := filepath.Abs(resolvedPath)
			if err != nil {
				return nil, fmt.Errorf("workspace config: resolving path %q: %w", proj.Path, err)
			}
			resolved[absPath] = NewResolvedProject(absPath, proj.Override)
		}
	}

	if len(unmatchedGlobs) > 0 {
		patterns := make([]string, len(unmatchedGlobs))
		for i, g := range unmatchedGlobs {
			patterns[i] = fmt.Sprintf("%q", g)
		}
		return nil, fmt.Errorf("workspace config: no matches for glob patterns: %s", strings.Join(patterns, ", "))
	}

	// Convert map to sorted slice for deterministic output
	result := make([]ResolvedProject, 0, len(resolved))
	for _, rp := range resolved {
		result = append(result, rp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	return result, nil
}

// Merge creates a domain.Config from shared config + project override (shallow merge).
// Project-level fields REPLACE shared fields; fields not specified inherit from shared.
func (wc *WorkspaceConfig) Merge(project *ResolvedProject) *Config {
	sv := SchemaVersion{Major: 1, Minor: 0}
	if wc.Version != "" {
		if parsed, err := ParseSchemaVersion(wc.Version); err == nil {
			sv = parsed
		}
	}
	cfg := &Config{
		Version: sv,
	}

	// Start with shared fields (if any)
	shared := wc.Shared

	// Merge Layers: override fully replaces shared
	if project.Override != nil && len(project.Override.Layers) > 0 {
		cfg.Layers = make([]Layer, len(project.Override.Layers))
		copy(cfg.Layers, project.Override.Layers)
	} else if shared != nil {
		cfg.Layers = make([]Layer, len(shared.Layers))
		copy(cfg.Layers, shared.Layers)
	}

	// Merge Rules: override fully replaces shared
	if project.Override != nil && len(project.Override.Rules) > 0 {
		cfg.Rules = make([]Rule, len(project.Override.Rules))
		copy(cfg.Rules, project.Override.Rules)
	} else if shared != nil {
		cfg.Rules = make([]Rule, len(shared.Rules))
		copy(cfg.Rules, shared.Rules)
	}

	// Merge Exclude: override fully replaces shared
	if project.Override != nil && len(project.Override.Exclude) > 0 {
		cfg.Exclude = make([]string, len(project.Override.Exclude))
		copy(cfg.Exclude, project.Override.Exclude)
	} else if shared != nil {
		cfg.Exclude = make([]string, len(shared.Exclude))
		copy(cfg.Exclude, shared.Exclude)
	}

	// Merge MaxViolations: override replaces shared (shared has no MaxViolations, but handle it)
	if project.Override != nil && project.Override.MaxViolations != nil {
		cfg.MaxViolations = *project.Override.MaxViolations
	}

	return cfg
}

// containsGlob checks if a path contains glob characters (*, ?, [).
func containsGlob(path string) bool {
	return strings.ContainsAny(path, "*?[")
}
