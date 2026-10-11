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

## Mock Plugins and the GC Cleanup

- `compiler.Executable` closes its plugin from a `runtime.AddCleanup` when garbage collected unclosed. A testify mock without a `Close` expectation then panics inside the cleanup and crashes the whole test binary, at random, often far from the leaking test.
- **Every mock plugin that is not explicitly closed needs** `mockPlugin.On("Close", mock.Anything).Return(nil).Maybe()`.
- **Verify with** `GOGC=1 go test -race -count=3 ./engines/extism/... .` after adding or changing Extism tests.
- See "Testing: Mock Plugins and the Garbage Collector" in `engines/README.md`.

---

## Build System

- **Main build**: `make test` runs `wasmdata-build` automatically
- **Manual WASM build**: `cd engines/extism/wasmdata && make build`
- **TinyGo source**: Located in `engines/extism/wasmdata/examples/` with separate go.mod