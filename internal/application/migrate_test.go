package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestDetectVersion(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    domain.SchemaVersion
		wantErr bool
	}{
		{
			name:    "version 1.0",
			data:    []byte("version: \"1.0\"\nlayers: []\n"),
			want:    domain.SchemaVersion{Major: 1, Minor: 0},
			wantErr: false,
		},
		{
			name:    "version 2.0",
			data:    []byte("version: \"2.0\"\nlayers: []\n"),
			want:    domain.SchemaVersion{Major: 2, Minor: 0},
			wantErr: false,
		},
		{
			name:    "no version",
			data:    []byte("layers: []\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectVersion(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("DetectVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMigrateService_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	data := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\nrules: []\n")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reg := domain.NewRegistry()
	reg.Register(domain.Migration{
		From: domain.SchemaVersion{Major: 1, Minor: 0},
		To:   domain.SchemaVersion{Major: 2, Minor: 0},
		Func: nopYAMLMigration,
	})

	svc := NewMigrateService(reg)
	result, err := svc.Migrate(configPath, domain.SchemaVersion{Major: 2, Minor: 0}, true)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if !result.DryRun {
		t.Error("expected dry run to be true")
	}
	if result.BackupPath != "" {
		t.Error("dry run should not create backup path")
	}
	if len(result.Steps) == 0 {
		t.Error("expected migration steps")
	}

	// Verify file was NOT modified
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(content) != string(data) {
		t.Error("dry run modified the config file")
	}
}

func TestMigrateService_BackupCreated(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	data := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\nrules: []\n")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reg := domain.NewRegistry()
	reg.Register(domain.Migration{
		From: domain.SchemaVersion{Major: 1, Minor: 0},
		To:   domain.SchemaVersion{Major: 2, Minor: 0},
		Func: nopYAMLMigration,
	})

	svc := NewMigrateService(reg)
	result, err := svc.Migrate(configPath, domain.SchemaVersion{Major: 2, Minor: 0}, false)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if result.BackupPath == "" {
		t.Fatal("expected backup path to be set")
	}
	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Fatal("backup file was not created")
	}

	// Verify backup contains original content
	backupData, err := os.ReadFile(result.BackupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupData) != string(data) {
		t.Error("backup content does not match original")
	}
}

func TestMigrateService_AlreadyAtTarget(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")

	data := []byte("version: \"1.0\"\nlayers:\n  - name: domain\n    paths: [\"./domain\"]\nrules: []\n")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reg := domain.NewRegistry()
	svc := NewMigrateService(reg)

	result, err := svc.Migrate(configPath, domain.SchemaVersion{Major: 1, Minor: 0}, true)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if len(result.Steps) != 1 || result.Steps[0] != "already at version 1.0" {
		t.Errorf("expected 'already at version' message, got %v", result.Steps)
	}
}

// nopYAMLMigration is a no-op migration function for testing that preserves YAML.
func nopYAMLMigration(input []byte) ([]byte, error) {
	return input, nil
}
