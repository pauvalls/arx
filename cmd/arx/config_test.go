package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- resolvePath tests ---

func TestResolvePath_TopLevel(t *testing.T) {
	parts, err := resolvePath("version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 1 || parts[0] != "version" {
		t.Errorf("expected [version], got %v", parts)
	}
}

func TestResolvePath_Nested(t *testing.T) {
	parts, err := resolvePath("severity_mapping.critical")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 2 || parts[0] != "severity_mapping" || parts[1] != "critical" {
		t.Errorf("expected [severity_mapping critical], got %v", parts)
	}
}

func TestResolvePath_DeeplyNested(t *testing.T) {
	parts, err := resolvePath("a.b.c.d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 4 {
		t.Errorf("expected 4 parts, got %d", len(parts))
	}
}

func TestResolvePath_EmptyKey(t *testing.T) {
	_, err := resolvePath("")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestResolvePath_EmptySegment(t *testing.T) {
	_, err := resolvePath("a..b")
	if err == nil {
		t.Error("expected error for empty segment")
	}
}

// --- parseValue tests ---

func TestParseValue_JSONArray(t *testing.T) {
	v, err := parseValue(`["vendor/**","test/**"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", v)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 elements, got %d", len(arr))
	}
	if arr[0] != "vendor/**" {
		t.Errorf("expected vendor/**, got %v", arr[0])
	}
}

func TestParseValue_JSONObject(t *testing.T) {
	v, err := parseValue(`{"key":"value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", v)
	}
	if m["key"] != "value" {
		t.Errorf("expected value, got %v", m["key"])
	}
}

func TestParseValue_JSONNumber(t *testing.T) {
	v, err := parseValue("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num, ok := v.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", v)
	}
	if num != 42 {
		t.Errorf("expected 42, got %v", num)
	}
}

func TestParseValue_JSONBool(t *testing.T) {
	v, err := parseValue("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", v)
	}
	if !b {
		t.Error("expected true")
	}
}

func TestParseValue_StringFallback(t *testing.T) {
	v, err := parseValue("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "hello" {
		t.Errorf("expected hello, got %v", v)
	}
}

func TestParseValue_StringWithSpaces(t *testing.T) {
	v, err := parseValue("some value with spaces")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "some value with spaces" {
		t.Errorf("expected 'some value with spaces', got %v", v)
	}
}

// --- setAtPath tests ---

func TestSetAtPath_TopLevel(t *testing.T) {
	doc := make(map[string]interface{})
	err := setAtPath(doc, []string{"max_violations"}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["max_violations"] != 10 {
		t.Errorf("expected 10, got %v", doc["max_violations"])
	}
}

func TestSetAtPath_Nested(t *testing.T) {
	doc := map[string]interface{}{
		"severity_mapping": map[string]interface{}{
			"critical": "error",
		},
	}
	err := setAtPath(doc, []string{"severity_mapping", "critical"}, "fatal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := doc["severity_mapping"].(map[string]interface{})
	if m["critical"] != "fatal" {
		t.Errorf("expected fatal, got %v", m["critical"])
	}
}

func TestSetAtPath_CreatesIntermediateMaps(t *testing.T) {
	doc := make(map[string]interface{})
	err := setAtPath(doc, []string{"a", "b", "c"}, "leaf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := doc["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	if b["c"] != "leaf" {
		t.Errorf("expected leaf, got %v", b["c"])
	}
}

func TestSetAtPath_ArrayValue(t *testing.T) {
	doc := make(map[string]interface{})
	arr := []interface{}{"vendor/**", "test/**"}
	err := setAtPath(doc, []string{"exclude"}, arr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := doc["exclude"].([]interface{})
	if len(got) != 2 {
		t.Errorf("expected 2 elements, got %d", len(got))
	}
}

func TestSetAtPath_EmptyPath(t *testing.T) {
	doc := make(map[string]interface{})
	err := setAtPath(doc, []string{}, "value")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestSetAtPath_TypeMismatch(t *testing.T) {
	doc := map[string]interface{}{
		"max_violations": 10,
	}
	err := setAtPath(doc, []string{"max_violations", "nested"}, "value")
	if err == nil {
		t.Error("expected error when traversing into non-map")
	}
}

// --- getAtPath tests ---

func TestGetAtPath_TopLevel(t *testing.T) {
	doc := map[string]interface{}{"version": "1.0"}
	v, err := getAtPath(doc, []string{"version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "1.0" {
		t.Errorf("expected 1.0, got %v", v)
	}
}

func TestGetAtPath_Nested(t *testing.T) {
	doc := map[string]interface{}{
		"severity_mapping": map[string]interface{}{
			"critical": "error",
		},
	}
	v, err := getAtPath(doc, []string{"severity_mapping", "critical"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "error" {
		t.Errorf("expected error, got %v", v)
	}
}

func TestGetAtPath_MissingPath(t *testing.T) {
	doc := map[string]interface{}{"version": "1.0"}
	_, err := getAtPath(doc, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestGetAtPath_MissingNested(t *testing.T) {
	doc := map[string]interface{}{
		"severity_mapping": map[string]interface{}{
			"critical": "error",
		},
	}
	_, err := getAtPath(doc, []string{"severity_mapping", "unknown"})
	if err == nil {
		t.Error("expected error for missing nested key")
	}
}

func TestGetAtPath_TypeMismatch(t *testing.T) {
	doc := map[string]interface{}{
		"max_violations": 10,
	}
	_, err := getAtPath(doc, []string{"max_violations", "nested"})
	if err == nil {
		t.Error("expected error when traversing into non-map")
	}
}

func TestGetAtPath_EmptyPath(t *testing.T) {
	doc := make(map[string]interface{})
	_, err := getAtPath(doc, []string{})
	if err == nil {
		t.Error("expected error for empty path")
	}
}

// --- Integration tests ---

func TestConfigSetGet_DottedPathArray(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	initialConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
severity_mapping:
  critical: error
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Set nested array
	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"config", "set", "severity_mapping.critical", `["vendor/**"]`})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), `- vendor/**`) {
		t.Errorf("expected YAML array with vendor/**, got:\n%s", string(data))
	}
}

func TestConfigGet_DottedPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	initialConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
severity_mapping:
  critical: error
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"config", "get", "severity_mapping.critical"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config get failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "error") {
		t.Errorf("expected 'error' in output, got: %s", output)
	}
}

func TestConfigSet_TopLevelNumber(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	initialConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
max_violations: 5
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"config", "set", "max_violations", "10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "max_violations: 10") {
		t.Errorf("expected max_violations: 10, got:\n%s", string(data))
	}
}

func TestConfigSet_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	initialConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)
	cmd.SetArgs([]string{"config", "set", "nonexistent.field", "value"})
	err := cmd.Execute()
	// Should NOT error — dotted path creates new keys (per design decision)
	if err != nil {
		t.Errorf("expected no error for new key creation, got: %v", err)
	}
}

// TestConfigValidateValidConfig tests validation of a valid config file
func TestConfigValidateValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	validConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", configPath}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("config validate should succeed, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "✓ Config valid") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// TestConfigValidateInvalidConfig tests validation of an invalid config file
func TestConfigValidateInvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	invalidConfig := `version: "1.0"
# Missing layers and rules
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", configPath}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err == nil {
		t.Error("config validate should fail for invalid config")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "✗") {
		t.Errorf("expected error message with ✗, got: %s", errOutput)
	}
}

// TestConfigValidateMissingFile tests validation when file doesn't exist
func TestConfigValidateMissingFile(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", "/nonexistent/path/arx.yaml"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err == nil {
		t.Error("config validate should fail for missing file")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Config file not found") {
		t.Errorf("expected 'Config file not found' error, got: %s", errOutput)
	}
}

// TestConfigValidateInvalidYAML tests validation of malformed YAML
func TestConfigValidateInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	invalidYAML := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
  invalid yaml: [
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", configPath}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err == nil {
		t.Error("config validate should fail for invalid YAML")
	}

	errOutput := stderr.String()
	if !strings.Contains(errOutput, "✗") {
		t.Errorf("expected error message, got: %s", errOutput)
	}
}

// TestConfigValidateWithPathFlag tests using --path flag
func TestConfigValidateWithPathFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.yaml")
	validConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", "--path", configPath}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("config validate should succeed, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "✓ Config valid") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// TestConfigValidateDefaultPath tests validation with default path (arx.yaml)
func TestConfigValidateDefaultPath(t *testing.T) {
	// Reset the package-level flag (may have been set by previous test)
	configValidatePath = ""

	// Use a temp dir with valid config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "arx.yaml")
	validConfig := `version: "1.0"
layers:
  - name: domain
    paths: [./domain]
rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	// Change to temp dir so default path resolves correctly
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	args := []string{"config", "validate"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("config validate should succeed, got error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "✓ Config valid") {
		t.Errorf("expected success message, got: %s", output)
	}
}
