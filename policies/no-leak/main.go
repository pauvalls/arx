//go:build tinygo

// Package main implements a WASM policy that checks for leaked layer dependencies.
//
// Build: tinygo build -o ../no-leak.wasm -target=wasi -no-debug .
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
	// Check: if deps are present but no violations, something is leaking.
	deps := arxDeps()
	violations := arxViolationCount()

	if deps > 0 && violations == 0 {
		msg := `{"message":"no-leak: deps exist without any violations — potential leak"}`
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
