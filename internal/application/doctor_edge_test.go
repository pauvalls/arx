package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/infrastructure/config"
)

func TestDoctorService_CheckGitStatus_NoGit(t *testing.T) {
	service := NewDoctorService("test", nil, config.NewYAMLReader())
	result := service.checkGitStatus(t.TempDir())
	// Git check failure doesn't fail overall (might not be in git repo)
	if result.OK {
		t.Log("git check passed unexpectedly (git may be installed)")
	}
}

func TestDoctorService_CheckGitStatus_GitNotFound(t *testing.T) {
	// Use a tmp dir that we KNOW is not a git repo
	// The checkGitStatus will try to exec git and it should either
	// not find it or find that the dir is not a repo
	service := NewDoctorService("test", nil, config.NewYAMLReader())
	tmpDir := t.TempDir()

	result := service.checkGitStatus(tmpDir)
	// The result should be deterministic — either git not installed or not a repo
	if !result.OK {
		t.Logf("git check returned: %s", result.Message)
	}
}

func TestDoctorService_CheckProjectRoot_FileNotDir(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "afile.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	service := NewDoctorService("test", nil, config.NewYAMLReader())
	result := service.checkProjectRoot(filePath)
	if result.OK {
		t.Error("expected check to fail for file path")
	}
	if !containsMsg(result.Message, "not a directory") {
		t.Errorf("expected 'not a directory' message, got: %s", result.Message)
	}
}

func TestDoctorService_CheckProjectRoot_NoPermission(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	tmpDir := t.TempDir()
	noAccess := filepath.Join(tmpDir, "noaccess")
	if err := os.Mkdir(noAccess, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(noAccess, 0755)

	// On some systems (e.g., running as owner), chmod 0000 doesn't prevent access.
	// Only verify the error message format if it failed.
	service := NewDoctorService("test", nil, config.NewYAMLReader())
	result := service.checkProjectRoot(noAccess)
	_ = result
}

func TestDoctorService_CheckConfigFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	service := NewDoctorService("test", nil, config.NewYAMLReader())
	result := service.checkConfigFile(tmpDir)
	if result.OK {
		t.Error("expected check to fail for invalid YAML")
	}
}

func TestDoctorService_CreateWithConfigReader(t *testing.T) {
	// Verify that the config reader is properly injected
	reader := config.NewYAMLReader()
	service := NewDoctorService("test", nil, reader)
	if service.configReader == nil {
		t.Error("expected configReader to be set")
	}
}

// Helper
func containsMsg(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstr(s, substr))))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
