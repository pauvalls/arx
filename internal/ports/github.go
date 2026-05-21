package ports

import "github.com/pauvalls/arx/internal/domain"

// GitHubClient defines the interface for GitHub API operations used by PR checks.
type GitHubClient interface {
	// CreateCheckRun creates a new check run and returns its ID.
	CreateCheckRun(owner, repo string, pr domain.PRInfo, output domain.CheckRunOutput) (int64, error)
	// UpdateCheckRun updates an existing check run with a conclusion and output.
	UpdateCheckRun(owner, repo string, checkRunID int64, conclusion domain.CheckRunConclusion, output domain.CheckRunOutput) error
	// ApprovePR approves the given pull request.
	ApprovePR(owner, repo string, prNumber int) error
}

// PRCheckRunner is the interface for running PR checks (avoids import cycle).
type PRCheckRunner interface {
	RunPRCheck(owner, repo string, pr domain.PRInfo) error
}
