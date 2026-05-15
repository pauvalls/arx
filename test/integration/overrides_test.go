package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/history"
)

func TestOverrides_ConfigParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create arx.yaml with overrides
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - com/example/domain/**
  - name: infrastructure
    paths:
      - com/example/infrastructure/**
rules:
  - id: no-infra-from-domain
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    overrides:
      - path: com/example/legacy/
        severity: warning
      - path: com/example/hidden/
        enabled: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(arxConfig), 0644); err != nil {
		t.Fatalf("failed to write arx.yaml: %v", err)
	}

	// Create pom.xml to trigger Java detector
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte(`<project><groupId>test</groupId><artifactId>test</artifactId></project>`), 0644); err != nil {
		t.Fatalf("failed to write pom.xml: %v", err)
	}

	// Create source files
	srcDirs := []string{
		"src/main/java/com/example/domain",
		"src/main/java/com/example/legacy",
		"src/main/java/com/example/infrastructure",
	}
	for _, dir := range srcDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	// Domain service (violation: imports infrastructure)
	domainFile := `package com.example.domain;
import com.example.infrastructure.DbService;
public class DomainService {
    private DbService db;
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "src/main/java/com/example/domain/DomainService.java"), []byte(domainFile), 0644); err != nil {
		t.Fatalf("failed to write domain file: %v", err)
	}

	// Legacy service (violation: imports infrastructure, but overridden)
	legacyFile := `package com.example.legacy;
import com.example.infrastructure.DbService;
public class LegacyService {
    private DbService db;
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "src/main/java/com/example/legacy/LegacyService.java"), []byte(legacyFile), 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Create service
	configReader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	historyStorage := history.NewFileSystemHistory(filepath.Join(tmpDir, ".arx-history"))

	service := application.NewAuditService(configReader, detectors, historyStorage, filepath.Join(tmpDir, ".arx-history"))

	ctx := context.Background()

	// Verify config is parsed correctly with overrides
	cfg, err := configReader.Read(filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("config.Read() error = %v", err)
	}

	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}

	rule := cfg.Rules[0]
	if len(rule.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(rule.Overrides))
	}

	if rule.Overrides[0].Path != "com/example/legacy/" {
		t.Errorf("override[0].path = %q, want %q", rule.Overrides[0].Path, "com/example/legacy/")
	}
	if rule.Overrides[0].Severity != domain.SeverityWarning {
		t.Errorf("override[0].severity = %q, want %q", rule.Overrides[0].Severity, domain.SeverityWarning)
	}

	if rule.Overrides[1].Path != "com/example/hidden/" {
		t.Errorf("override[1].path = %q, want %q", rule.Overrides[1].Path, "com/example/hidden/")
	}
	if rule.Overrides[1].Enabled == nil || *rule.Overrides[1].Enabled {
		t.Error("override[1].enabled should be false")
	}

	// Run audit
	report, err := service.Audit(ctx, tmpDir, filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	// Verify report structure
	if report.ProjectRoot != tmpDir {
		t.Errorf("Expected project root %q, got %q", tmpDir, report.ProjectRoot)
	}
	if report.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
	if report.CouplingMatrix.FromTo == nil {
		t.Error("Expected coupling matrix to be initialized")
	}
	if report.DebtScore.BySeverity == nil {
		t.Error("Expected debt score severity map to be initialized")
	}

	t.Logf("Config with overrides parsed: rule %q has %d overrides", rule.ID, len(rule.Overrides))
	t.Logf("Audit completed: %d violations, debt score: %d", len(report.Violations), report.DebtScore.Total)
}

func TestOverrides_Validate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Test valid config with overrides
	validConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infra/**
rules:
  - id: R1
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    overrides:
      - path: legacy/
        severity: warning
      - path: hidden/
        enabled: false
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reader := config.NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("config.Read() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should not error: %v", err)
	}

	// Test invalid config with bad severity in override
	invalidConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infra/**
rules:
  - id: R1
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
    overrides:
      - path: legacy/
        severity: critical
`
	configPath2 := filepath.Join(tmpDir, "arx-invalid.yaml")
	if err := os.WriteFile(configPath2, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg2, err := reader.Read(configPath2)
	if err != nil {
		t.Fatalf("config.Read() error = %v", err)
	}

	if err := cfg2.Validate(); err == nil {
		t.Error("config with invalid override severity should error")
	}
}

func TestOverrides_BackwardCompat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Config without any overrides (backward compat)
	configYAML := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
  - name: infrastructure
    paths:
      - infra/**
rules:
  - id: R1
    from: domain
    to:
      - infrastructure
    type: Cannot
    severity: error
`
	configPath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	reader := config.NewYAMLReader()
	cfg, err := reader.Read(configPath)
	if err != nil {
		t.Fatalf("config.Read() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("config without overrides should validate: %v", err)
	}

	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}

	// Rules without Overrides should have nil (backward compat)
	if cfg.Rules[0].Overrides != nil {
		t.Error("rule without overrides should have nil Overrides slice")
	}
}
