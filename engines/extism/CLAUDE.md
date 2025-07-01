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

## Build System

- **Main build**: `make test` runs `wasmdata-build` automatically
- **Manual WASM build**: `cd engines/extism/wasmdata && make build`
- **TinyGo source**: Located in `engines/extism/wasmdata/examples/` with separate go.mod