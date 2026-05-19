package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

func TestBaselineCmd_DiffFlag(t *testing.T) {
	flag := baselineCmd.Flags().Lookup("diff")
	if flag == nil {
		t.Fatal("--diff flag not found on baseline command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--diff default = %q, want false", flag.DefValue)
	}
}

func TestBaselineCmd_HistoryFlag(t *testing.T) {
	flag := baselineCmd.Flags().Lookup("history")
	if flag == nil {
		t.Fatal("--history flag not found on baseline command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--history default = %q, want false", flag.DefValue)
	}
}

func TestBaselineCmd_RefreshThresholdFlag(t *testing.T) {
	flag := baselineCmd.Flags().Lookup("refresh-threshold")
	if flag == nil {
		t.Fatal("--refresh-threshold flag not found on baseline command")
	}
	if flag.DefValue != "3" {
		t.Errorf("--refresh-threshold default = %q, want 3", flag.DefValue)
	}
}

// captureStdout runs fn and returns captured stdout as string.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func TestBaselineRenderer_DiffTable(t *testing.T) {
	t.Run("shows diff with added and resolved", func(t *testing.T) {
		added := []domain.Violation{
			{ID: "D-01", File: "internal/domain/service.go", SourceLayer: "domain", TargetLayer: "infrastructure"},
			{ID: "D-02", File: "internal/domain/repo.go", SourceLayer: "domain", TargetLayer: "infrastructure"},
		}
		resolved := []domain.Violation{
			{ID: "C-01", SourceLayer: "domain", TargetLayer: "infrastructure", Message: "circular dependency resolved"},
		}

		output := captureStdout(func() {
			renderDiffOutput("2026-05-19 12:00:00", added, resolved)
		})

		if !blContains(output, "BASELINE DIFF") {
			t.Error("output should contain BASELINE DIFF header")
		}
		if !blContains(output, "Added:    2 violations") {
			t.Error("output should show 2 added violations")
		}
		if !blContains(output, "Resolved: 1 violation") {
			t.Error("output should show 1 resolved violation")
		}
		if !blContains(output, "D-01") {
			t.Error("output should include D-01")
		}
		if !blContains(output, "C-01") {
			t.Error("output should include C-01")
		}
	})

	t.Run("no changes when empty", func(t *testing.T) {
		output := captureStdout(func() {
			renderDiffOutput("2026-05-19 12:00:00", nil, nil)
		})

		if !blContains(output, "No changes since last snapshot") {
			t.Error("output should show no changes message")
		}
	})
}

func TestBaselineRenderer_HistoryTable(t *testing.T) {
	t.Run("shows history table with data", func(t *testing.T) {
		trend := []domain.TrendPoint{
			{Date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Total: 12, Errors: 5, Warnings: 6, Info: 1},
			{Date: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC), Total: 8, Errors: 3, Warnings: 4, Info: 1},
			{Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Total: 5, Errors: 2, Warnings: 2, Info: 1},
		}

		output := captureStdout(func() {
			renderHistoryOutput(trend)
		})

		if !blContains(output, "BASELINE HISTORY") {
			t.Error("output should contain header")
		}
		if !blContains(output, "2026-05-01") {
			t.Error("output should include date")
		}
		if !blContains(output, "2026-05-10") {
			t.Error("output should include latest date")
		}
	})

	t.Run("shows empty message when no history", func(t *testing.T) {
		output := captureStdout(func() {
			renderHistoryOutput(nil)
		})

		t.Logf("Captured output: %q", output)
		if !blContains(output, "No baseline history") {
			t.Errorf("output should show no history message, got: %q", output)
		}
	})
}

// blContains is a helper to check substring in string.
func blContains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
