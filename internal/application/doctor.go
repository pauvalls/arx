package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/ports"
)

// CheckResult represents the result of a single diagnostic check
type CheckResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// DoctorResult represents the complete diagnostic result
type DoctorResult struct {
	ProjectRoot     CheckResult `json:"project_root"`
	ConfigFile      CheckResult `json:"config_file"`
	Detectors       CheckResult `json:"detectors"`
	GitStatus       CheckResult `json:"git_status"`
	Version         CheckResult `json:"version"`
	AllChecksPassed bool        `json:"all_checks_passed"`
}

// DoctorService runs diagnostic checks on an arx project
type DoctorService struct {
	version      string
	detectors    []ports.Detector
	configReader ports.ConfigReader
	gitClient    ports.GitClient
}

// NewDoctorService creates a new DoctorService with the given dependencies.
// gitClient is optional (nil is accepted, git checks will be skipped).
func NewDoctorService(version string, detectors []ports.Detector, configReader ports.ConfigReader, gitClient ...ports.GitClient) *DoctorService {
	s := &DoctorService{
		version:      version,
		detectors:    detectors,
		configReader: configReader,
	}
	if len(gitClient) > 0 {
		s.gitClient = gitClient[0]
	}
	return s
}

// Check runs all diagnostic checks on the given project root
func (s *DoctorService) Check(projectRoot string) DoctorResult {
	result := DoctorResult{
		AllChecksPassed: true,
	}

	// Check 1: Project root exists
	result.ProjectRoot = s.checkProjectRoot(projectRoot)
	if !result.ProjectRoot.OK {
		result.AllChecksPassed = false
	}

	// Check 2: Config file exists and is valid
	result.ConfigFile = s.checkConfigFile(projectRoot)
	if !result.ConfigFile.OK {
		result.AllChecksPassed = false
	}

	// Check 3: Detectors can find files
	result.Detectors = s.checkDetectors(projectRoot)
	if !result.Detectors.OK {
		result.AllChecksPassed = false
	}

	// Check 4: Git status (if repo)
	result.GitStatus = s.checkGitStatus(projectRoot)
	// Git check failure doesn't fail overall (project might not be in git)

	// Check 5: Version info
	result.Version = s.checkVersion()

	return result
}

// checkProjectRoot verifies the project root exists and is accessible
func (s *DoctorService) checkProjectRoot(projectRoot string) CheckResult {
	info, err := os.Stat(projectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return CheckResult{OK: false, Message: fmt.Sprintf("Project root does not exist: %s", projectRoot)}
		}
		return CheckResult{OK: false, Message: fmt.Sprintf("Cannot access project root: %v", err)}
	}

	if !info.IsDir() {
		return CheckResult{OK: false, Message: "Project root is not a directory"}
	}

	return CheckResult{OK: true, Message: fmt.Sprintf("Project root: %s", projectRoot)}
}

// checkConfigFile verifies the config file exists and is valid
func (s *DoctorService) checkConfigFile(projectRoot string) CheckResult {
	configPath := filepath.Join(projectRoot, "arx.yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return CheckResult{OK: false, Message: "Config file not found: arx.yaml"}
	}

	// Try to read and validate using the injected config reader
	cfg, err := s.configReader.Read(configPath)
	if err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Failed to read config: %v", err)}
	}

	if err := s.configReader.Validate(cfg); err != nil {
		return CheckResult{OK: false, Message: fmt.Sprintf("Config validation error: %v", err)}
	}

	return CheckResult{OK: true, Message: fmt.Sprintf("Config valid: %d layers, %d rules", len(cfg.Layers), len(cfg.Rules))}
}

// checkDetectors verifies detectors can find files
func (s *DoctorService) checkDetectors(projectRoot string) CheckResult {
	ctx := context.Background()
	detectedCount := 0
	var detectedLanguages []string

	for _, det := range s.detectors {
		applicable, err := det.Detect(ctx, projectRoot)
		if err != nil {
			continue // Skip detectors that fail
		}
		if applicable {
			detectedCount++
			detectedLanguages = append(detectedLanguages, det.Name())
		}
	}

	if detectedCount == 0 {
		return CheckResult{
			OK:      false,
			Message: "No language detectors found applicable files (Go, TypeScript, etc.)",
		}
	}

	return CheckResult{
		OK:      true,
		Message: fmt.Sprintf("Detected %d language(s): %s", detectedCount, strings.Join(detectedLanguages, ", ")),
	}
}

// checkGitStatus checks if the project is in a git repository and its status
func (s *DoctorService) checkGitStatus(projectRoot string) CheckResult {
	if s.gitClient == nil {
		return CheckResult{OK: false, Message: "Git client not configured"}
	}

	// Check if git is available
	if !s.gitClient.CheckGitInstalled() {
		return CheckResult{OK: false, Message: "Git not installed"}
	}

	// Check if in git repo
	ctx := context.Background()
	_, err := s.gitClient.Run(ctx, projectRoot, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return CheckResult{OK: true, Message: "Not a git repository"}
	}

	// Get current branch
	branchOutput, err := s.gitClient.Run(ctx, projectRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return CheckResult{OK: false, Message: "Failed to get git branch"}
	}
	branch := strings.TrimSpace(branchOutput)

	// Check for uncommitted changes
	statusOutput, err := s.gitClient.Status(ctx, projectRoot)
	if err != nil {
		return CheckResult{OK: false, Message: "Failed to get git status"}
	}

	if len(strings.TrimSpace(statusOutput)) > 0 {
		return CheckResult{OK: true, Message: fmt.Sprintf("Git: %s (dirty)", branch)}
	}

	return CheckResult{OK: true, Message: fmt.Sprintf("Git: %s (clean)", branch)}
}

// checkVersion returns the arx version
func (s *DoctorService) checkVersion() CheckResult {
	return CheckResult{
		OK:      true,
		Message: fmt.Sprintf("arx version: %s", s.version),
	}
}
