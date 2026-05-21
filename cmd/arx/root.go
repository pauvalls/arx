package main

import (
	"fmt"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/bootstrap"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/fs"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/infrastructure/preset"
	"github.com/pauvalls/arx/internal/ports"
	"github.com/spf13/cobra"
)

// rootCmd is the root command for the arx CLI
var rootCmd = &cobra.Command{
	Use:   "arx",
	Short: "Architecture audit CLI for cross-language projects",
	Long: `Arx is a cross-language architecture audit CLI that validates
architectural rules against real codebases and explains why violations
matter and how to fix them.

It is not a linter, not a static analyzer, and not a code quality tool.
It is an architecture guard with a teaching soul: every violation comes
with a didactic explanation that helps developers understand architectural
principles, not just fix a warning.

Arx supports Go and TypeScript projects out of the box, with a pluggable
detector system for additional languages.

Use 'arx init' to generate a configuration file for your project,
and 'arx check' to validate your architecture against the rules.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       VersionString(),
}

// Execute runs the root command and handles errors gracefully
func Execute() error {
	return rootCmd.Execute()
}

// newInitService creates an InitService with the default file writer and preset service
// presetLoader adapts the infrastructure preset.LoadPreset function to the PresetLoader interface.
type presetLoader struct{}

func (presetLoader) LoadPreset(name string) ([]byte, error) {
	return preset.LoadPreset(name)
}

func (presetLoader) ListPresets() []string {
	return preset.ListPresets()
}

func newInitService() *application.InitService {
	writer := fs.NewWalker(nil)
	presetService := application.NewPresetService(presetLoader{})
	return application.NewInitServiceWithPreset(writer, presetService)
}

// newCheckService creates a CheckService with all dependencies wired.
// If cache is nil, caching is disabled.
// If cfg is provided, plugin detectors from the config are included.
func newCheckService(format ports.OutputFormat, cache ports.Cache, cfgs ...*domain.Config) *application.CheckService {
	reader := config.NewYAMLReader()

	var detectors []ports.Detector
	if len(cfgs) > 0 && cfgs[0] != nil {
		detectors = bootstrap.BuildDetectorsWithPlugins(cfgs[0])
	} else {
		detectors = detector.GetDetectors()
	}

	var reporter ports.Reporter
	switch format {
	case ports.OutputFormatJSON:
		reporter = output.NewJSONReporter()
	case ports.OutputFormatSARIF:
		reporter = output.NewSARIFReporter()
	case ports.OutputFormatMarkdown:
		reporter = output.NewMarkdownReporter()
	case ports.OutputFormatJUnit:
		reporter = output.NewJUnitReporter()
	case ports.OutputFormatGitHubAnnotations:
		reporter = output.NewGitHubAnnotationsReporter()
	default:
		reporter = output.NewTerminalReporter()
	}

	return application.NewCheckServiceWithCache(reader, detectors, reporter, cache)
}

// printError prints a user-friendly error message
func printError(err error) {
	if err != nil {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "Error: %s\n", err.Error())
	}
}
