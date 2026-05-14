package application

import (
	"strings"
	"testing"
)

func TestInitService_InitWithPreset_Valid(t *testing.T) {
	writer := &mockFileWriter{files: make(map[string][]byte)}
	presetService := NewPresetService()

	err := InitWithPreset("clean", "arx.yaml", false, writer, presetService)
	if err != nil {
		t.Fatalf("InitWithPreset failed: %v", err)
	}

	// Verify file was written
	content, ok := writer.files["arx.yaml"]
	if !ok {
		t.Fatal("config file was not written")
	}

	// Verify header contains preset name
	if !strings.Contains(string(content), "# Preset: clean") {
		t.Error("header should contain preset name")
	}

	// Verify header contains warning
	if !strings.Contains(string(content), "⚠️") {
		t.Error("header should contain warning emoji")
	}

	// Verify YAML content is valid
	if !strings.Contains(string(content), "version:") {
		t.Error("content should contain version field")
	}
}

func TestInitService_InitWithPreset_FileExists_NoForce(t *testing.T) {
	writer := &mockFileWriter{
		files:  make(map[string][]byte),
		exists: map[string]bool{"arx.yaml": true},
	}
	presetService := NewPresetService()

	err := InitWithPreset("clean", "arx.yaml", false, writer, presetService)
	if err == nil {
		t.Fatal("expected error for existing file without force, got nil")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention file exists: %v", err)
	}
}

func TestInitService_InitWithPreset_FileExists_WithForce(t *testing.T) {
	writer := &mockFileWriter{
		files:  make(map[string][]byte),
		exists: map[string]bool{"arx.yaml": true},
	}
	presetService := NewPresetService()

	err := InitWithPreset("clean", "arx.yaml", true, writer, presetService)
	if err != nil {
		t.Fatalf("InitWithPreset with force failed: %v", err)
	}

	// Verify file was overwritten
	content, ok := writer.files["arx.yaml"]
	if !ok {
		t.Fatal("config file was not written")
	}

	// Verify it's the new content (should contain preset header)
	if !strings.Contains(string(content), "# Preset: clean") {
		t.Error("file should be overwritten with preset content")
	}
}

func TestInitService_InitWithPreset_InvalidPreset(t *testing.T) {
	writer := &mockFileWriter{files: make(map[string][]byte)}
	presetService := NewPresetService()

	err := InitWithPreset("nonexistent", "arx.yaml", false, writer, presetService)
	if err == nil {
		t.Fatal("expected error for invalid preset, got nil")
	}

	if !strings.Contains(err.Error(), "failed to load preset") {
		t.Errorf("error should mention failed to load: %v", err)
	}
}

func TestInitWithPreset_HeaderFormat(t *testing.T) {
	writer := &mockFileWriter{files: make(map[string][]byte)}
	presetService := NewPresetService()

	err := InitWithPreset("hexagonal", "arx.yaml", false, writer, presetService)
	if err != nil {
		t.Fatalf("InitWithPreset failed: %v", err)
	}

	content := string(writer.files["arx.yaml"])
	lines := strings.Split(content, "\n")

	// Verify header structure
	expectedLines := []string{
		"# Arx Architecture Configuration",
		"# Preset: hexagonal",
		"# Generated:",
		"#",
		"# ⚠️  This is a starting point",
	}

	for i, expected := range expectedLines {
		if i >= len(lines) {
			t.Errorf("missing header line %d: %q", i, expected)
			continue
		}
		if !strings.Contains(lines[i], expected) {
			t.Errorf("header line %d: expected %q, got %q", i, expected, lines[i])
		}
	}
}
