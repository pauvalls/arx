package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/pauvalls/arx/internal/infrastructure/detector"
	"github.com/pauvalls/arx/internal/infrastructure/history"
)

func TestAuditCommand_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary project with arx.yaml
	tmpDir := t.TempDir()

	// Create arx.yaml
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - com/example/domain/**
  - name: application
    paths:
      - com/example/app/**
  - name: infrastructure
    paths:
      - com/example/infrastructure/**
rules:
  - id: domain-no-import-application
    from: domain
    to:
      - application
    type: Cannot
    severity: error
    explanation: Domain must not depend on application
`
	os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(arxConfig), 0644)

	// Create Java source files
	os.MkdirAll(filepath.Join(tmpDir, "src/main/java/com/example/domain"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src/main/java/com/example/app"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src/main/java/com/example/infrastructure"), 0755)

	// Domain file (clean - no violations)
	os.WriteFile(filepath.Join(tmpDir, "src/main/java/com/example/domain/Order.java"), []byte(`package com.example.domain;
import java.util.List;

public class Order {
    private List<String> items;
}`), 0644)

	// Application file importing domain (allowed)
	os.WriteFile(filepath.Join(tmpDir, "src/main/java/com/example/app/OrderService.java"), []byte(`package com.example.app;
import com.example.domain.Order;

public class OrderService {
    public Order createOrder() {
        return new Order();
    }
}`), 0644)

	// Infrastructure file importing domain (allowed)
	os.WriteFile(filepath.Join(tmpDir, "src/main/java/com/example/infrastructure/OrderRepository.java"), []byte(`package com.example.infrastructure;
import com.example.domain.Order;

public class OrderRepository {
    public void save(Order order) {}
}`), 0644)

	// Create service
	configReader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	historyStorage := history.NewFileSystemHistory(filepath.Join(tmpDir, ".arx-history"))

	service := application.NewAuditService(configReader, detectors, historyStorage, filepath.Join(tmpDir, ".arx-history"))

	// Run audit
	ctx := context.Background()
	report, err := service.Audit(ctx, tmpDir, filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	// Verify report structure
	if report.ProjectRoot != tmpDir {
		t.Errorf("Expected project root %q, got %q", tmpDir, report.ProjectRoot)
	}

	// Should have timestamp
	if report.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Should have coupling matrix
	if report.CouplingMatrix.FromTo == nil {
		t.Error("Expected coupling matrix to be initialized")
	}

	// Should have debt score
	if report.DebtScore.BySeverity == nil {
		t.Error("Expected debt score severity map to be initialized")
	}

	t.Logf("Audit completed: %d violations, debt score: %d", len(report.Violations), report.DebtScore.Total)
}

func TestAuditCommand_HistoryPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	historyDir := filepath.Join(tmpDir, ".arx-history")

	// Create minimal arx.yaml
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
rules: []
`
	os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(arxConfig), 0644)

	// Create service
	configReader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	historyStorage := history.NewFileSystemHistory(historyDir)

	service := application.NewAuditService(configReader, detectors, historyStorage, historyDir)

	// Run audit
	ctx := context.Background()
	report, err := service.Audit(ctx, tmpDir, filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	// Verify history was saved
	files, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed to read history directory: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected audit history file to be created")
	}

	// Verify we can load the latest audit
	latest, err := historyStorage.LoadLatest(ctx)
	if err != nil {
		t.Fatalf("LoadLatest() error = %v", err)
	}

	if latest == nil {
		t.Error("Expected to load latest audit")
	}

	// Compare dates (timestamps may lose precision during JSON serialization)
	if latest.Timestamp.Format("2006-01-02") != report.Timestamp.Format("2006-01-02") {
		t.Errorf("Loaded latest audit date %q doesn't match saved report date %q",
			latest.Timestamp.Format("2006-01-02"), report.Timestamp.Format("2006-01-02"))
	}

	t.Logf("History persistence verified: %d files in history dir", len(files))
}

func TestAuditCommand_TrendComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	historyDir := filepath.Join(tmpDir, ".arx-history")

	// Create minimal arx.yaml
	arxConfig := `version: "1.0"
layers:
  - name: domain
    paths:
      - domain/**
rules: []
`
	os.WriteFile(filepath.Join(tmpDir, "arx.yaml"), []byte(arxConfig), 0644)

	// Create service
	configReader := config.NewYAMLReader()
	detectors := detector.GetDetectors()
	historyStorage := history.NewFileSystemHistory(historyDir)

	service := application.NewAuditService(configReader, detectors, historyStorage, historyDir)

	ctx := context.Background()

	// Run first audit
	report1, err := service.Audit(ctx, tmpDir, filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("First Audit() error = %v", err)
	}

	// Add a violation to the first report and save manually to simulate a previous audit
	report1.Violations = append(report1.Violations, domain.Violation{
		ID:       "v1",
		Severity: "error",
	})
	report1.DebtScore.AddViolation("error")
	historyStorage.Save(ctx, report1)

	// Run second audit (should compare with first)
	report2, err := service.Audit(ctx, tmpDir, filepath.Join(tmpDir, "arx.yaml"))
	if err != nil {
		t.Fatalf("Second Audit() error = %v", err)
	}

	// Verify trend report exists
	if report2.TrendReport.Status == "" {
		t.Error("Expected trend report to be populated")
	}

	t.Logf("Trend comparison: status=%s, violation_delta=%d, debt_delta=%d",
		report2.TrendReport.Status,
		report2.TrendReport.ViolationDelta,
		report2.TrendReport.DebtDelta)
}

func TestAuditCommand_RetentionPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	historyDir := filepath.Join(tmpDir, ".arx-history")

	// Create history storage
	historyStorage := history.NewFileSystemHistory(historyDir)

	// Create 15 audits (exceeds default max of 10)
	ctx := context.Background()
	for i := 0; i < 15; i++ {
		report := &domain.AuditReport{
			Timestamp:   time.Now().AddDate(0, 0, -i),
			ProjectRoot: tmpDir,
		}
		_, err := historyStorage.Save(ctx, report)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	// Enforce retention
	deleted, err := historyStorage.DeleteOld(ctx, 10)
	if err != nil {
		t.Fatalf("DeleteOld() error = %v", err)
	}

	if deleted != 5 {
		t.Errorf("Expected to delete 5 audits, got %d", deleted)
	}

	// Verify only 10 remain
	files, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed to read history directory: %v", err)
	}

	// Count JSON files (not symlinks)
	jsonCount := 0
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" && f.Name() != "last-audit.json" {
			jsonCount++
		}
	}

	if jsonCount > 10 {
		t.Errorf("Expected at most 10 audit files, got %d", jsonCount)
	}

	t.Logf("Retention policy verified: deleted %d audits, %d remain", deleted, jsonCount)
}
