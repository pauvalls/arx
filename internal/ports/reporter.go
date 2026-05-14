package ports

import "github.com/pauvalls/arx/internal/domain"

// OutputFormat defines the output format for reports
type OutputFormat string

const (
	// OutputFormatTerminal outputs violations in human-readable terminal format
	OutputFormatTerminal OutputFormat = "terminal"
	// OutputFormatJSON outputs violations in JSON format
	OutputFormatJSON OutputFormat = "json"
)

// Reporter defines the interface for reporting violations
type Reporter interface {
	// Report outputs violations in the specified format
	Report(violations []domain.Violation, format OutputFormat) error
}
