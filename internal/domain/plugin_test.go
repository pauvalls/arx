package domain

import (
	"strings"
	"testing"
)

func TestPluginConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  PluginConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plugin",
			plugin: PluginConfig{
				Name:      "dart-detector",
				Command:   "dart run bin/detect.dart",
				Languages: []string{"dart"},
			},
			wantErr: false,
		},
		{
			name: "valid plugin with all fields",
			plugin: PluginConfig{
				Name:       "elixir-detector",
				Command:    "mix run detect.exs",
				Args:       []string{"--verbose"},
				Languages:  []string{"elixir"},
				Timeout:    "60s",
				Extensions: []string{".ex", ".exs"},
			},
			wantErr: false,
		},
		{
			name: "valid plugin with multiple languages",
			plugin: PluginConfig{
				Name:      "web-detector",
				Command:   "node detector.mjs",
				Languages: []string{"javascript", "typescript"},
			},
			wantErr: false,
		},
		{
			name:    "missing name",
			plugin:  PluginConfig{Command: "test", Languages: []string{"go"}},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name:    "empty command",
			plugin:  PluginConfig{Name: "test", Languages: []string{"go"}},
			wantErr: true,
			errMsg:  "command is required",
		},
		{
			name:    "no languages",
			plugin:  PluginConfig{Name: "test", Command: "test"},
			wantErr: true,
			errMsg:  "at least one language must be specified",
		},
		{
			name:    "invalid name with leading digit",
			plugin:  PluginConfig{Name: "1test", Command: "test", Languages: []string{"go"}},
			wantErr: true,
			errMsg:  "must match",
		},
		{
			name:    "invalid name with special chars",
			plugin:  PluginConfig{Name: "test@detector", Command: "test", Languages: []string{"go"}},
			wantErr: true,
			errMsg:  "must match",
		},
		{
			name:    "name conflicts with builtin go",
			plugin:  PluginConfig{Name: "go", Command: "test", Languages: []string{"custom"}},
			wantErr: true,
			errMsg:  "conflicts with built-in detector",
		},
		{
			name:    "name conflicts with builtin python",
			plugin:  PluginConfig{Name: "python", Command: "test", Languages: []string{"custom"}},
			wantErr: true,
			errMsg:  "conflicts with built-in detector",
		},
		{
			name:    "invalid timeout",
			plugin:  PluginConfig{Name: "test", Command: "test", Languages: []string{"go"}, Timeout: "not-a-duration"},
			wantErr: true,
			errMsg:  "invalid timeout",
		},
		{
			name:    "valid timeout",
			plugin:  PluginConfig{Name: "test", Command: "test", Languages: []string{"go"}, Timeout: "30s"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPluginConfigInConfig_Validate(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion{Major: 1, Minor: 0},
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Plugins: []PluginConfig{
			{Name: "my-detector", Command: "my-detect", Languages: []string{"custom"}},
			{Name: "", Command: "bad", Languages: []string{"custom"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid plugin, got nil")
	}
	if !strings.Contains(err.Error(), "plugins[1]") {
		t.Errorf("error should reference plugins[1], got: %v", err)
	}
}

func TestPluginConfigInConfig_DuplicateNames(t *testing.T) {
	cfg := Config{
		Version: SchemaVersion{Major: 1, Minor: 0},
		Layers: []Layer{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Plugins: []PluginConfig{
			{Name: "dup", Command: "test1", Languages: []string{"a"}},
			{Name: "dup", Command: "test2", Languages: []string{"b"}},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate plugin names, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate plugin name") {
		t.Errorf("error should mention duplicate name, got: %v", err)
	}
}
