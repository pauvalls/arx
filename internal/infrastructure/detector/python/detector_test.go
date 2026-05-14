package python

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestDetector_Name(t *testing.T) {
	t.Parallel()

	detector := New()
	if name := detector.Name(); name != "python" {
		t.Errorf("Expected name 'python', got %q", name)
	}
}

func TestDetector_Detect_PythonProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests==2.28.0\n"), 0644)

	detector := New()
	isPython, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !isPython {
		t.Error("Expected to detect Python project")
	}
}

func TestDetector_Detect_NotPythonProject(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	detector := New()
	isPython, err := detector.Detect(context.Background(), tmpDir)

	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if isPython {
		t.Error("Expected to not detect Python project")
	}
}

func TestDetector_findPythonFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "utils.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(""), 0644)

	detector := New()
	files, err := detector.findPythonFiles(tmpDir)

	if err != nil {
		t.Fatalf("findPythonFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 Python files, got %d", len(files))
	}
}

func TestDetector_findPythonFiles_SkipsVenv(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "venv", "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "venv", "lib", "site.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(""), 0644)

	detector := New()
	files, err := detector.findPythonFiles(tmpDir)

	if err != nil {
		t.Fatalf("findPythonFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 Python file (venv skipped), got %d", len(files))
	}
}

func TestDetector_importMatchesLayer(t *testing.T) {
	t.Parallel()

	detector := New()

	tests := []struct {
		name         string
		importPath   string
		layerPattern string
		expected     bool
	}{
		{"exact match", "domain/models", "domain/**", true},
		{"nested match", "domain/models/user", "domain/**", true},
		{"no match", "infrastructure/db", "domain/**", false},
		{"single star", "domain/models", "domain/*", true},
		{"single star no nested", "domain/models/user", "domain/*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := detector.importMatchesLayer(tt.importPath, tt.layerPattern)
			if result != tt.expected {
				t.Errorf("importMatchesLayer(%q, %q) = %v, want %v",
					tt.importPath, tt.layerPattern, result, tt.expected)
			}
		})
	}
}

func TestDetector_ExtractImports_Cancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("import os\n"), 0644)

	detector := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := detector.ExtractImports(ctx, tmpDir, []domain.Layer{})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}
