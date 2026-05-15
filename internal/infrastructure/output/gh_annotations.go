package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// GitHubAnnotationsReporter implements ports.Reporter for GitHub Actions workflow commands
type GitHubAnnotationsReporter struct {
	tool string
}

// NewGitHubAnnotationsReporter creates a new GitHub Annotations reporter
func NewGitHubAnnotationsReporter() *GitHubAnnotationsReporter {
	return &GitHubAnnotationsReporter{
		tool: "arx",
	}
}

// Report implements ports.Reporter interface
// Outputs violations as GitHub Actions workflow commands:
//   - Error: ::error file=path,line=N,title=Rule ID::message
//   - Warning: ::warning file=path,line=N,title=Rule ID::message
//   - Info: ::notice file=path,line=N,title=Rule ID::message
func (r *GitHubAnnotationsReporter) Report(violations []domain.Violation, format ports.OutputFormat) error {
	for _, v := range violations {
		cmd := r.formatViolation(v)
		fmt.Fprintln(os.Stdout, cmd)
	}
	return nil
}

// formatViolation formats a single violation as a GitHub workflow command
func (r *GitHubAnnotationsReporter) formatViolation(v domain.Violation) string {
	// Determine command type based on severity
	var cmdType string
	switch v.Severity {
	case domain.SeverityWarning:
		cmdType = "warning"
	case domain.SeverityInfo:
		cmdType = "notice"
	default:
		cmdType = "error"
	}

	// Escape file path for workflow command
	escapedFile := escapeWorkflowParam(v.File)

	// Build title from rule ID (truncate to 50 chars if needed)
	title := v.RuleID
	if len(title) > 50 {
		title = title[:50]
	}
	escapedTitle := escapeWorkflowParam(title)

	// Escape message for workflow command
	escapedMessage := escapeWorkflowMessage(v.Message)

	// Format: ::{type} file={file},line={line},title={title}::{message}
	return fmt.Sprintf("::%s file=%s,line=%d,title=%s::%s",
		cmdType,
		escapedFile,
		v.Line,
		escapedTitle,
		escapedMessage,
	)
}

// escapeWorkflowParam escapes special characters in workflow command parameters
// Per GitHub Actions docs: % -> %25, \r -> %0D, \n -> %0A
func escapeWorkflowParam(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// escapeWorkflowMessage escapes special characters in workflow command messages
// Also escapes control characters that could break the command syntax
func escapeWorkflowMessage(s string) string {
	// First escape percent signs
	s = strings.ReplaceAll(s, "%", "%25")
	// Escape carriage returns and newlines
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// buildAnnotations constructs all annotations from violations (used for testing)
func (r *GitHubAnnotationsReporter) buildAnnotations(violations []domain.Violation) []string {
	annotations := make([]string, 0, len(violations))
	for _, v := range violations {
		annotations = append(annotations, r.formatViolation(v))
	}
	return annotations
}
