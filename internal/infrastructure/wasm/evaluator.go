package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/pauvalls/arx/internal/domain"
)

// MaxWasmSize is the maximum allowed WASM module size in bytes (1MB).
const MaxWasmSize = 1 * 1024 * 1024

// wasmEvaluator implements domain.WasmEvaluator using the wazero runtime.
type wasmEvaluator struct {
	mu       sync.Mutex
	runtime  wazero.Runtime
	compiled wazero.CompiledModule

	// Per-evaluation state (set before each Evaluate call).
	deps     []domain.Dependency
	layers   []domain.Layer
	existing []domain.Violation
	emitted  []domain.Violation
}

// NewEvaluator creates a new WASM policy evaluator from the given wasm file.
// If cache is non-nil, the evaluator will use it for storing/retrieving compiled modules.
// Returns an error if the WASM module exceeds MaxWasmSize (1MB).
func NewEvaluator(wasmPath string, cache *Cache) (domain.WasmEvaluator, error) {
	// Check file size before reading to prevent loading oversized modules
	info, err := os.Stat(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat wasm file %s: %w", wasmPath, err)
	}
	if info.Size() > MaxWasmSize {
		return nil, fmt.Errorf("WASM module exceeds %d byte limit (size: %d)", MaxWasmSize, info.Size())
	}

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wasm file %s: %w", wasmPath, err)
	}

	// Store raw bytes in disk cache for later retrieval
	if cache != nil {
		// Store raw bytes so that the cache can serve them later without re-reading
		cache.Set(wasmBytes, struct{}{})
	}

	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Set up WASI support (needed for WASI-compiled modules)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the WASM module
	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("failed to compile wasm module %s: %w", wasmPath, err)
	}

	e := &wasmEvaluator{
		runtime:  r,
		compiled: compiled,
	}

	// Set up ARX host module
	if err := e.setupHostModule(ctx); err != nil {
		compiled.Close(ctx)
		r.Close(ctx)
		return nil, fmt.Errorf("failed to set up host module: %w", err)
	}

	return e, nil
}

// setupHostModule creates and instantiates the ARX host module with host API functions.
func (e *wasmEvaluator) setupHostModule(ctx context.Context) error {
	_, err := e.runtime.NewHostModuleBuilder("arx").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			e.mu.Lock()
			n := uint32(len(e.deps))
			e.mu.Unlock()
			return n
		}).
		Export("arx_deps").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			e.mu.Lock()
			n := uint32(len(e.layers))
			e.mu.Unlock()
			return n
		}).
		Export("arx_layer_count").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, _ api.Module) uint32 {
			e.mu.Lock()
			n := uint32(len(e.existing))
			e.mu.Unlock()
			return n
		}).
		Export("arx_violation_count").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, idx uint32) uint64 {
			return e.hostGetDep(m, idx)
		}).
		Export("arx_get_dep").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, ptr uint32, length uint32) uint32 {
			return e.hostEmitViolation(m, ptr, length)
		}).
		Export("arx_emit_violation").
		Instantiate(ctx)
	return err
}

// hostGetDep writes the dependency at the given index into guest memory as JSON.
// Returns a uint64 where the low 32 bits are the memory offset and high 32 bits are the data length.
func (e *wasmEvaluator) hostGetDep(m api.Module, idx uint32) uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	if int(idx) >= len(e.deps) {
		return 0
	}
	dep := e.deps[idx]
	data, err := json.Marshal(dep)
	if err != nil {
		return 0
	}
	// Write JSON to guest memory at offset 0
	if !m.Memory().Write(0, data) {
		return 0
	}
	return (uint64(len(data)) << 32) | 0
}

// hostEmitViolation reads a JSON-encoded violation from guest memory and appends it.
func (e *wasmEvaluator) hostEmitViolation(m api.Module, ptr uint32, length uint32) uint32 {
	data, ok := m.Memory().Read(ptr, length)
	if !ok {
		return 1
	}
	var v domain.Violation
	if err := json.Unmarshal(data, &v); err != nil {
		return 1
	}
	e.mu.Lock()
	e.emitted = append(e.emitted, v)
	e.mu.Unlock()
	return 0
}

// Evaluate runs the WASM policy with the given dependencies, layers, and existing violations.
func (e *wasmEvaluator) Evaluate(ctx context.Context, deps []domain.Dependency, layers []domain.Layer, violations []domain.Violation, params map[string]interface{}) ([]domain.Violation, error) {
	e.mu.Lock()

	// Set per-evaluation state (host functions will read these)
	e.deps = deps
	e.layers = layers
	e.existing = violations
	e.emitted = nil
	e.mu.Unlock()

	// Apply default timeout of 5 seconds
	evalCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		evalCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	// Create a fresh module instance for this evaluation
	config := wazero.NewModuleConfig().WithName("").WithSysNanotime().WithSysWalltime()
	instance, err := e.runtime.InstantiateModule(evalCtx, e.compiled, config)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer func() {
		_ = instance.Close(evalCtx)
	}()

	// Call the evaluate() exported function
	evalFn := instance.ExportedFunction("evaluate")
	if evalFn == nil {
		return nil, fmt.Errorf("module does not export 'evaluate' function")
	}

	results, err := evalFn.Call(evalCtx)
	if err != nil {
		return nil, fmt.Errorf("evaluate() call failed: %w", err)
	}

	// Collect and return emitted violations
	e.mu.Lock()
	result := make([]domain.Violation, len(e.emitted))
	copy(result, e.emitted)
	e.mu.Unlock()

	_ = results // result count matches emitted; we trust hostEmitViolation tracked them
	return result, nil
}

// Close releases all resources held by the evaluator.
func (e *wasmEvaluator) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.runtime != nil {
		ctx := context.Background()
		if e.compiled != nil {
			_ = e.compiled.Close(ctx)
		}
		return e.runtime.Close(ctx)
	}
	return nil
}
