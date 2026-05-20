package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// generateTestKey creates an RSA key pair for testing.
func generateTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	}
	pemData := pem.EncodeToMemory(pemBlock)
	return key, pemData
}

func TestGenerateJWT(t *testing.T) {
	_, pemData := generateTestKey(t)

	token, err := GenerateJWT(12345, pemData)
	if err != nil {
		t.Fatalf("GenerateJWT() unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateJWT() returned empty token")
	}

	// Verify JWT structure: three base64 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	// Verify each part is non-empty
	for i, part := range parts {
		if part == "" {
			t.Errorf("JWT part %d is empty", i)
		}
	}
}

func TestGenerateJWT_InvalidKey(t *testing.T) {
	_, err := GenerateJWT(12345, []byte("not-a-valid-key"))
	if err == nil {
		t.Fatal("expected error for invalid private key")
	}
}

func TestGenerateJWT_EmptyKey(t *testing.T) {
	_, err := GenerateJWT(12345, nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}

func TestGetInstallationToken(t *testing.T) {
	// Start a mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/app/installations/42/access_tokens"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token, got %s", auth)
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("expected GitHub API accept header")
		}

		// Return a token
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"token":"ghs_test123","expires_at":"2025-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	token, err := GetInstallationToken("test.jwt.token", 42, server.URL)
	if err != nil {
		t.Fatalf("GetInstallationToken() unexpected error: %v", err)
	}
	if token != "ghs_test123" {
		t.Errorf("token = %q, want %q", token, "ghs_test123")
	}
}

func TestGetInstallationToken_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer server.Close()

	_, err := GetInstallationToken("test.jwt.token", 42, server.URL)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestGetInstallationToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	_, err := GetInstallationToken("test.jwt.token", 42, server.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
