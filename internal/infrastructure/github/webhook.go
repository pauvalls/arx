package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// WebhookEvent holds the parsed fields from a GitHub pull_request webhook event.
type WebhookEvent struct {
	Action         string
	PRNumber       int
	BaseSHA        string
	HeadSHA        string
	BaseRef        string
	HeadRef        string
	RepoOwner      string
	RepoName       string
	InstallationID int64
}

// PRCheckRunner is the interface for running PR checks (avoids import cycle).
type PRCheckRunner interface {
	RunPRCheck(owner, repo string, event WebhookEvent) error
}

// VerifyWebhookSignature checks the HMAC-SHA256 signature of a webhook payload.
func VerifyWebhookSignature(payload []byte, signature string, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	sigHex := strings.TrimPrefix(signature, "sha256=")
	expectedMAC, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	actualMAC := mac.Sum(nil)

	return hmac.Equal(actualMAC, expectedMAC)
}

// ParseWebhookEvent parses a GitHub pull_request webhook payload.
func ParseWebhookEvent(body []byte, eventType string) (*WebhookEvent, error) {
	if eventType != "pull_request" {
		return nil, fmt.Errorf("unsupported event type: %s (expected pull_request)", eventType)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing webhook payload: %w", err)
	}

	action, _ := raw["action"].(string)
	prNumber, _ := raw["number"].(float64)

	prRaw, ok := raw["pull_request"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing pull_request field in webhook payload")
	}

	// Extract head
	headRaw, ok := prRaw["head"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing pull_request.head field")
	}
	headSHA, _ := headRaw["sha"].(string)
	headRef, _ := headRaw["ref"].(string)

	// Extract base
	baseRaw, ok := prRaw["base"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing pull_request.base field")
	}
	baseSHA, _ := baseRaw["sha"].(string)
	baseRef, _ := baseRaw["ref"].(string)

	// Extract repository
	repoRaw, ok := raw["repository"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing repository field")
	}
	repoName, _ := repoRaw["name"].(string)

	ownerRaw, ok := repoRaw["owner"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing repository.owner field")
	}
	repoOwner, _ := ownerRaw["login"].(string)

	// Extract installation
	var installationID int64
	installRaw, ok := raw["installation"].(map[string]any)
	if ok {
		if id, ok := installRaw["id"].(float64); ok {
			installationID = int64(id)
		}
	}

	return &WebhookEvent{
		Action:         action,
		PRNumber:       int(prNumber),
		BaseSHA:        baseSHA,
		HeadSHA:        headSHA,
		BaseRef:        baseRef,
		HeadRef:        headRef,
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		InstallationID: installationID,
	}, nil
}

// HandleWebbook creates an HTTP handler for GitHub webhooks.
func HandleWebhook(secret string, runner PRCheckRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Verify signature
		signature := r.Header.Get("X-Hub-Signature-256")
		if !VerifyWebhookSignature(body, signature, secret) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		eventType := r.Header.Get("X-GitHub-Event")

		// Parse event
		event, err := ParseWebhookEvent(body, eventType)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse event: %v", err), http.StatusBadRequest)
			return
		}

		// Run PR check asynchronously if we have a runner
		if runner != nil {
			go func() {
				_ = runner.RunPRCheck(event.RepoOwner, event.RepoName, *event) //nolint:errcheck
			}()
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	}
}
