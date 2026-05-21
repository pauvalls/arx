package application

import (
	"context"
	"fmt"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"gopkg.in/yaml.v3"
)

// InitService wraps the Init use case functions with dependency injection.
// It provides a clean API for initializing Arx configuration in a project.
type InitService struct {
	writer       ports.FileWriter
	presetService ports.PresetService
}

// NewInitService creates a new InitService with the given FileWriter dependency.
func NewInitService(writer ports.FileWriter) *InitService {
	return &InitService{
		writer: writer,
	}
}

// NewInitServiceWithPreset creates a new InitService with both FileWriter and PresetService dependencies.
func NewInitServiceWithPreset(writer ports.FileWriter, presetService ports.PresetService) *InitService {
	return &InitService{
		writer:       writer,
		presetService: presetService,
	}
}

// Scan analyzes a project root and returns detected project information.
func (s *InitService) Scan(projectRoot string) (*ProjectInfo, error) {
	return ScanProject(projectRoot)
}

// Generate creates a default configuration based on project information.
func (s *InitService) Generate(projectInfo *ProjectInfo) (*domain.Config, error) {
	return GenerateConfig(projectInfo)
}

// Write persists a configuration to the specified path.
func (s *InitService) Write(config *domain.Config, outputPath string) error {
	return WriteConfig(config, outputPath, s.writer)
}

// Init runs the complete initialization workflow: scan, generate, and write.
func (s *InitService) Init(projectRoot, outputPath string) (*domain.Config, error) {
	return InitWithPreset(s, projectRoot, outputPath, "")
}

// InitWithPreset runs initialization with an optional preset template.
// If presetName is empty, uses automatic detection.
// Deprecated: Use s.InitWithPreset directly instead.
func InitWithPreset(s *InitService, projectRoot, outputPath, presetName string) (*domain.Config, error) {
	info, err := s.Scan(projectRoot)
	if err != nil {
		return nil, err
	}

	config, err := GenerateConfigWithPreset(info, presetName, s.presetService)
	if err != nil {
		return nil, err
	}

	if err := s.Write(config, outputPath); err != nil {
		return nil, err
	}

	return config, nil
}

// InitWithPreset initializes a project configuration using a named preset.
// It loads the preset via PresetService, generates a YAML file with a header comment,
// and writes it to outputPath. If the file exists and force is false, returns an error.
// Returns the generated configuration on success.
func (s *InitService) InitWithPreset(presetName, outputPath string, force bool) (*domain.Config, error) {
	if s.presetService == nil {
		return nil, fmt.Errorf("preset service not configured")
	}

	// Check if file exists and force is not set
	if !force && s.writer.Exists(outputPath) {
		return nil, fmt.Errorf("configuration file already exists: %s (use force=true to overwrite)", outputPath)
	}

	// Load preset configuration
	config, err := s.presetService.LoadPreset(presetName)
	if err != nil {
		return nil, fmt.Errorf("loading preset %q: %w", presetName, err)
	}

	// Serialize to YAML
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config to YAML: %w", err)
	}

	// Generate header with preset info and timestamp
	timestamp := time.Now().Format(time.RFC3339)
	header := fmt.Sprintf(`# Arx Architecture Configuration
# Preset: %s
# Generated: %s
# 
# ⚠️  This is a starting point. Review and customize before running 'arx check'.

`, presetName, timestamp)

	content := append([]byte(header), yamlBytes...)

	// Write file
	if err := s.writer.Write(outputPath, content); err != nil {
		return nil, fmt.Errorf("writing config file: %w", err)
	}

	return config, nil
}

// CheckService wraps the Check use case functions with dependency injection.
// It provides a clean API for running architecture checks on a project.
type CheckService struct {
	reader    ports.ConfigReader
	detectors []ports.Detector
	reporter  ports.Reporter
	cache     ports.Cache
	Jobs      int // max workers for detector concurrency (0 = unlimited)
}

// NewCheckService creates a new CheckService with the given dependencies.
func NewCheckService(reader ports.ConfigReader, detectors []ports.Detector, reporter ports.Reporter) *CheckService {
	return &CheckService{
		reader:    reader,
		detectors: detectors,
		reporter:  reporter,
	}
}

// NewCheckServiceWithCache creates a new CheckService with an optional cache.
func NewCheckServiceWithCache(reader ports.ConfigReader, detectors []ports.Detector, reporter ports.Reporter, cache ports.Cache) *CheckService {
	return &CheckService{
		reader:    reader,
		detectors: detectors,
		reporter:  reporter,
		cache:     cache,
	}
}

// Load reads and validates the configuration.
func (s *CheckService) Load(configPath string) (*domain.Config, error) {
	return LoadConfig(configPath, s.reader)
}

// Detect runs all applicable detectors and returns aggregated dependencies.
func (s *CheckService) Detect(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return RunDetectors(ctx, projectRoot, layers, s.detectors, s.Jobs)
}

// DetectWithStatus runs all detectors and returns per-detector status along with aggregated dependencies.
func (s *CheckService) DetectWithStatus(ctx context.Context, projectRoot string, layers []domain.Layer) (*DetectorResult, error) {
	return RunDetectorsWithStatus(ctx, projectRoot, layers, s.detectors, s.Jobs)
}

// DetectWithProfile runs all applicable detectors with profiling and returns performance data.
func (s *CheckService) DetectWithProfile(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, *domain.PerformanceReport, error) {
	return RunDetectorsWithProfile(ctx, projectRoot, layers, s.detectors, s.Jobs)
}

// DetectCached runs all applicable detectors with caching support.
// If cache is nil, falls back to Detect (backward compatible).
func (s *CheckService) DetectCached(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return RunDetectorsCached(ctx, projectRoot, layers, s.detectors, s.cache)
}

// DetectCachedWithStatus runs all applicable detectors with caching and returns per-detector status.
// If cache is nil, falls back to DetectWithStatus (backward compatible).
func (s *CheckService) DetectCachedWithStatus(ctx context.Context, projectRoot string, layers []domain.Layer) (*DetectorResult, error) {
	return RunDetectorsCachedWithStatus(ctx, projectRoot, layers, s.detectors, s.cache)
}

// Evaluate checks dependencies against rules and returns violations.
// userFuncs is an optional compiled user-function map (may be nil).
func (s *CheckService) Evaluate(dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer, userFuncs ...map[string]domain.Expr) []domain.Violation {
	return EvaluateArchitecture(dependencies, rules, layers, userFuncs...)
}

// Report outputs violations in the specified format.
func (s *CheckService) Report(violations []domain.Violation, format ports.OutputFormat) error {
	return GenerateReport(violations, format, s.reporter)
}

// Check runs the complete check workflow: load, detect, evaluate, and report.
func (s *CheckService) Check(ctx context.Context, configPath, projectRoot string, format ports.OutputFormat) error {
	config, err := s.Load(configPath)
	if err != nil {
		return err
	}

	// Cross-language detector is pre-appended to s.detectors by the composition root
	dependencies, err := RunDetectors(ctx, projectRoot, config.Layers, s.detectors, s.Jobs)
	if err != nil {
		return err
	}

	violations := s.Evaluate(dependencies, config.Rules, config.Layers, config.UserFunctions())

	if err := s.Report(violations, format); err != nil {
		return err
	}

	return nil
}
