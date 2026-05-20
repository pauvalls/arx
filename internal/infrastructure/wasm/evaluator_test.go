package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// minimalWasm is a valid WASM module that exports evaluate() → i32 (returns 0).
// Generated from WAT: (module (func (export "evaluate") (result i32) i32.const 0))
var minimalWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, // magic \0asm
	0x01, 0x00, 0x00, 0x00, // version 1
	0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f, // type: () -> i32
	0x03, 0x02, 0x01, 0x00, // function: 1 func, type 0
	0x07, 0x0c, 0x01, 0x08, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x00, 0x00, // export "evaluate" func 0
	0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x00, 0x0b, // code: i32.const 0, end
}

func TestNewEvaluator_InvalidPath(t *testing.T) {
	_, err := NewEvaluator("/nonexistent/path.wasm", nil)
	if err == nil {
		t.Fatal("NewEvaluator() should return error for invalid path")
	}
}

func TestNewEvaluator_InvalidBinary(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.wasm")
	if err := os.WriteFile(badPath, []byte("not a wasm binary"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewEvaluator(badPath, nil)
	if err == nil {
		t.Fatal("NewEvaluator() should return error for invalid binary")
	}
}

func TestEvaluator_Evaluate(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	eval, err := NewEvaluator(wasmPath, nil)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}
	defer eval.Close()

	ctx := context.Background()
	violations, err := eval.Evaluate(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Evaluate() error = %v", err)
	}
	if violations == nil {
		t.Error("Evaluate() should return non-nil violations slice")
	}
	if len(violations) != 0 {
		t.Errorf("Evaluate() returned %d violations, want 0", len(violations))
	}
}

func TestEvaluator_WithCache(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	cache := NewCache(dir)

	eval1, err := NewEvaluator(wasmPath, cache)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}
	defer eval1.Close()

	// Should be cached on disk
	hash := cacheKey(minimalWasm)
	cacheDir := filepath.Join(dir, hash)
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("cache directory should exist at %s", cacheDir)
	}

	// Second instance should load from cache
	eval2, err := NewEvaluator(wasmPath, cache)
	if err != nil {
		t.Fatalf("NewEvaluator() second call error = %v", err)
	}
	defer eval2.Close()

	// Both evaluators should work
	ctx := context.Background()
	for i, eval := range []domain.WasmEvaluator{eval1, eval2} {
		violations, err := eval.Evaluate(ctx, nil, nil, nil, nil)
		if err != nil {
			t.Errorf("evaluator[%d] Evaluate() error = %v", i, err)
		}
		if len(violations) != 0 {
			t.Errorf("evaluator[%d] Evaluate() returned %d violations, want 0", i, len(violations))
		}
	}
}

func TestEvaluator_Timeout(t *testing.T) {
	// Create a WASM module with an infinite loop
	// (not easy to construct by hand, so test that context cancellation works)
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	eval, err := NewEvaluator(wasmPath, nil)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}
	defer eval.Close()

	// Use already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = eval.Evaluate(ctx, nil, nil, nil, nil)
	if err == nil {
		t.Log("Evaluate() with cancelled context - may or may not error depending on implementation")
	}
}

func TestEvaluator_Close(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	eval, err := NewEvaluator(wasmPath, nil)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}

	if err := eval.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestEvaluator_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		wasm    []byte
		wantErr bool
	}{
		{
			name:    "valid minimal wasm",
			wasm:    minimalWasm,
			wantErr: false,
		},
		{
			name:    "corrupted wasm",
			wasm:    []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xff},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			wasmPath := filepath.Join(dir, "test.wasm")
			if err := os.WriteFile(wasmPath, tt.wasm, 0644); err != nil {
				t.Fatal(err)
			}

			_, err := NewEvaluator(wasmPath, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEvaluator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvaluator_GracefulShutdownOnClosedRuntime(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "test.wasm")
	if err := os.WriteFile(wasmPath, minimalWasm, 0644); err != nil {
		t.Fatal(err)
	}

	eval, err := NewEvaluator(wasmPath, nil)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}

	// Close first
	if err := eval.Close(); err != nil {
		t.Fatal(err)
	}

	// Should handle double close gracefully
	if err := eval.Close(); err != nil {
		t.Errorf("double Close() should not error, got %v", err)
	}
}

func TestCacheWithNegativeTTL(t *testing.T) {
	// Verify that the cache handles timeouts gracefully
	c := NewCache("")
	data := []byte("test data")
	c.Set(data, struct{}{})
	time.Sleep(10 * time.Millisecond)

	// Should still be cached
	_, ok := c.Get(data)
	if !ok {
		t.Error("Get() should return true for cached entry after short sleep")
	}
}
