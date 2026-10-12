# Extism Engine Guidelines

Specific guidance for working with the Extism WebAssembly engine.

---

## WASM Module Management

- **ALWAYS use `wasmdata.TestModule`** for test WASM references
- **NO helper functions** - Use embedded module directly, avoid `readTestWasm()`, `getTestWasmBytes()`, or `FindWasmFile()` patterns
- **Use wasmdata constants** for entrypoints: `wasmdata.EntrypointGreet`, `wasmdata.EntrypointRun`, etc.
- **Build integration**: WASM builds are part of `make test` via `wasmdata-build` target

---

## Correct Test Patterns

```go
// Use embedded module directly
wasmBytes := wasmdata.TestModule
entrypoint := wasmdata.EntrypointGreet
```

---

## Mock Plugins and Lifetimes

- Nothing closes a `compiler.Executable` on garbage collection, but one bound to a lifetime (`WatchLifetime`, or an evaluator constructor's ctx) calls `Close` on its plugin when that ctx ends, e.g. `t.Context()` at test end. A testify mock without a `Close` expectation then panics in the watcher goroutine and crashes the test binary.
- **Every mock plugin on an executable whose lifetime may end during the test needs** `mockPlugin.On("Close", mock.Anything).Return(nil).Maybe()`, unless the test closes it explicitly with a matching expectation.
- See "Testing: Mock Plugins and Lifetimes" in `engines/README.md`.

---

## Build System

- **Main build**: `make test` runs `wasmdata-build` automatically
- **Manual WASM build**: `cd engines/extism/wasmdata && make build`
- **TinyGo source**: Located in `engines/extism/wasmdata/examples/` with separate go.mod