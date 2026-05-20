package wasm

import (
	"sync"

	"github.com/pauvalls/arx/internal/domain"
)

// Manager manages wazero runtime instances and provides cached WASM evaluators.
type Manager struct {
	mu        sync.Mutex
	cache     *Cache
	cacheDir  string
	evaluators map[string]domain.WasmEvaluator
}

// NewManager creates a new WASM evaluator manager.
// cacheDir specifies the directory for persistent WASM caching (may be empty).
func NewManager(cacheDir string) *Manager {
	return &Manager{
		cache:      NewCache(cacheDir),
		cacheDir:   cacheDir,
		evaluators: make(map[string]domain.WasmEvaluator),
	}
}

// GetEvaluator returns a cached or new evaluator for the given wasmPath.
// Evaluators are cached by absolute file path to avoid redundant compilation.
func (m *Manager) GetEvaluator(wasmPath string) (domain.WasmEvaluator, error) {
	m.mu.Lock()
	// Check if we already have an evaluator for this path
	if eval, ok := m.evaluators[wasmPath]; ok {
		m.mu.Unlock()
		return eval, nil
	}
	m.mu.Unlock()

	// Create a new evaluator
	eval, err := NewEvaluator(wasmPath, m.cache)
	if err != nil {
		return nil, err
	}

	// Cache the evaluator for reuse
	m.mu.Lock()
	m.evaluators[wasmPath] = eval
	m.mu.Unlock()

	return eval, nil
}

// Close closes all managed evaluators and releases all resources.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for path, eval := range m.evaluators {
		if err := eval.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(m.evaluators, path)
	}
	return firstErr
}
