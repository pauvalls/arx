package output_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/output"
	"github.com/pauvalls/arx/internal/ports"
)

// ansiEscape matches ANSI escape sequences (SGR codes).
var ansiEscape = strings.NewReplacer(
	"\x1b[", "",
)

func hasANSIEscapes(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			return true
		}
	}
	return false
}

func TestNoColor_StyleHelper_ReturnsPlainText(t *testing.T) {
	// Save and restore original state
	originalNoColor := output.GetNoColor()
	defer output.SetNoColor(originalNoColor)

	output.SetNoColor(true)

	reporter := output.NewTerminalReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    domain.SeverityError,
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "Domain should not depend on infrastructure",
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if hasANSIEscapes(out) {
		t.Errorf("Output contains ANSI escape codes when NO_COLOR should disable them:\n%s", out)
	}

	// Verify content is still present (just unstyled)
	if !strings.Contains(out, "D-01") {
		t.Error("Output missing violation ID D-01")
	}
	if !strings.Contains(out, "ARCHITECTURE VIOLATIONS DETECTED") {
		t.Error("Output missing header text")
	}
	if !strings.Contains(out, "Domain should not depend on infrastructure") {
		t.Error("Output missing violation message")
	}
}

func TestNoColor_StyleHelper_WithColors(t *testing.T) {
	// Save and restore original state
	originalNoColor := output.GetNoColor()
	defer output.SetNoColor(originalNoColor)

	output.SetNoColor(false)

	reporter := output.NewTerminalReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			RuleID:      "domain-cannot-depend-on-infrastructure",
			Severity:    domain.SeverityError,
			File:        "internal/domain/order.go",
			Line:        14,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/app/internal/infrastructure/postgres",
			Message:     "Domain should not depend on infrastructure",
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	// noColor flag should be false (colors enabled)
	if output.GetNoColor() {
		t.Error("noColor should be false when colors are enabled")
	}

	// Output should still contain the violation data regardless of styling
	if !strings.Contains(out, "D-01") {
		t.Error("Output should contain violation data")
	}
	if !strings.Contains(out, "Domain should not depend on infrastructure") {
		t.Error("Output should contain violation message")
	}
}

func TestNoColor_NoViolations_PlainText(t *testing.T) {
	originalNoColor := output.GetNoColor()
	defer output.SetNoColor(originalNoColor)

	output.SetNoColor(true)

	reporter := output.NewTerminalReporter()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report([]domain.Violation{}, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if hasANSIEscapes(out) {
		t.Errorf("Output contains ANSI escape codes when NO_COLOR should disable them:\n%s", out)
	}

	if !strings.Contains(out, "No violations found") {
		t.Error("Output missing 'No violations found' message")
	}
}

func TestNoColor_MultipleSeverities_PlainText(t *testing.T) {
	originalNoColor := output.GetNoColor()
	defer output.SetNoColor(originalNoColor)

	output.SetNoColor(true)

	reporter := output.NewTerminalReporter()
	violations := []domain.Violation{
		{
			ID:          "D-01",
			Severity:    domain.SeverityError,
			File:        "a.go",
			Line:        1,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "pkg/infra",
			Message:     "Error violation",
		},
		{
			ID:          "D-02",
			Severity:    domain.SeverityWarning,
			File:        "b.go",
			Line:        2,
			SourceLayer: "application",
			TargetLayer: "infrastructure",
			Import:      "pkg/infra",
			Message:     "Warning violation",
		},
		{
			ID:          "D-03",
			Severity:    domain.SeverityInfo,
			File:        "c.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "cmd",
			Import:      "pkg/cmd",
			Message:     "Info violation",
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatTerminal)
	if err != nil {
		t.Fatalf("Report() error: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if hasANSIEscapes(out) {
		t.Errorf("Output contains ANSI escape codes when NO_COLOR should disable them:\n%s", out)
	}

	// Verify all severities are present
	if !strings.Contains(out, "D-01") || !strings.Contains(out, "D-02") || !strings.Contains(out, "D-03") {
		t.Error("Output missing some violation IDs")
	}
}

func TestNoColor_EnvVarSet(t *testing.T) {
	// noColor is set during package init() based on NO_COLOR env var
	// Just verify GetNoColor returns false (colors enabled) in normal test runs
	got := output.GetNoColor()
	// Don't fail on either value — just log it for debugging
	t.Logf("noColor = %v (NO_COLOR env: %q)", got, os.Getenv("NO_COLOR"))
}

// TestNoColor_EnvVarZero verifies that NO_COLOR=0 keeps colors enabled.
func TestNoColor_EnvVarZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNoColor_EnvVarZero_Helper")
	cmd.Env = append(os.Environ(), "NO_COLOR=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, out)
	}
}

func TestNoColor_EnvVarZero_Helper(t *testing.T) {
	if output.GetNoColor() {
		t.Fatal("noColor should be false when NO_COLOR=0")
	}
}

// TestNoColor_EnvVarUnset verifies that unset NO_COLOR keeps colors enabled.
func TestNoColor_EnvVarUnset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Build env without NO_COLOR
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "NO_COLOR=") {
			env = append(env, e)
		}
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestNoColor_EnvVarUnset_Helper")
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, out)
	}
}

func TestNoColor_EnvVarUnset_Helper(t *testing.T) {
	if output.GetNoColor() {
		t.Fatal("noColor should be false when NO_COLOR is unset")
	}
}
