package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  WorkspaceConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal config",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "services/auth"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty version",
			config: WorkspaceConfig{
				Version: "",
				Projects: []WorkspaceProject{
					{Path: "services/auth"},
				},
			},
			wantErr: true,
			errMsg:  "version",
		},
		{
			name:    "no projects",
			config:  WorkspaceConfig{Version: "1"},
			wantErr: true,
			errMsg:  "project",
		},
		{
			name: "empty project path",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: ""},
				},
			},
			wantErr: true,
			errMsg:  "path",
		},
		{
			name: "duplicate project paths",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "services/auth"},
					{Path: "services/auth"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestWorkspaceConfig_ResolveProjects(t *testing.T) {
	// Use a temp dir with known subdirectories
	tmpDir := t.TempDir()

	// Create some subdirectories that simulate projects
	subDirs := []string{"services/auth", "services/api", "libs/shared"}
	for _, d := range subDirs {
		fullPath := filepath.Join(tmpDir, d)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("failed to create test dir %s: %v", d, err)
		}
	}

	tests := []struct {
		name      string
		config    WorkspaceConfig
		wantCount int
		wantErr   bool
		errMsg    string
	}{
		{
			name: "simple glob resolves multiple",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "services/*"},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "explicit path resolves one",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "libs/shared"},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "double-star glob",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "**/auth"},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "no matches returns error",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "void/*"},
				},
			},
			wantErr: true,
			errMsg:  "no matches",
		},
		{
			name: "mixed globs and explicit paths",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "services/*"},
					{Path: "libs/shared"},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "overlapping glob results deduplicated",
			config: WorkspaceConfig{
				Version: "1",
				Projects: []WorkspaceProject{
					{Path: "services/*"},
					{Path: "services/auth"},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := tt.config.ResolveProjects(tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveProjects() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ResolveProjects() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveProjects() unexpected error = %v", err)
			}
			if len(projects) != tt.wantCount {
				t.Errorf("ResolveProjects() returned %d projects, want %d", len(projects), tt.wantCount)
			}
			// Verify each project has an absolute path and non-empty name
			for _, p := range projects {
				if !filepath.IsAbs(p.Path) {
					t.Errorf("ResolveProjects() project path %q is not absolute", p.Path)
				}
				if p.Name == "" {
					t.Errorf("ResolveProjects() project has empty name for path %q", p.Path)
				}
			}
		})
	}
}

func TestWorkspaceConfig_Merge(t *testing.T) {
	sharedLayers := []Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "infra", Paths: []string{"infra/**"}},
		{Name: "api", Paths: []string{"api/**"}},
	}
	sharedRules := []Rule{
		{ID: "no-domain-to-infra", From: "domain", To: []string{"infra"}, Type: RuleTypeCannot, Severity: SeverityError},
	}

	tests := []struct {
		name     string
		shared   *WorkspaceShared
		override *ProjectOverride
		// expectations
		wantLayerCount int
		wantRuleCount  int
	}{
		{
			name: "full override replaces shared",
			shared: &WorkspaceShared{
				Layers: sharedLayers,
				Rules:  sharedRules,
			},
			override: &ProjectOverride{
				Layers: []Layer{{Name: "custom", Paths: []string{"custom/**"}}},
				Rules:  []Rule{{ID: "custom-rule", From: "custom", To: []string{"custom"}, Type: RuleTypeCannot, Severity: SeverityError}},
			},
			wantLayerCount: 1,
			wantRuleCount:  1,
		},
		{
			name: "partial override keeps shared rules",
			shared: &WorkspaceShared{
				Layers: sharedLayers,
				Rules:  sharedRules,
			},
			override: &ProjectOverride{
				Layers: []Layer{{Name: "custom", Paths: []string{"custom/**"}}},
				// no rules override — should inherit shared
			},
			wantLayerCount: 1,
			wantRuleCount:  1,
		},
		{
			name: "partial override keeps shared layers",
			shared: &WorkspaceShared{
				Layers: sharedLayers,
				Rules:  sharedRules,
			},
			override: &ProjectOverride{
				Rules: []Rule{{ID: "custom-rule", From: "domain", To: []string{"infra"}, Type: RuleTypeCannot, Severity: SeverityError}},
			},
			wantLayerCount: 3,
			wantRuleCount:  1,
		},
		{
			name: "no override uses shared",
			shared: &WorkspaceShared{
				Layers: sharedLayers,
				Rules:  sharedRules,
			},
			override:      nil,
			wantLayerCount: 3,
			wantRuleCount:  1,
		},
		{
			name:     "nil shared uses override only",
			shared:   nil,
			override: &ProjectOverride{Layers: []Layer{{Name: "single", Paths: []string{"single/**"}}}, Rules: []Rule{{ID: "r1", From: "single", To: []string{"single"}, Type: RuleTypeCannot, Severity: SeverityError}}},
			wantLayerCount: 1,
			wantRuleCount:  1,
		},
		{
			name:     "nil override uses shared",
			shared:   &WorkspaceShared{Layers: sharedLayers, Rules: sharedRules},
			override: nil,
			wantLayerCount: 3,
			wantRuleCount:  1,
		},
		{
			name: "override replaces layers but inherits no shared rules when not set",
			shared: &WorkspaceShared{
				Layers: sharedLayers,
			},
			override: &ProjectOverride{
				Layers: []Layer{{Name: "ovr", Paths: []string{"ovr/**"}}},
			},
			wantLayerCount: 1,
			wantRuleCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := &WorkspaceConfig{
				Version: "1",
				Shared:  tt.shared,
			}

			rp := &ResolvedProject{
				Name:     "test",
				Path:     "/tmp/test",
				Override: tt.override,
			}

			merged := wc.Merge(rp)
			if merged == nil {
				t.Fatal("Merge() returned nil")
			}

			if len(merged.Layers) != tt.wantLayerCount {
				t.Errorf("Merge() layers count = %d, want %d", len(merged.Layers), tt.wantLayerCount)
			}
			if len(merged.Rules) != tt.wantRuleCount {
				t.Errorf("Merge() rules count = %d, want %d", len(merged.Rules), tt.wantRuleCount)
			}

			// Verify shallow merge: if override has layers, they should be EXACTLY the override's layers
			if tt.override != nil && len(tt.override.Layers) > 0 {
				for i, l := range merged.Layers {
					if l.Name != tt.override.Layers[i].Name {
						t.Errorf("Merge() layers[%d].Name = %q, want %q (should be override layers)", i, l.Name, tt.override.Layers[i].Name)
					}
				}
			} else if tt.shared != nil {
				// No override layers — should be shared layers
				for i, l := range merged.Layers {
					if l.Name != tt.shared.Layers[i].Name {
						t.Errorf("Merge() layers[%d].Name = %q, want %q (should be shared layers)", i, l.Name, tt.shared.Layers[i].Name)
					}
				}
			}

			// Verify shallow merge: if override has rules, they should be EXACTLY the override's rules
			if tt.override != nil && len(tt.override.Rules) > 0 {
				for i, r := range merged.Rules {
					if r.ID != tt.override.Rules[i].ID {
						t.Errorf("Merge() rules[%d].ID = %q, want %q (should be override rules)", i, r.ID, tt.override.Rules[i].ID)
					}
				}
			} else if tt.shared != nil {
				// No override rules — should be shared rules
				for i, r := range merged.Rules {
					if r.ID != tt.shared.Rules[i].ID {
						t.Errorf("Merge() rules[%d].ID = %q, want %q (should be shared rules)", i, r.ID, tt.shared.Rules[i].ID)
					}
				}
			}

			// Verify exclude patterns
			wantExcludeLen := 0
			if tt.override != nil && len(tt.override.Exclude) > 0 {
				wantExcludeLen = len(tt.override.Exclude)
			} else if tt.shared != nil && len(tt.shared.Exclude) > 0 {
				wantExcludeLen = len(tt.shared.Exclude)
			}
			if len(merged.Exclude) != wantExcludeLen {
				t.Errorf("Merge() exclude count = %d, want %d", len(merged.Exclude), wantExcludeLen)
			}
		})
	}
}

func TestResolvedProject_NameFromPath(t *testing.T) {
	tests := []struct {
		path     string
		wantName string
	}{
		{"/home/user/projects/services/auth", "auth"},
		{"/home/user/projects/libs/shared", "shared"},
		{"/absolute/path", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rp := NewResolvedProject(tt.path, nil)
			if rp.Name != tt.wantName {
				t.Errorf("NewResolvedProject(%q).Name = %q, want %q", tt.path, rp.Name, tt.wantName)
			}
		})
	}
}


