package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	payload := []byte(`{"action":"opened","number":1}`)
	secret := "my-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			signature: expectedSig,
			secret:    secret,
			want:      true,
		},
		{
			name:      "invalid signature",
			payload:   payload,
			signature: "sha256=invalid",
			secret:    secret,
			want:      false,
		},
		{
			name:      "wrong secret",
			payload:   payload,
			signature: expectedSig,
			secret:    "wrong-secret",
			want:      false,
		},
		{
			name:      "empty signature",
			payload:   payload,
			signature: "",
			secret:    secret,
			want:      false,
		},
		{
			name:      "tampered payload",
			payload:   []byte(`{"action":"opened","number":2}`),
			signature: expectedSig,
			secret:    secret,
			want:      false,
		},
		{
			name:      "missing sha256 prefix",
			payload:   payload,
			signature: hex.EncodeToString(mac.Sum(nil)),
			secret:    secret,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyWebhookSignature(tt.payload, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("VerifyWebhookSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWebhookEvent_PR(t *testing.T) {
	body := map[string]any{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]any{
			"head": map[string]any{
				"sha": "headsha123",
				"ref": "feature/test",
			},
			"base": map[string]any{
				"sha": "basesha456",
				"ref": "main",
			},
		},
		"repository": map[string]any{
			"owner": map[string]any{
				"login": "test-owner",
			},
			"name": "test-repo",
		},
		"installation": map[string]any{
			"id": float64(789),
		},
	}
	payload, _ := json.Marshal(body)

	event, err := ParseWebhookEvent(payload, "pull_request")
	if err != nil {
		t.Fatalf("ParseWebhookEvent() unexpected error: %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Action != "opened" {
		t.Errorf("Action = %q, want %q", event.Action, "opened")
	}
	if event.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", event.PRNumber)
	}
	if event.BaseSHA != "basesha456" {
		t.Errorf("BaseSHA = %q, want %q", event.BaseSHA, "basesha456")
	}
	if event.HeadSHA != "headsha123" {
		t.Errorf("HeadSHA = %q, want %q", event.HeadSHA, "headsha123")
	}
	if event.RepoOwner != "test-owner" {
		t.Errorf("RepoOwner = %q, want %q", event.RepoOwner, "test-owner")
	}
	if event.RepoName != "test-repo" {
		t.Errorf("RepoName = %q, want %q", event.RepoName, "test-repo")
	}
	if event.InstallationID != 789 {
		t.Errorf("InstallationID = %d, want 789", event.InstallationID)
	}
	if event.BaseRef != "main" {
		t.Errorf("BaseRef = %q, want %q", event.BaseRef, "main")
	}
	if event.HeadRef != "feature/test" {
		t.Errorf("HeadRef = %q, want %q", event.HeadRef, "feature/test")
	}
}

func TestParseWebhookEvent_WrongEventType(t *testing.T) {
	body := []byte(`{"action":"created"}`)
	_, err := ParseWebhookEvent(body, "issues")
	if err == nil {
		t.Fatal("expected error for non-pull_request event")
	}
	if !strings.Contains(err.Error(), "pull_request") {
		t.Errorf("error should mention pull_request: %v", err)
	}
}

func TestParseWebhookEvent_InvalidJSON(t *testing.T) {
	_, err := ParseWebhookEvent([]byte("{invalid}"), "pull_request")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseWebhookEvent_MissingFields(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	_, err := ParseWebhookEvent(body, "pull_request")
	if err == nil {
		t.Fatal("expected error for missing pull_request field")
	}
}

func TestHandleWebhook(t *testing.T) {
	// Create a valid payload with HMAC signature
	secret := "test-secret"
	body := map[string]any{
		"action": "opened",
		"number": 1,
		"pull_request": map[string]any{
			"head": map[string]any{
				"sha": "h1",
				"ref": "feature",
			},
			"base": map[string]any{
				"sha": "b1",
				"ref": "main",
			},
		},
		"repository": map[string]any{
			"owner": map[string]any{"login": "owner"},
			"name":  "repo",
		},
		"installation": map[string]any{
			"id": float64(1),
		},
	}
	payload, _ := json.Marshal(body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	handler := HandleWebhook(secret, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/github-webhook", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "pull_request")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

func TestHandleWebhook_BadSignature(t *testing.T) {
	handler := HandleWebhook("secret", nil)
	req := httptest.NewRequest(http.MethodPost, "/api/github-webhook", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	req.Header.Set("X-GitHub-Event", "pull_request")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleWebhook_WrongMethod(t *testing.T) {
	handler := HandleWebhook("secret", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/github-webhook", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWebhook_NonPROpened(t *testing.T) {
	secret := "test-secret"
	body := map[string]any{
		"action": "closed",
		"number": 1,
		"pull_request": map[string]any{
			"head": map[string]any{"sha": "h1", "ref": "f"},
			"base": map[string]any{"sha": "b1", "ref": "m"},
		},
		"repository": map[string]any{
			"owner": map[string]any{"login": "o"},
			"name":  "r",
		},
		"installation": map[string]any{"id": float64(1)},
	}
	payload, _ := json.Marshal(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	handler := HandleWebhook(secret, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/github-webhook", bytes.NewReader(payload))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "pull_request")
	rec := httptest.NewRecorder()

	handler(rec, req)
	// closed action is not checked — we accept all PR events for simplicity
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}
