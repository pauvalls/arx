package config

import (
	"os"
	"testing"
)

func TestInterpolateEnvVars_BasicDollarVar(t *testing.T) {
	os.Setenv("TEST_VERSION", "1.0")
	defer os.Unsetenv("TEST_VERSION")

	input := []byte(`version: $TEST_VERSION`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "version: 1.0" {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "version: 1.0")
	}
}

func TestInterpolateEnvVars_BraceVar(t *testing.T) {
	os.Setenv("TEST_VERSION", "2.0")
	defer os.Unsetenv("TEST_VERSION")

	input := []byte(`version: ${TEST_VERSION}`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "version: 2.0" {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "version: 2.0")
	}
}

func TestInterpolateEnvVars_DefaultValue(t *testing.T) {
	// Unset — should use default
	os.Unsetenv("UNDEFINED_VAR")

	input := []byte(`path: ${UNDEFINED_VAR:-/default/path}`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "path: /default/path" {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "path: /default/path")
	}
}

func TestInterpolateEnvVars_DefaultValueWithSetEnv(t *testing.T) {
	os.Setenv("EXISTING_VAR", "custom")
	defer os.Unsetenv("EXISTING_VAR")

	input := []byte(`path: ${EXISTING_VAR:-/fallback}`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "path: custom" {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "path: custom")
	}
}

func TestInterpolateEnvVars_DollarEscape(t *testing.T) {
	input := []byte(`price: $$100`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "price: $100" {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "price: $100")
	}
}

func TestInterpolateEnvVars_MissingVarNoDefault(t *testing.T) {
	os.Unsetenv("MISSING_VAR")

	input := []byte(`version: ${MISSING_VAR}`)
	_, err := InterpolateEnvVars(input)
	if err == nil {
		t.Fatal("InterpolateEnvVars() expected error for missing var, got nil")
	}
}

func TestInterpolateEnvVars_NoVarsNoChange(t *testing.T) {
	input := []byte(`version: "1.0"
layers:
  - name: domain
    paths: ["./domain"]`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("InterpolateEnvVars() changed input with no vars:\n got:  %q\nwant: %q", string(output), string(input))
	}
}

func TestInterpolateEnvVars_MultipleVars(t *testing.T) {
	os.Setenv("SRC", "src")
	os.Setenv("DST", "dst")
	defer func() {
		os.Unsetenv("SRC")
		os.Unsetenv("DST")
	}()

	input := []byte(`from: $SRC
to: [$DST]`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	expected := "from: src\nto: [dst]"
	if string(output) != expected {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), expected)
	}
}

func TestInterpolateEnvVars_EmptyDefault(t *testing.T) {
	os.Unsetenv("UNDEFINED")

	input := []byte(`value: ${UNDEFINED:-}`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != "value: " {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), "value: ")
	}
}

func TestInterpolateEnvVars_NoFalsePositivesInAnchors(t *testing.T) {
	// YAML anchors use & and *, not $ — make sure we don't match accidentally
	input := []byte(`default: &default
  path: ./defaults
override:
  <<: *default`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	if string(output) != string(input) {
		t.Errorf("InterpolateEnvVars() should not modify YAML anchors:\n got:  %q\nwant: %q", string(output), string(input))
	}
}

func TestInterpolateEnvVars_LiteralDollarInKey(t *testing.T) {
	// When a $ appears in what looks like a key position (followed by alphanum then :),
	// the raw-bytes processing treats it as a variable. Users should $$-escape literal
	// dollar signs in keys. This test verifies no crash with $ in content.
	os.Setenv("SCHEMA", "yes")
	defer os.Unsetenv("SCHEMA")

	input := []byte(`$$schema: file.json
version: $SCHEMA`)
	output, err := InterpolateEnvVars(input)
	if err != nil {
		t.Fatalf("InterpolateEnvVars() error = %v", err)
	}
	// $$schema → $schema (escaped), $SCHEMA → "yes"
	expected := "$schema: file.json\nversion: yes"
	if string(output) != expected {
		t.Errorf("InterpolateEnvVars() = %q, want %q", string(output), expected)
	}
}
