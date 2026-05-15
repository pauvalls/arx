package ports

import "github.com/pauvalls/arx/internal/domain"

// OutputFormat defines the output format for reports
type OutputFormat string

const (
	// OutputFormatTerminal outputs violations in human-readable terminal format
	OutputFormatTerminal OutputFormat = "terminal"
	// OutputFormatJSON outputs violations in JSON format
	OutputFormatJSON OutputFormat = "json"
	// OutputFormatSARIF outputs violations in SARIF 2.1.0 format
	OutputFormatSARIF OutputFormat = "sarif"
	// OutputFormatMarkdown outputs violations in Markdown format
	OutputFormatMarkdown OutputFormat = "markdown"
	// OutputFormatJUnit outputs violations in JUnit XML format
	OutputFormatJUnit OutputFormat = "junit"
	// OutputFormatGitHubAnnotations outputs violations as GitHub Actions workflow commands
	OutputFormatGitHubAnnotations OutputFormat = "annotations"
	// OutputFormatHTML outputs violations in HTML format
	OutputFormatHTML OutputFormat = "html"
)

// Reporter defines the interface for reporting violations
type Reporter interface {
	// Report outputs violations in the specified format
	Report(violations []domain.Violation, format OutputFormat) error
}
