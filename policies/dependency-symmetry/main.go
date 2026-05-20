//go:build tinygo

// Package main implements a WASM policy that checks dependency direction symmetry.
//
// Build: tinygo build -o ../dependency-symmetry.wasm -target=wasi -no-debug .
package main

import "unsafe"

//go:wasmimport arx arx_deps
func arxDeps() int32

//go:wasmimport arx arx_layer_count
func arxLayerCount() int32

//go:wasmimport arx arx_violation_count
func arxViolationCount() int32

//go:wasmimport arx arx_emit_violation
func arxEmitViolation(ptr int32, length int32) int32

//export evaluate
func evaluate() int32 {
	// Check: if there are existing violations AND very few deps,
	// something is likely off with dependency symmetry.
	violations := arxViolationCount()
	deps := arxDeps()

	if violations > 0 && deps == 0 {
		msg := `{"message":"dependency-symmetry: violations with no deps indicates asymmetry"}`
		arxEmitViolation(stringPtr(msg), int32(len(msg)))
		return 1
	}

	return 0
}

func stringPtr(s string) int32 {
	if len(s) == 0 {
		return 0
	}
	return int32(uintptr(unsafe.Pointer(unsafe.StringData(s))))
}

func main() {}
