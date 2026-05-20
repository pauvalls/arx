package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test-token", "https://api.github.com")
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestClient_CreateCheckRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/repos/owner/repo/check-runs"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["name"] != "arx" {
			t.Errorf("name = %v, want arx", body["name"])
		}
		if body["head_sha"] != "headsha" {
			t.Errorf("head_sha = %v, want headsha", body["head_sha"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 99}`))
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	output := domain.CheckRunOutput{
		Title:   "Arx PR Check",
		Summary: "No violations found",
	}
	id, err := c.CreateCheckRun("owner", "repo", domain.PRInfo{
		HeadSHA: "headsha",
	}, output)
	if err != nil {
		t.Fatalf("CreateCheckRun() unexpected error: %v", err)
	}
	if id != 99 {
		t.Errorf("id = %d, want 99", id)
	}
}

func TestClient_UpdateCheckRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		expectedPath := "/repos/owner/repo/check-runs/42"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["conclusion"] != "success" {
			t.Errorf("conclusion = %v, want success", body["conclusion"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	output := domain.CheckRunOutput{
		Title:   "Arx PR Check",
		Summary: "All clear",
	}
	err := c.UpdateCheckRun("owner", "repo", 42, domain.CheckRunSuccess, output)
	if err != nil {
		t.Fatalf("UpdateCheckRun() unexpected error: %v", err)
	}
}

func TestClient_GetPRDiff(t *testing.T) {
	expectedDiff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 line1
-line2
+new_line2
 line3`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/compare/base...head" {
			t.Errorf("expected compare path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedDiff))
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	diff, err := c.GetPRDiff("owner", "repo", "base", "head")
	if err != nil {
		t.Fatalf("GetPRDiff() unexpected error: %v", err)
	}
	if diff != expectedDiff {
		t.Errorf("diff = %q, want %q", diff, expectedDiff)
	}
}

func TestClient_ApprovePR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/repos/owner/repo/pulls/42/reviews"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["event"] != "APPROVE" {
			t.Errorf("event = %v, want APPROVE", body["event"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	err := c.ApprovePR("owner", "repo", 42)
	if err != nil {
		t.Fatalf("ApprovePR() unexpected error: %v", err)
	}
}

func TestClient_GetPRDiff_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	_, err := c.GetPRDiff("owner", "repo", "base", "head")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_CreateCheckRun_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	c := NewClient("test-token", server.URL)
	_, err := c.CreateCheckRun("owner", "repo", domain.PRInfo{HeadSHA: "h"}, domain.CheckRunOutput{})
	if err == nil {
		t.Fatal("expected error for 422")
	}
}

func TestClient_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	c := NewClient("", server.URL)
	_, err := c.CreateCheckRun("owner", "repo", domain.PRInfo{HeadSHA: "h"}, domain.CheckRunOutput{})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestClient_BaseURL(t *testing.T) {
	c := NewClient("token", "https://api.github.com")
	if c.baseURL != "https://api.github.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.github.com")
	}
}
