package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx-config.yaml")
	content := []byte(`version: "1.0"
github_app:
  app_id: 123456
  app_slug: arx-bot
  installation_id: 789012
  private_key_path: /tmp/key.pem
webhook:
  secret: "my-secret"
pr_check:
  auto_approve: true
  auto_approve_on: success
  summary_format: markdown
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg.GitHubApp.AppID != 123456 {
		t.Errorf("AppID = %d, want 123456", cfg.GitHubApp.AppID)
	}
	if cfg.GitHubApp.AppSlug != "arx-bot" {
		t.Errorf("AppSlug = %q, want %q", cfg.GitHubApp.AppSlug, "arx-bot")
	}
	if cfg.GitHubApp.InstallationID != 789012 {
		t.Errorf("InstallationID = %d, want 789012", cfg.GitHubApp.InstallationID)
	}
	if cfg.GitHubApp.WebhookSecret != "my-secret" {
		t.Errorf("WebhookSecret = %q, want %q", cfg.GitHubApp.WebhookSecret, "my-secret")
	}
	if !cfg.PRCheck.AutoApprove {
		t.Errorf("AutoApprove = false, want true")
	}
	if cfg.PRCheck.AutoApproveOn != "success" {
		t.Errorf("AutoApproveOn = %q, want %q", cfg.PRCheck.AutoApproveOn, "success")
	}
	if cfg.PRCheck.SummaryFormat != "markdown" {
		t.Errorf("SummaryFormat = %q, want %q", cfg.PRCheck.SummaryFormat, "markdown")
	}
	// PrivateKey should not be serialized/set yet (loaded separately)
	if len(cfg.GitHubApp.PrivateKey) != 0 {
		t.Errorf("PrivateKey should be empty after LoadConfig, got %d bytes", len(cfg.GitHubApp.PrivateKey))
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadConfig_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.pem")
	// Write a valid test private key (PKCS#1 format, RSA 2048)
	keyContent := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0gDk3DPZ0fC7YNA7j4hB0FQ7H0L9a6Y9h0X3e5a8nF0W
3eV8mRgH6Hj0L9a6Y9h0X3e5a8nF0W3eV8mRgH6Hj0L9a6Y9h0X3e5a8nF0W
-----END RSA PRIVATE KEY-----`)
	if err := os.WriteFile(keyPath, keyContent, 0600); err != nil {
		t.Fatal(err)
	}

	keyData, err := LoadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() unexpected error: %v", err)
	}
	if len(keyData) == 0 {
		t.Error("expected non-empty key data")
	}
}

func TestLoadPrivateKey_Missing(t *testing.T) {
	_, err := LoadPrivateKey("/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestConfig_LoadFull(t *testing.T) {
	// Test the full config loading with private key
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(keyPath, []byte("fake-key-data\n"), 0600); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(tmpDir, "arx-config.yaml")
	content := []byte(`version: "1.0"
github_app:
  app_id: 42
  installation_id: 99
  private_key_path: ` + keyPath + `
webhook:
  secret: "s3cr3t"
pr_check:
  auto_approve: true
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFullConfig(configPath)
	if err != nil {
		t.Fatalf("LoadFullConfig() unexpected error: %v", err)
	}

	if cfg.GitHubApp.AppID != 42 {
		t.Errorf("AppID = %d, want 42", cfg.GitHubApp.AppID)
	}
	if cfg.GitHubApp.InstallationID != 99 {
		t.Errorf("InstallationID = %d, want 99", cfg.GitHubApp.InstallationID)
	}
	if string(cfg.GitHubApp.PrivateKey) != "fake-key-data\n" {
		t.Errorf("PrivateKey = %q, want %q", string(cfg.GitHubApp.PrivateKey), "fake-key-data\n")
	}
	if cfg.GitHubApp.WebhookSecret != "s3cr3t" {
		t.Errorf("WebhookSecret = %q, want %q", cfg.GitHubApp.WebhookSecret, "s3cr3t")
	}
}

func TestAutoApproveEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: false,
		},
		{
			name: "auto_approve true",
			cfg: &Config{
				PRCheck: PRCheckConfig{
					AutoApprove: true,
				},
			},
			want: true,
		},
		{
			name: "auto_approve false",
			cfg: &Config{
				PRCheck: PRCheckConfig{
					AutoApprove: false,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoApproveEnabled(tt.cfg)
			if got != tt.want {
				t.Errorf("AutoApproveEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
