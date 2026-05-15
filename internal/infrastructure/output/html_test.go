package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	"golang.org/x/net/html"
)

func TestHTMLReporter_ValidHTML5(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID:          "viol-001",
			RuleID:      "no-domain-to-infra",
			File:        "domain/service.go",
			Line:        10,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/infra/db",
			Message:     "Domain layer should not depend on infrastructure",
			Severity:    domain.SeverityError,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse HTML to validate structure
	doc, err := html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	// Verify document structure
	if doc.Type != html.DocumentNode {
		t.Error("Expected document node")
	}

	// Check for DOCTYPE
	hasDoctype := false
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.DoctypeNode {
			hasDoctype = true
			break
		}
	}
	if !hasDoctype {
		t.Error("Missing DOCTYPE declaration")
	}

	// Check for html, head, and body elements
	hasHTML := false
	hasHead := false
	hasBody := false

	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				hasHTML = true
			case "head":
				hasHead = true
			case "body":
				hasBody = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkNode(c)
		}
	}
	checkNode(doc)

	if !hasHTML {
		t.Error("Missing <html> element")
	}
	if !hasHead {
		t.Error("Missing <head> element")
	}
	if !hasBody {
		t.Error("Missing <body> element")
	}
}

func TestHTMLReporter_EmptyReport(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain "No violations found"
	if !strings.Contains(output, "No violations found") {
		t.Error("Empty report should indicate no violations found")
	}

	// Should still be valid HTML
	_, err = html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Empty report HTML is invalid: %v", err)
	}

	// Should contain CSS styles
	if !strings.Contains(output, ":root") {
		t.Error("Empty report should include CSS styles")
	}
}

func TestHTMLReporter_FullReport(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID:          "viol-001",
			RuleID:      "no-domain-to-infra",
			File:        "domain/service.go",
			Line:        10,
			SourceLayer: "domain",
			TargetLayer: "infrastructure",
			Import:      "github.com/example/infra/db",
			Message:     "Domain layer should not depend on infrastructure",
			Severity:    domain.SeverityError,
		},
		{
			ID:          "viol-002",
			RuleID:      "no-app-to-presentation",
			File:        "application/service.go",
			Line:        25,
			SourceLayer: "application",
			TargetLayer: "presentation",
			Import:      "github.com/example/web/handler",
			Message:     "Application layer should not depend on presentation",
			Severity:    domain.SeverityWarning,
		},
		{
			ID:          "viol-003",
			RuleID:      "suggestion-modularize",
			File:        "main.go",
			Line:        5,
			SourceLayer: "cmd",
			TargetLayer: "domain",
			Import:      "github.com/example/domain/model",
			Message:     "Consider extracting this to a separate package",
			Severity:    domain.SeverityInfo,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Validate HTML structure
	_, err = html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Full report HTML is invalid: %v", err)
	}

	// Check for all severity types
	if !strings.Contains(output, "violation error") {
		t.Error("Missing error severity class")
	}
	if !strings.Contains(output, "violation warning") {
		t.Error("Missing warning severity class")
	}
	if !strings.Contains(output, "violation info") {
		t.Error("Missing info severity class")
	}

	// Check for summary counts
	if !strings.Contains(output, "Errors") {
		t.Error("Missing errors count in summary")
	}
	if !strings.Contains(output, "Warnings") {
		t.Error("Missing warnings count in summary")
	}
	if !strings.Contains(output, "Info") {
		t.Error("Missing info count in summary")
	}

	// Check for violation details
	if !strings.Contains(output, "domain/service.go:10") {
		t.Error("Missing first violation file reference")
	}
	if !strings.Contains(output, "Domain layer should not depend on infrastructure") {
		t.Error("Missing first violation message")
	}
}

func TestHTMLReporter_SpecialCharsEscaping(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID:          "viol-xss",
			RuleID:      "xss-test",
			File:        "<script>alert('xss')</script>.go",
			Line:        1,
			SourceLayer: "layer<script>",
			TargetLayer: "layer</script>",
			Import:      "import&test\"quote",
			Message:     "Message with <script>evil()</script> and & special \"chars\"",
			Severity:    domain.SeverityError,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Validate HTML structure (should still be valid despite special chars)
	_, err = html.Parse(strings.NewReader(output))
	if err != nil {
		t.Fatalf("HTML with special chars is invalid: %v", err)
	}

	// Check that script tags are escaped (not executed)
	if strings.Contains(output, "<script>alert('xss')</script>") {
		t.Error("Script tags should be escaped in output")
	}

	// Check for HTML entities
	if !strings.Contains(output, "&lt;script&gt;") {
		t.Error("Script tags should be escaped as HTML entities")
	}
	if !strings.Contains(output, "&amp;") {
		t.Error("Ampersands should be escaped as &amp;")
	}
	if !strings.Contains(output, "&#34;") && !strings.Contains(output, "&quot;") {
		t.Error("Quotes should be escaped as HTML entities")
	}

	// Ensure no raw script execution
	if strings.Contains(output, "alert('xss')") {
		t.Error("XSS payload should not be executable")
	}
}

func TestHTMLReporter_WrongFormat(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{}

	err := reporter.Report(violations, ports.OutputFormatJSON)
	if err == nil {
		t.Error("Expected error when using wrong format")
	}
	if !strings.Contains(err.Error(), "html reporter only supports html format") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestHTMLReporter_CSSStyles(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID: "D-01", RuleID: "test-rule", File: "test.go", Line: 1,
			SourceLayer: "domain", TargetLayer: "infra", Import: "x", Message: "test",
			Severity: domain.SeverityError,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output exists
	if len(output) == 0 {
		t.Fatal("HTML output is empty")
	}

	showLen := len(output)
	if showLen > 500 {
		showLen = 500
	}
	t.Logf("Output first %d chars:\n%s", showLen, output[:showLen])

	// Check for HTML5 structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE declaration")
	}
	if !strings.Contains(output, "</html>") {
		t.Error("Missing closing HTML tag")
	}

	// Check for CSS variables
	if !strings.Contains(output, "--color-error") {
		t.Error("Missing --color-error CSS variable")
	}
	if !strings.Contains(output, "--color-warning") {
		t.Error("Missing --color-warning CSS variable")
	}
	if !strings.Contains(output, "--color-info") {
		t.Error("Missing --color-info CSS variable")
	}

	// Check for print media query
	if !strings.Contains(output, "@media print") {
		t.Error("Missing @media print styles")
	}

	// Check for responsive design
	if !strings.Contains(output, "max-width") {
		t.Error("Missing responsive max-width")
	}

	// Check violation data appears
	if !strings.Contains(output, "D-01") {
		t.Error("Missing violation ID in output")
	}
}

func TestHTMLReporter_ViolationSorting(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID:          "viol-002",
			File:        "z_file.go",
			Line:        5,
			SourceLayer: "domain",
			TargetLayer: "infra",
			Import:      "import",
			Message:     "msg",
			Severity:    domain.SeverityError,
		},
		{
			ID:          "viol-001",
			File:        "a_file.go",
			Line:        10,
			SourceLayer: "domain",
			TargetLayer: "infra",
			Import:      "import",
			Message:     "msg",
			Severity:    domain.SeverityError,
		},
		{
			ID:          "viol-003",
			File:        "a_file.go",
			Line:        3,
			SourceLayer: "domain",
			TargetLayer: "infra",
			Import:      "import",
			Message:     "msg",
			Severity:    domain.SeverityError,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check sorting order: a_file.go:3 should appear before a_file.go:10, which should appear before z_file.go:5
	aFile3Pos := strings.Index(output, "a_file.go:3")
	aFile10Pos := strings.Index(output, "a_file.go:10")
	zFile5Pos := strings.Index(output, "z_file.go:5")

	if aFile3Pos == -1 || aFile10Pos == -1 || zFile5Pos == -1 {
		t.Fatal("Missing file references in output")
	}

	if aFile3Pos > aFile10Pos {
		t.Error("Violations not sorted correctly: a_file.go:3 should come before a_file.go:10")
	}
	if aFile10Pos > zFile5Pos {
		t.Error("Violations not sorted correctly: a_file.go:10 should come before z_file.go:5")
	}
}

func TestHTMLReporter_OverriddenViolation(t *testing.T) {
	reporter := NewHTMLReporter()
	violations := []domain.Violation{
		{
			ID:               "viol-001",
			RuleID:           "test-rule",
			File:             "test.go",
			Line:             10,
			SourceLayer:      "domain",
			TargetLayer:      "infrastructure",
			Import:           "import",
			Message:          "Test violation",
			Severity:         domain.SeverityWarning,
			OriginalSeverity: domain.SeverityError,
			Overridden:       true,
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := reporter.Report(violations, ports.OutputFormatHTML)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check for overridden indicator
	if !strings.Contains(output, "overridden") {
		t.Error("Missing overridden indicator")
	}

	// Check for original severity
	if !strings.Contains(output, "error") {
		t.Error("Missing original severity reference")
	}
}
