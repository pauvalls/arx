package github

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// jwtHeader is the JWT header for RS256.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtPayload is the JWT payload for GitHub App authentication.
type jwtPayload struct {
	IAT int64  `json:"iat"`
	EXP int64  `json:"exp"`
	ISS string `json:"iss"`
}

// installationTokenResponse is the response from the GitHub API for installation tokens.
type installationTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// base64URLEncode encodes data as base64url without padding.
func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// GenerateJWT creates a JWT signed with RS256 using the GitHub App's private key.
func GenerateJWT(appID int64, privateKeyPEM []byte) (string, error) {
	if len(privateKeyPEM) == 0 {
		return "", fmt.Errorf("private key is empty")
	}

	// Parse PEM block
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block from private key")
	}

	// Parse RSA private key
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		key8, err8 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err8 != nil {
			return "", fmt.Errorf("failed to parse private key (tried PKCS1: %w, PKCS8: %w)", err, err8)
		}
		var ok bool
		key, ok = key8.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("private key is not RSA")
		}
	}

	// Create JWT header
	header := jwtHeader{
		Alg: "RS256",
		Typ: "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT header: %w", err)
	}

	// Create JWT payload
	now := time.Now().Unix()
	payload := jwtPayload{
		IAT: now,
		EXP: now + 600, // 10 minutes
		ISS: fmt.Sprintf("%d", appID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT payload: %w", err)
	}

	// Create signing input
	headerB64 := base64URLEncode(headerJSON)
	payloadB64 := base64URLEncode(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	// Hash the signing input
	hasher := sha256.New()
	hasher.Write([]byte(signingInput))
	hash := hasher.Sum(nil)

	// Sign with RSA PKCS1v15
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash)
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}

	// Base64url encode signature
	sigB64 := base64URLEncode(signature)

	return signingInput + "." + sigB64, nil
}

// GetInstallationToken exchanges a JWT for a GitHub App installation access token.
func GetInstallationToken(jwt string, installationID int64, apiBaseURL string) (string, error) {
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", apiBaseURL, installationID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp installationTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	return tokenResp.Token, nil
}
