package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestDiffRenderer_RenderJSON_Structure(t *testing.T) {
	result := ports.DiffResultData{
		RefBefore:     "v1.0.0",
		RefAfter:      "v2.0.0",
		ConfigChanged: false,
		Added: []domain.Violation{
			{
				ID:          "V001",
				RuleID:      "R001",
				Severity:    domain.SeverityError,
				File:        "internal/domain/user.go",
				Line:        10,
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/db",
				Message:     "domain cannot depend on infrastructure",
			},
		},
		Resolved: []domain.Violation{
			{
				ID:          "V002",
				RuleID:      "R002",
				Severity:    domain.SeverityWarning,
				File:        "internal/app/service.go",
				Line:        20,
				SourceLayer: "application",
				TargetLayer: "domain",
				Import:      "github.com/example/entity",
				Message:     "application should use ports",
			},
		},
		Unchanged: []domain.Violation{
			{
				ID:          "V003",
				RuleID:      "R003",
				Severity:    domain.SeverityError,
				File:        "internal/domain/handler.go",
				Line:        30,
				SourceLayer: "domain",
				TargetLayer: "presentation",
				Import:      "github.com/example/http",
				Message:     "domain cannot depend on presentation",
			},
		},
	}

	renderer := NewDiffRenderer()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderer.RenderJSON(result)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Parse JSON
	var parsed DiffJSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}

	// Verify all fields
	if parsed.RefBefore != "v1.0.0" {
		t.Errorf("ref_before = %q, want %q", parsed.RefBefore, "v1.0.0")
	}
	if parsed.RefAfter != "v2.0.0" {
		t.Errorf("ref_after = %q, want %q", parsed.RefAfter, "v2.0.0")
	}
	if parsed.ConfigChanged {
		t.Error("config_changed should be false")
	}
	if len(parsed.Added) != 1 {
		t.Errorf("added count = %d, want 1", len(parsed.Added))
	}
	if len(parsed.Resolved) != 1 {
		t.Errorf("resolved count = %d, want 1", len(parsed.Resolved))
	}
	if len(parsed.Unchanged) != 1 {
		t.Errorf("unchanged count = %d, want 1", len(parsed.Unchanged))
	}

	// Verify violation details
	if parsed.Added[0].ID != "V001" {
		t.Errorf("added[0].id = %q, want %q", parsed.Added[0].ID, "V001")
	}
	if parsed.Added[0].RuleID != "R001" {
		t.Errorf("added[0].rule_id = %q, want %q", parsed.Added[0].RuleID, "R001")
	}
	if parsed.Added[0].File != "internal/domain/user.go" {
		t.Errorf("added[0].file = %q, want %q", parsed.Added[0].File, "internal/domain/user.go")
	}
	if parsed.Added[0].Line != 10 {
		t.Errorf("added[0].line = %d, want 10", parsed.Added[0].Line)
	}
}

func TestDiffRenderer_RenderJSON_Empty(t *testing.T) {
	result := ports.DiffResultData{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
	}

	renderer := NewDiffRenderer()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderer.RenderJSON(result)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("RenderJSON() error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	var parsed DiffJSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(parsed.Added) != 0 || len(parsed.Resolved) != 0 || len(parsed.Unchanged) != 0 {
		t.Error("empty diff should have empty arrays")
	}
}

func TestDiffRenderer_RenderJSON_JqParseable(t *testing.T) {
	result := ports.DiffResultData{
		RefBefore: "HEAD~1",
		RefAfter:  "HEAD",
		Added: []domain.Violation{
			{RuleID: "R001", SourceLayer: "domain", TargetLayer: "infrastructure", Import: "x"},
		},
	}

	renderer := NewDiffRenderer()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = renderer.RenderJSON(result)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Verify it can be parsed as JSON (jq-parseable)
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("output is not jq-parseable JSON: %v", err)
	}

	// Verify top-level keys exist
	expectedKeys := []string{"ref_before", "ref_after", "config_changed", "added", "resolved", "unchanged", "summary"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}
