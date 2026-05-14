package application

import (
	"context"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// InitService wraps the Init use case functions with dependency injection.
// It provides a clean API for initializing Arx configuration in a project.
type InitService struct {
	writer ports.FileWriter
}

// NewInitService creates a new InitService with the given FileWriter dependency.
func NewInitService(writer ports.FileWriter) *InitService {
	return &InitService{
		writer: writer,
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
	info, err := s.Scan(projectRoot)
	if err != nil {
		return nil, err
	}

	config, err := s.Generate(info)
	if err != nil {
		return nil, err
	}

	if err := s.Write(config, outputPath); err != nil {
		return nil, err
	}

	return config, nil
}

// Writer returns the FileWriter dependency for use in InitWithPreset.
func (s *InitService) Writer() ports.FileWriter {
	return s.writer
}

// CheckService wraps the Check use case functions with dependency injection.
// It provides a clean API for running architecture checks on a project.
type CheckService struct {
	reader    ports.ConfigReader
	detectors []ports.Detector
	reporter  ports.Reporter
}

// NewCheckService creates a new CheckService with the given dependencies.
func NewCheckService(reader ports.ConfigReader, detectors []ports.Detector, reporter ports.Reporter) *CheckService {
	return &CheckService{
		reader:    reader,
		detectors: detectors,
		reporter:  reporter,
	}
}

// Load reads and validates the configuration.
func (s *CheckService) Load(configPath string) (*domain.Config, error) {
	return LoadConfig(configPath, s.reader)
}

// Detect runs all applicable detectors and returns aggregated dependencies.
func (s *CheckService) Detect(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	return RunDetectors(ctx, projectRoot, layers, s.detectors)
}

// Evaluate checks dependencies against rules and returns violations.
func (s *CheckService) Evaluate(dependencies []domain.Dependency, rules []domain.Rule, layers []domain.Layer) []domain.Violation {
	return EvaluateArchitecture(dependencies, rules, layers)
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

	dependencies, err := s.Detect(ctx, projectRoot, config.Layers)
	if err != nil {
		return err
	}

	violations := s.Evaluate(dependencies, config.Rules, config.Layers)

	if err := s.Report(violations, format); err != nil {
		return err
	}

	return nil
}
