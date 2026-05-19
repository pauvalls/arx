package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resetConfigFlags resets all package-level config flags to their defaults.
// This MUST be called at the start of every test that uses cobra commands,
// because Go test ordering is non-deterministic and the flags are shared state.
func resetConfigFlags() {
	configValidatePath = ""
	configValidateStrict = false
	configValidateSchema = false
	configValidateOverride = ""
}

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
	resetConfigFlags()
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
	resetConfigFlags()
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
	resetConfigFlags()
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
	resetConfigFlags()
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
	resetConfigFlags()
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

// TestConfigValidateOverrideFlag verifies the --override flag is present
func TestConfigValidateOverrideFlag(t *testing.T) {
	if configValidateCmd.Flags().Lookup("override") == nil {
		t.Error("missing --override flag on config validate")
	}
}

// TestConfigValidateWithOverride verifies composed config validates correctly
func TestConfigValidateWithOverride(t *testing.T) {
	resetConfigFlags()
	tmpDir := t.TempDir()

	// Create base config with partial configuration
	baseConfig := `version: "1.0"
layers:
  - name: domain
    paths: ["./domain"]
rules: []
`
	basePath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(basePath, []byte(baseConfig), 0644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	// Create override that adds a rule
	overrideConfig := `rules:
  - id: test-rule
    from: domain
    to: [domain]
    type: Cannot
    severity: error
`
	overridePath := filepath.Join(tmpDir, "override.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideConfig), 0644); err != nil {
		t.Fatalf("failed to write override: %v", err)
	}

	// Reset package-level flags
	configValidatePath = ""
	configValidateOverride = ""

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", basePath, "--override", overridePath}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("config validate with --override should succeed, got: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "✓ Config valid") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// TestConfigValidateWithOverrideMissingFile
func TestConfigValidateWithOverrideMissingFile(t *testing.T) {
	resetConfigFlags()
	tmpDir := t.TempDir()

	baseConfig := `version: "1.0"
layers:
  - name: domain
    paths: ["./domain"]
rules: []
`
	basePath := filepath.Join(tmpDir, "arx.yaml")
	if err := os.WriteFile(basePath, []byte(baseConfig), 0644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	configValidatePath = ""
	configValidateOverride = ""

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", basePath, "--override", "/nonexistent/override.yaml"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err == nil {
		t.Error("config validate should fail for missing override file")
	}
}

// TestConfigValidateSchemaFlag verifies --schema still works with the new generator
func TestConfigValidateSchemaFlag(t *testing.T) {
	resetConfigFlags()
	configValidateSchema = true
	defer func() { configValidateSchema = false }()

	cmd := &cobra.Command{}
	cmd.AddCommand(configCmd)

	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	args := []string{"config", "validate", "--schema"}
	cmd.SetArgs(args)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config validate --schema error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "$schema") {
		t.Errorf("expected $schema field in output, got: %s", output)
	}
	if !strings.Contains(output, "properties") {
		t.Errorf("expected properties in output, got: %s", output)
	}
	if !strings.Contains(output, "version") {
		t.Errorf("expected version property in output, got: %s", output)
	}
}

// TestConfigValidateMissingFile tests validation when file doesn't exist
func TestConfigValidateMissingFile(t *testing.T) {
	resetConfigFlags()
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
	resetConfigFlags()
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
	resetConfigFlags()
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
	resetConfigFlags()

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
