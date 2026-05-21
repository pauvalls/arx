package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// Compile-time check that Client implements ports.GitHubClient.
var _ ports.GitHubClient = (*Client)(nil)

// Client is an HTTP client for the GitHub REST API.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{
		token:   token,
		baseURL: baseURL,
		http:    http.DefaultClient,
	}
}

// doRequest performs an HTTP request to the GitHub API.
func (c *Client) doRequest(method, url string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// apiURL builds a full GitHub API URL from a path.
func (c *Client) apiURL(path string) string {
	return c.baseURL + path
}

// checkRunRequest is the request body for creating/updating check runs.
type checkRunRequest struct {
	Name        string                `json:"name"`
	HeadSHA     string                `json:"head_sha,omitempty"`
	Status      string                `json:"status,omitempty"`
	Conclusion  string                `json:"conclusion,omitempty"`
	Output      *domain.CheckRunOutput `json:"output,omitempty"`
}

// CreateCheckRun creates a check run on a GitHub commit.
func (c *Client) CreateCheckRun(owner, repo string, pr domain.PRInfo, output domain.CheckRunOutput) (int64, error) {
	url := c.apiURL(fmt.Sprintf("/repos/%s/%s/check-runs", owner, repo))

	body := checkRunRequest{
		Name:    "arx",
		HeadSHA: pr.HeadSHA,
		Status:  "completed",
		Output:  &output,
	}

	// Use "in_progress" status if no conclusion yet
	if output.Title != "" || output.Summary != "" {
		body.Conclusion = string(domain.CheckRunSuccess)
		// If there are annotations with failure level, set conclusion to failure
		for _, a := range output.Annotations {
			if a.AnnotationLevel == "failure" {
				body.Conclusion = string(domain.CheckRunFailure)
				break
			}
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("marshaling request: %w", err)
	}

	respBody, _, err := c.doRequest(http.MethodPost, url, jsonBody)
	if err != nil {
		return 0, err
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("parsing response: %w", err)
	}

	return result.ID, nil
}

// UpdateCheckRun updates an existing check run with a conclusion.
func (c *Client) UpdateCheckRun(owner, repo string, checkRunID int64, conclusion domain.CheckRunConclusion, output domain.CheckRunOutput) error {
	url := c.apiURL(fmt.Sprintf("/repos/%s/%s/check-runs/%d", owner, repo, checkRunID))

	body := checkRunRequest{
		Status:     "completed",
		Conclusion: string(conclusion),
		Output:     &output,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, _, err = c.doRequest(http.MethodPatch, url, jsonBody)
	return err
}

// GetPRDiff fetches the unified diff between two commits.
func (c *Client) GetPRDiff(owner, repo, baseSHA, headSHA string) (string, error) {
	url := c.apiURL(fmt.Sprintf("/repos/%s/%s/compare/%s...%s", owner, repo, baseSHA, headSHA))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

// ApprovePR approves a pull request via the GitHub API.
func (c *Client) ApprovePR(owner, repo string, prNumber int) error {
	url := c.apiURL(fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, prNumber))

	body := map[string]string{
		"event": "APPROVE",
		"body":  "✅ All architecture checks passed — auto-approved by arx.",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, _, err = c.doRequest(http.MethodPost, url, jsonBody)
	return err
}
