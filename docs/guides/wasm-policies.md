# WASM Policies

## What Are Policies?

WASM policies let you write architecture rules in **any language** that compiles to WebAssembly. While YAML-configured rules (Cannot, Must, MustNotCircular) cover common patterns, policies enable arbitrary logic — cross-cutting concerns, custom metrics, data flow validation, and anything else you can express in code.

```yaml
rules:
  - id: P-01
    wasm:
      path: policies/layer-balance.wasm
      params: { min: 3, max: 8 }
    severity: error
```

## How They Work

1. You write a policy in any WASM-capable language (TinyGo, Rust, AssemblyScript, C, etc.)
2. Compile it to a `.wasm` binary
3. Configure the policy path in your `arx.yaml`
4. During `arx check`, arx evaluates the WASM module via [wazero](https://wazero.io) — a pure Go WebAssembly runtime with zero CGO dependencies

The evaluation flow:

```
Config → Load WASM → Create evaluator → Call evaluate() → Collect violations
```

Each policy is cached in `.arx-cache/policies/<sha256>/wasm.bin` for fast re-evaluation.

## Host API

WASM policies import host functions from the `arx` module. These give policies access to the current architecture state:

### `arx_deps() -> i32`

Returns the total number of dependencies detected in the project.

### `arx_layer_count() -> i32`

Returns the number of configured layers.

### `arx_violation_count() -> i32`

Returns the number of violations from previous rule evaluation phases.

### `arx_get_dep(idx) -> i64`

Returns dependency data at index `idx` as a JSON string written to guest memory. Returns a pointer/length pair encoded as `i64`. The guest reads this from its own linear memory.

### `arx_emit_violation(ptr, len) -> i32`

Emits a violation from the WASM guest. Takes a pointer and length to a JSON string in guest memory:

```json
{
  "rule_id": "P-01",
  "message": "Layer imbalance detected",
  "severity": "error"
}
```

Returns `0` on success, non-zero on error.

## Reference Policies

Arx ships with three pre-built reference policies in `policies/`:

### layer-balance

Checks that each layer has a well-balanced number of dependencies relative to configured limits.

```yaml
rules:
  - id: P-01
    wasm:
      path: policies/layer-balance.wasm
      params: { min: 3, max: 8 }
    severity: warning
    explanation: Layer dependency count should be between 3 and 8
```

### dependency-symmetry

Flags asymmetric dependency patterns — when layer A depends on layer B but not vice versa beyond a tolerance threshold.

```yaml
rules:
  - id: P-02
    wasm:
      path: policies/dependency-symmetry.wasm
    severity: warning
    explanation: Dependencies between layers should be roughly symmetric
```

### no-leak

Flags layers that depend on other layers without having any violations — potential "hidden" dependencies that aren't caught by standard rules.

```yaml
rules:
  - id: P-03
    wasm:
      path: policies/no-leak.wasm
    severity: info
    explanation: Dependencies should be tracked by explicit rules
```

## Authoring with TinyGo

TinyGo is the recommended way to write policies. Here's a complete example:

```go
package main

import "unsafe"

//go:wasmimport arx arx_deps
func arxDeps() int32

//go:wasmimport arx arx_emit_violation
func arxEmitViolation(ptr unsafe.Pointer, len int32) int32

//go:export evaluate
func evaluate() int32 {
    deps := arxDeps()
    if deps > 100 {
        msg := `{"rule_id":"P-01","message":"Too many deps: " + string(deps) + "","severity":"warning"}`
        ptr := unsafe.Pointer(unsafe.StringData(msg))
        arxEmitViolation(ptr, int32(len(msg)))
    }
    return 0
}

func main() {}
```

### Building

```bash
tinygo build -o policies/my-policy.wasm -target=wasi policies/my-policy.go
```

Or use the provided `policies/build.sh` to build all reference policies:

```bash
make wasm-policies
```

This requires [TinyGo](https://tinygo.org/) to be installed. If TinyGo isn't available, pre-built `.wasm` stubs are used instead.

## Authoring in Other Languages

Any language that compiles to WASI works:

### Rust

```rust
#[link(wasm_import_module = "arx")]
extern "C" {
    fn arx_deps() -> i32;
    fn arx_emit_violation(ptr: *const u8, len: i32) -> i32;
}

#[no_mangle]
pub extern "C" fn evaluate() -> i32 {
    let deps = unsafe { arx_deps() };
    // ... policy logic ...
    0
}
```

Build: `cargo build --target wasm32-wasi --release`

### C

```c
__attribute__((import_module("arx"), import_name("arx_deps"))) 
int arx_deps(void);

__attribute__((import_module("arx"), import_name("arx_emit_violation"))) 
int arx_emit_violation(const char* ptr, int len);

__attribute__((export_name("evaluate")))
int evaluate(void) {
    // ... policy logic ...
    return 0;
}
```

Build: `clang --target=wasm32-wasi -O2 -o policy.wasm policy.c`

## Best Practices

1. **Keep policies focused** — one policy per concern, just like rules
2. **Use params for configurability** — pass `min`, `max`, `threshold` via `wasm.params`
3. **Handle errors gracefully** — return meaningful error codes from `evaluate()`
4. **Test with small projects first** — policies work on the full dep graph, so test incrementally
5. **Cache aggressively** — policies are cached by content hash; invalidate by changing the `.wasm` file

## Limitations

- Maximum WASM module size: **1 MB** (configurable)
- Timeout: **5 seconds** per policy evaluation (configurable)
- No WASM module marketplace (yet)
- No remote WASM execution
- Policies run with the same privileges as arx — treat them as trusted code
