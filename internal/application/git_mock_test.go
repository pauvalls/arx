package application

import (
	"context"
	"fmt"
	"strings"
)

// mockGitClient implements ports.GitClient for testing.
type mockGitClient struct {
	// Stored responses keyed by a combination of method + args.
	diffResult   map[diffKey]diffResponse
	statusResult string
	statusErr    error
	runResults   map[string]runResponse
	gitInstalled bool
}

type diffKey struct {
	baseRef string
	headRef string
	repo    string
}

type diffResponse struct {
	diff string
	err  error
}

type runResponse struct {
	output string
	err    error
}

func newMockGitClient() *mockGitClient {
	return &mockGitClient{
		diffResult:   make(map[diffKey]diffResponse),
		runResults:   make(map[string]runResponse),
		gitInstalled: true,
	}
}

// withDiff sets up a mock response for a specific diff call.
func (m *mockGitClient) withDiff(baseRef, headRef, repo, diff string, err error) *mockGitClient {
	m.diffResult[diffKey{baseRef, headRef, repo}] = diffResponse{diff, err}
	return m
}

// withRun sets up a mock response for a specific git command.
// argKey is the joined args (e.g., "rev-parse --abbrev-ref HEAD").
func (m *mockGitClient) withRun(argKey string, output string, err error) *mockGitClient {
	m.runResults[argKey] = runResponse{output, err}
	return m
}

// withStatus sets up a mock response for Status().
func (m *mockGitClient) withStatus(status string, err error) *mockGitClient {
	m.statusResult = status
	m.statusErr = err
	return m
}

func (m *mockGitClient) Diff(_ context.Context, baseRef, headRef, repoPath string) (string, error) {
	key := diffKey{baseRef, headRef, repoPath}
	if resp, ok := m.diffResult[key]; ok {
		return resp.diff, resp.err
	}
	// Fallback: try without repo path
	key2 := diffKey{baseRef, headRef, ""}
	if resp, ok := m.diffResult[key2]; ok {
		return resp.diff, resp.err
	}
	return "", fmt.Errorf("unexpected Diff(%q, %q, %q)", baseRef, headRef, repoPath)
}

func (m *mockGitClient) Status(_ context.Context, _ string) (string, error) {
	return m.statusResult, m.statusErr
}

func (m *mockGitClient) Run(_ context.Context, _ string, args ...string) (string, error) {
	argKey := strings.Join(args, " ")
	if resp, ok := m.runResults[argKey]; ok {
		return resp.output, resp.err
	}
	return "", fmt.Errorf("unexpected Run(%v)", args)
}

func (m *mockGitClient) CheckGitInstalled() bool {
	return m.gitInstalled
}
