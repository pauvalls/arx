package wasm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("")
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	defer m.Close()
}

func TestManager_GetEvaluator(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(dir)
	defer m.Close()

	eval, err := m.GetEvaluator(wasmPath)
	if err != nil {
		t.Fatalf("GetEvaluator() error = %v", err)
	}
	if eval == nil {
		t.Fatal("GetEvaluator() returned nil")
	}
}

func TestManager_GetEvaluator_Cached(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(dir)
	defer m.Close()

	// First call creates evaluator
	eval1, err := m.GetEvaluator(wasmPath)
	if err != nil {
		t.Fatalf("GetEvaluator() first call error = %v", err)
	}

	// Second call returns cached evaluator (or creates a new one)
	eval2, err := m.GetEvaluator(wasmPath)
	if err != nil {
		t.Fatalf("GetEvaluator() second call error = %v", err)
	}

	// Both should be functional
	if eval1 == nil || eval2 == nil {
		t.Fatal("GetEvaluator() returned nil")
	}
}

func TestManager_GetEvaluator_InvalidPath(t *testing.T) {
	m := NewManager("")
	defer m.Close()

	_, err := m.GetEvaluator("/nonexistent/path.wasm")
	if err == nil {
		t.Fatal("GetEvaluator() should return error for invalid path")
	}
}

func TestManager_Close(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(dir)

	eval, err := m.GetEvaluator(wasmPath)
	if err != nil {
		t.Fatalf("GetEvaluator() error = %v", err)
	}

	// Close the manager
	if err := m.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Evaluator should still be closable (double close is safe)
	if err := eval.Close(); err != nil {
		t.Errorf("eval.Close() after manager close error = %v", err)
	}
}

func TestManager_MultipleEvaluators(t *testing.T) {
	dir := t.TempDir()

	// Create two different WASM files
	wasmPath1 := filepath.Join(dir, "test1.wasm")
	if err := os.WriteFile(wasmPath1, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	wasmPath2 := filepath.Join(dir, "test2.wasm")
	if err := os.WriteFile(wasmPath2, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(dir)
	defer m.Close()

	eval1, err := m.GetEvaluator(wasmPath1)
	if err != nil {
		t.Fatalf("GetEvaluator(1) error = %v", err)
	}

	eval2, err := m.GetEvaluator(wasmPath2)
	if err != nil {
		t.Fatalf("GetEvaluator(2) error = %v", err)
	}

	if eval1 == nil || eval2 == nil {
		t.Fatal("evaluators should not be nil")
	}

	// Close should clean up both
	if err := m.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestManager_CacheDirectory(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, ".arx-cache", "policies")
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(cacheDir)
	defer m.Close()

	_, err := m.GetEvaluator(wasmPath)
	if err != nil {
		t.Fatalf("GetEvaluator() error = %v", err)
	}

	// Cache directory should have been created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("cache directory should exist at %s", cacheDir)
	}
}
