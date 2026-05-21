package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/pauvalls/arx/internal/bootstrap"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
)

// testDetectors returns the real detectors for tests that need them.
var testDetectors = detector.GetDetectors()

// TestDoctorServiceCheckProjectRootExists tests project root check when directory exists
func TestDoctorServiceCheckProjectRootExists(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewDoctorService("test-version", nil, config.NewYAMLReader())

	result := service.checkProjectRoot(tmpDir)

	if !result.OK {
		t.Errorf("expected project root check to pass, got: %s", result.Message)
	}
	expectedMsg := "Project root: " + tmpDir
	if result.Message != expectedMsg {
		t.Errorf("expected message %q, got: %q", expectedMsg, result.Message)
	}
}

// TestDoctorServiceCheckProjectRootNotExists tests project root check when directory doesn't exist
func TestDoctorServiceCheckProjectRootNotExists(t *testing.T) {
	service := NewDoctorService("test-version", nil, config.NewYAMLReader())

	result := service.checkProjectRoot("/nonexistent/path")

	if result.OK {
		t.Error("expected project root check to fail for nonexistent path")
	}
	if result.Message == "" {
		t.Error("expected error message")
	}
}

// TestDoctorServiceCheckConfigFileValid tests config check with valid config
func TestDoctorServiceCheckConfigFileValid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	validConfig := []byte(`version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
`)
	if err := os.WriteFile(configPath, validConfig, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	service := NewDoctorService("test-version", nil, config.NewYAMLReader())
	result := service.checkConfigFile(tmpDir)

	if !result.OK {
		t.Errorf("expected config check to pass, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "Config valid: 1 layers, 1 rules") {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// TestDoctorServiceCheckConfigFileMissing tests config check when file is missing
func TestDoctorServiceCheckConfigFileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewDoctorService("test-version", nil, config.NewYAMLReader())

	result := service.checkConfigFile(tmpDir)

	if result.OK {
		t.Error("expected config check to fail for missing file")
	}
	if result.Message != "Config file not found: arx.yaml" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// TestDoctorServiceCheckConfigFileInvalid tests config check with invalid config
func TestDoctorServiceCheckConfigFileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	invalidConfig := []byte(`version: "1.0"
# Missing layers and rules
`)
	if err := os.WriteFile(configPath, invalidConfig, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	service := NewDoctorService("test-version", nil, config.NewYAMLReader())
	result := service.checkConfigFile(tmpDir)

	if result.OK {
		t.Error("expected config check to fail for invalid config")
	}
	if result.Message == "" {
		t.Error("expected error message")
	}
}

// TestDoctorServiceCheckDetectorsNoFiles tests detector check when no files found
func TestDoctorServiceCheckDetectorsNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	service := NewDoctorService("test-version", nil, config.NewYAMLReader())

	result := service.checkDetectors(tmpDir)

	if result.OK {
		t.Error("expected detector check to fail when no files found")
	}
	if result.Message == "" {
		t.Error("expected error message")
	}
}

// TestDoctorServiceCheckDetectorsWithGoFiles tests detector check with Go files
func TestDoctorServiceCheckDetectorsWithGoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod for Go detector
	goMod := []byte(`module github.com/test/project

go 1.23
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a Go file
	goDir := filepath.Join(tmpDir, "domain")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	goFile := []byte(`package domain

import "fmt"

func Hello() {
	fmt.Println("Hello")
}
`)
	if err := os.WriteFile(filepath.Join(goDir, "hello.go"), goFile, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create minimal arx.yaml
	configPath := filepath.Join(tmpDir, "arx.yaml")
	cfgData := []byte(`version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test
    from: domain
    to: [domain]
    type: Cannot
`)
	if err := os.WriteFile(configPath, cfgData, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	service := NewDoctorService("test-version", testDetectors, config.NewYAMLReader())
	result := service.checkDetectors(tmpDir)

	if !result.OK {
		t.Errorf("expected detector check to pass with Go files, got: %s", result.Message)
	}
}

// TestDoctorServiceCheckVersion tests version check
func TestDoctorServiceCheckVersion(t *testing.T) {
	service := NewDoctorService("v1.2.3", nil, config.NewYAMLReader())

	result := service.checkVersion()

	if !result.OK {
		t.Error("expected version check to pass")
	}
	if result.Message != "arx version: v1.2.3" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// TestDoctorServiceCheckAllChecksPassed tests full check with all passing
func TestDoctorServiceCheckAllChecksPassed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid config
	configPath := filepath.Join(tmpDir, "arx.yaml")
	cfgData := []byte(`version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test
    from: domain
    to: [domain]
    type: Cannot
`)
	if err := os.WriteFile(configPath, cfgData, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Create go.mod for Go detector
	goMod := []byte(`module github.com/test/project

go 1.23
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create Go file
	goDir := filepath.Join(tmpDir, "domain")
	if err := os.MkdirAll(goDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	goFile := []byte(`package domain`)
	if err := os.WriteFile(filepath.Join(goDir, "hello.go"), goFile, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	service := NewDoctorService("test-version", testDetectors, config.NewYAMLReader())
	result := service.Check(tmpDir)

	if !result.AllChecksPassed {
		t.Error("expected all checks to pass")
	}
	if !result.ProjectRoot.OK {
		t.Errorf("project root check failed: %s", result.ProjectRoot.Message)
	}
	if !result.ConfigFile.OK {
		t.Errorf("config check failed: %s", result.ConfigFile.Message)
	}
	if !result.Detectors.OK {
		t.Errorf("detectors check failed: %s", result.Detectors.Message)
	}
	if !result.Version.OK {
		t.Errorf("version check failed: %s", result.Version.Message)
	}
}

// TestDoctorServiceCheckAllChecksFailed tests full check with failures
func TestDoctorServiceCheckAllChecksFailed(t *testing.T) {
	tmpDir := t.TempDir()
	// No config, no source files

	service := NewDoctorService("test-version", nil, config.NewYAMLReader())
	result := service.Check(tmpDir)

	if result.AllChecksPassed {
		t.Error("expected some checks to fail")
	}
	if !result.ConfigFile.OK {
		// Expected - config missing
	}
	if !result.Detectors.OK {
		// Expected - no detectors
	}
}
