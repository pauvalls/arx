package application

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// PRCheckService runs architecture checks on pull requests.
type PRCheckService struct {
	checkService *CheckService
	ghClient     ports.GitHubClient  // optional (nil for CLI-only mode)
	gitClient    ports.GitClient
	autoApprove  bool
}

// NewPRCheckService creates a new PRCheckService.
// NewPRCheckService creates a new PRCheckService.
// ghClient and gitClient are optional (nil is accepted).
// autoApprove enables automatic PR approval when no violations are found.
func NewPRCheckService(checkService *CheckService, ghClient ports.GitHubClient, autoApprove bool, gitClient ...ports.GitClient) *PRCheckService {
	s := &PRCheckService{
		checkService: checkService,
		ghClient:     ghClient,
		autoApprove:  autoApprove,
	}
	if len(gitClient) > 0 {
		s.gitClient = gitClient[0]
	}
	return s
}

// PRCheckResult holds the result of a PR check.
type PRCheckResult struct {
	NewViolations []domain.Violation
	ResolvedCount int
	Passed        bool
	CheckRunID    int64
	Summary       string
}

// RunPRCheck runs a PR check from a webhook (implements ports.PRCheckRunner).
func (s *PRCheckService) RunPRCheck(owner, repo string, pr domain.PRInfo) error {

	result, err := s.Run(context.Background(), pr)
	if err != nil {
		return fmt.Errorf("PR check failed: %w", err)
	}

	// Create/update check run via GitHub API
	if s.ghClient != nil && result.CheckRunID > 0 {
		conclusion := domain.CheckRunSuccess
		if !result.Passed {
			conclusion = domain.CheckRunFailure
		}
		output := domain.CheckRunOutput{
			Title:   "Arx Architecture Check",
			Summary: result.Summary,
		}
		if err := s.ghClient.UpdateCheckRun(owner, repo, result.CheckRunID, conclusion, output); err != nil {
			return fmt.Errorf("updating check run: %w", err)
		}
	}

	// Auto-approve if enabled and no violations
	if s.ghClient != nil && s.autoApprove && result.Passed {
		if err := s.ghClient.ApprovePR(owner, repo, pr.PRNumber); err != nil {
			return fmt.Errorf("auto-approve failed: %w", err)
		}
	}

	return nil
}

// Run runs a complete PR check cycle.
// 1. Get diff via git CLI
// 2. Run full check on the working tree (assumes checked out to head)
// 3. Parse diff, filter violations to PR-introduced only
// 4. Create check run if ghClient provided
func (s *PRCheckService) Run(ctx context.Context, pr domain.PRInfo) (*PRCheckResult, error) {
	if err := pr.Validate(); err != nil {
		return nil, fmt.Errorf("invalid PR info: %w", err)
	}

	// Get diff between base and head
	var diffOutput string
	var err error

	if s.gitClient != nil {
		diffOutput, err = s.gitClient.Diff(ctx, pr.BaseSHA, pr.HeadSHA, pr.RepoPath)
		if err != nil {
			// Fallback: try git diff-tree via Run
			diffOutput, err = s.gitClient.Run(ctx, pr.RepoPath, "diff-tree", "--no-commit-id", "-r", "-p", pr.BaseSHA, pr.HeadSHA)
			if err != nil {
				return nil, fmt.Errorf("getting git diff: %w", err)
			}
		}
	} else {
		diffOutput, err = GetGitDiff(pr.RepoPath, pr.BaseSHA, pr.HeadSHA)
		if err != nil {
			// Fallback: try git diff-tree
			diffOutput, err = GetGitDiffTree(pr.RepoPath, pr.BaseSHA, pr.HeadSHA)
			if err != nil {
				return nil, fmt.Errorf("getting git diff: %w", err)
			}
		}
	}

	// Parse diff
	diffSummary, err := ParseDiff(diffOutput)
	if err != nil {
		return nil, fmt.Errorf("parsing diff: %w", err)
	}

	// Run full architecture check
	configPath := filepath.Join(pr.RepoPath, "arx.yaml")
	config, err := s.checkService.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	dependencies, err := s.checkService.Detect(ctx, pr.RepoPath, config.Layers)
	if err != nil {
		return nil, fmt.Errorf("detecting dependencies: %w", err)
	}

	allViolations := s.checkService.Evaluate(dependencies, config.Rules, config.Layers)

	// Filter to PR-introduced violations
	newViolations := FilterViolationsForDiff(allViolations, diffSummary)

	result := &PRCheckResult{
		NewViolations: newViolations,
		Passed:        len(newViolations) == 0,
	}

	// Build summary
	if result.Passed {
		result.Summary = fmt.Sprintf("✅ No new architecture violations in this PR (%d total violations unchanged).", len(allViolations))
	} else {
		result.Summary = fmt.Sprintf("❌ %d new architecture violation(s) found in this PR.", len(newViolations))
	}

	// Create check run if GitHub client is available
	if s.ghClient != nil {
		output := buildCheckRunOutput(result, allViolations)
		checkRunID, err := s.ghClient.CreateCheckRun("", "", pr, output)
		if err != nil {
			// Non-fatal: we still want to return the result
			return result, nil
		}
		result.CheckRunID = checkRunID
	}

	return result, nil
}

// buildCheckRunOutput creates a CheckRunOutput from the PR check result.
func buildCheckRunOutput(result *PRCheckResult, allViolations []domain.Violation) domain.CheckRunOutput {
	output := domain.CheckRunOutput{
		Title:   "Arx Architecture Check",
		Summary: result.Summary,
		Text:    fmt.Sprintf("Total violations in repository: %d\nNew in this PR: %d", len(allViolations), len(result.NewViolations)),
	}

	for _, v := range result.NewViolations {
		output.Annotations = append(output.Annotations, domain.CheckRunAnnotation{
			Path:            v.File,
			StartLine:       v.Line,
			EndLine:         v.Line,
			AnnotationLevel: "failure",
			Message:         v.Message,
			Title:           fmt.Sprintf("[%s] %s → %s", v.RuleID, v.SourceLayer, v.TargetLayer),
		})
	}

	return output
}

// hunkHeaderRE parses unified diff hunk headers: @@ -old,count +new,count @@
var hunkHeaderRE = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// ParseDiff parses a unified diff string and returns a PRDiffSummary.
func ParseDiff(diffOutput string) (*domain.PRDiffSummary, error) {
	if diffOutput == "" {
		return &domain.PRDiffSummary{
			Hunks: nil,
			Stats: map[string]int{
				"files":      0,
				"insertions": 0,
				"deletions":  0,
			},
		}, nil
	}

	lines := strings.Split(diffOutput, "\n")
	var hunks []domain.DiffHunk
	var currentFile string
	filesSet := make(map[string]bool)
	insertions := 0
	deletions := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Detect file headers
		if strings.HasPrefix(line, "diff --git ") {
			// Extract file path from the last "b/path" part
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				// The last part is the second file path (b/path)
				last := parts[len(parts)-1]
				currentFile = strings.TrimPrefix(last, "b/")
				filesSet[currentFile] = true
			}
			continue
		}

		// Skip index lines
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			// "new file" and "deleted file" tell us about file mode but we handle it
			continue
		}

		// Parse hunk header
		if strings.HasPrefix(line, "@@") {
			matches := hunkHeaderRE.FindStringSubmatch(line)
			if len(matches) < 3 {
				continue
			}
			oldL, _ := strconv.Atoi(matches[1])
			newL, _ := strconv.Atoi(matches[2])

			// Parse the hunk body — advance line by line
			for i+1 < len(lines) {
				i++
				hunkLine := lines[i]

				// Stop when we hit the next hunk or file header
				if strings.HasPrefix(hunkLine, "@@") ||
					strings.HasPrefix(hunkLine, "diff --git ") {
					// Backtrack so the outer loop processes this as a header
					i--
					break
				}

				// Skip empty lines with no content (but still track context)
				hunk := domain.DiffHunk{
					File:    currentFile,
					Content: hunkLine,
				}

				if len(hunkLine) > 0 {
					switch hunkLine[0] {
					case '+':
						hunk.OldLine = 0
						hunk.NewLine = newL
						newL++
						insertions++
					case '-':
						hunk.OldLine = oldL
						hunk.NewLine = 0
						oldL++
						deletions++
					default:
						// Context line (including '\ No newline at end of file')
						hunk.OldLine = oldL
						hunk.NewLine = newL
						oldL++
						newL++
					}
				} else {
					// Empty line in diff — context line
					hunk.OldLine = oldL
					hunk.NewLine = newL
					oldL++
					newL++
				}

				hunks = append(hunks, hunk)
			}
			continue
		}
	}

	return &domain.PRDiffSummary{
		Hunks: hunks,
		Stats: map[string]int{
			"files":      len(filesSet),
			"insertions": insertions,
			"deletions":  deletions,
		},
	}, nil
}

// GetGitDiff runs git diff to get the diff between two refs.
func GetGitDiff(repoPath, baseSHA, headSHA string) (string, error) {
	if repoPath == "" {
		repoPath = "."
	}

	cmd := exec.Command("git", "diff", "--unified=0", baseSHA, headSHA)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return string(output), nil
}

// GetGitDiffTree runs git diff-tree as a fallback.
func GetGitDiffTree(repoPath, baseSHA, headSHA string) (string, error) {
	if repoPath == "" {
		repoPath = "."
	}

	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "-r", "-p", baseSHA, headSHA)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff-tree failed: %w", err)
	}

	return string(output), nil
}

// FilterViolationsForDiff filters violations to only those on lines present in the diff.
// New files: all violations in that file are included.
// Deleted files: violations are skipped entirely.
// Modified files: only violations on new/changed lines are included.
func FilterViolationsForDiff(violations []domain.Violation, diff *domain.PRDiffSummary) []domain.Violation {
	if len(violations) == 0 {
		return nil
	}

	if diff == nil || len(diff.Hunks) == 0 {
		return nil
	}

	// Build a set of (file, line) pairs present in the diff
	type fileLine struct {
		file string
		line int
	}

	diffLines := make(map[fileLine]bool)
	newFiles := make(map[string]bool)
	deletedFiles := make(map[string]bool)

	for _, hunk := range diff.Hunks {
		key := fileLine{file: hunk.File, line: hunk.NewLine}
		if hunk.NewLine > 0 {
			diffLines[key] = true
		}
		if hunk.NewLine > 0 && hunk.OldLine == 0 {
			// This is a new file line (addition)
			newFiles[hunk.File] = true
		}
		if hunk.NewLine == 0 && hunk.OldLine > 0 {
			// This is a deletion
			deletedFiles[hunk.File] = true
		}
	}

	// For new files, all lines are in the diff
	newFileLines := make(map[string]bool)
	for _, hunk := range diff.Hunks {
		if hunk.OldLine == 0 && hunk.NewLine > 0 {
			newFileLines[hunk.File] = true
		}
	}

	var filtered []domain.Violation
	for _, v := range violations {
		// Skip violations in deleted files
		if deletedFiles[v.File] {
			continue
		}

		// Check if the violation is on a diff line
		key := fileLine{file: v.File, line: v.Line}
		if diffLines[key] {
			// For new files, always include
			if newFiles[v.File] {
				filtered = append(filtered, v)
				continue
			}
			// For modified files, only include if it's a new line
			// Check if this specific line is an addition (old line was 0)
			isNewLine := false
			for _, hunk := range diff.Hunks {
				if hunk.File == v.File && hunk.NewLine == v.Line && hunk.OldLine == 0 {
					isNewLine = true
					break
				}
			}
			if isNewLine {
				filtered = append(filtered, v)
			}
		}
	}

	return filtered
}


