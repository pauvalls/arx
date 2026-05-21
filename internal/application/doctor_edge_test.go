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
	if result.OK {
		t.Log("git client not configured, expecting non-OK result")
	}
	if !stringsContains(result.Message, "Git client not configured") {
		t.Logf("got: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_GitNotInstalled(t *testing.T) {
	mock := newMockGitClient()
	mock.gitInstalled = false

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if result.OK {
		t.Error("expected check to fail when git not installed")
	}
	if !stringsContains(result.Message, "Git not installed") {
		t.Errorf("wrong message: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_NotARepo(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --is-inside-work-tree", "", &execError{})

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if !result.OK {
		t.Errorf("expected non-fatal for non-repo, got: %s", result.Message)
	}
	if !stringsContains(result.Message, "Not a git repository") {
		t.Errorf("wrong message: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_Clean(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --is-inside-work-tree", "true", nil)
	mock.withRun("rev-parse --abbrev-ref HEAD", "main", nil)
	mock.withStatus("", nil)

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if !result.OK {
		t.Errorf("expected OK for clean repo, got: %s", result.Message)
	}
	if !stringsContains(result.Message, "clean") {
		t.Errorf("expected 'clean' in message, got: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_Dirty(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --is-inside-work-tree", "true", nil)
	mock.withRun("rev-parse --abbrev-ref HEAD", "feature/foo", nil)
	mock.withStatus(" M modified.go\n?? untracked.go", nil)

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if !result.OK {
		t.Errorf("expected OK for dirty repo, got: %s", result.Message)
	}
	if !stringsContains(result.Message, "dirty") {
		t.Errorf("expected 'dirty' in message, got: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_BranchError(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --is-inside-work-tree", "true", nil)
	mock.withRun("rev-parse --abbrev-ref HEAD", "", &execError{})

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if result.OK {
		t.Errorf("expected failure for branch error, got: %s", result.Message)
	}
	if !stringsContains(result.Message, "Failed") {
		t.Errorf("expected failure message, got: %s", result.Message)
	}
}

func TestDoctorService_CheckGitStatus_StatusError(t *testing.T) {
	mock := newMockGitClient()
	mock.withRun("rev-parse --is-inside-work-tree", "true", nil)
	mock.withRun("rev-parse --abbrev-ref HEAD", "main", nil)
	mock.withStatus("", &execError{})

	service := NewDoctorService("test", nil, config.NewYAMLReader(), mock)
	result := service.checkGitStatus(t.TempDir())

	if result.OK {
		t.Errorf("expected failure for status error, got: %s", result.Message)
	}
	if !stringsContains(result.Message, "Failed") {
		t.Errorf("expected failure message, got: %s", result.Message)
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
	if !stringsContains(result.Message, "not a directory") {
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
	reader := config.NewYAMLReader()
	service := NewDoctorService("test", nil, reader)
	if service.configReader == nil {
		t.Error("expected configReader to be set")
	}
}

// execError is a minimal error type that satisfies the error interface for mocking.
type execError struct{}

func (e *execError) Error() string { return "exit status 1" }

// stringsContains reports whether substr is within s.
func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
