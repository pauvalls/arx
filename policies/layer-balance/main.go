//go:build tinygo

// Package main implements a WASM policy that checks layer file counts.
//
// Build: tinygo build -o ../layer-balance.wasm -target=wasi -no-debug .
package main

import "unsafe"

//go:wasmimport arx arx_deps
func arxDeps() int32

//go:wasmimport arx arx_layer_count
func arxLayerCount() int32

//go:wasmimport arx arx_emit_violation
func arxEmitViolation(ptr int32, length int32) int32

//export evaluate
func evaluate() int32 {
	deps := arxDeps()
	layers := arxLayerCount()

	if layers > 0 && deps > layers*10 {
		msg := `{"message":"layer-balance: deps/layer ratio too high"}`
		arxEmitViolation(stringPtr(msg), int32(len(msg)))
		return 1
	}

	return 0
}

//go:wasmimport arx arx_get_dep
func arxGetDep(idx int32) int64

func stringPtr(s string) int32 {
	if len(s) == 0 {
		return 0
	}
	return int32(uintptr(unsafe.Pointer(unsafe.StringData(s))))
}

func main() {}
