# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A unified abstraction package for loading and running various scripting languages and WASM modules in your Go app.

## Overview

go-polyscript democratizes different scripting engines by abstracting the loading, data handling, runtime, and results handling, allowing for interchangeability of scripting languages. The library provides interfaces and implementations for "engines", "executables", "evaluators", and the final "result".

The top-level package (`polyscript.go`) exposes a single generic constructor — `polyscript.New[E]` — that handles the common cases for all three engines. For finer-grained control (custom compilers, custom data providers, etc.), the per-engine sub-packages under `engines/` are also exported.

## Features

- **Unified Abstraction API**: Common interfaces and implementations for several scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Thread-safe Data Management**: Multiple ways to provide input data to scripts
- **Compilation, Evaluation, and Data Handling**: Compile scripts once with static data when creating the evaluator instance, then run multiple evaluation executions with variable runtime input.

## Engines Implemented

- **Risor**: A fast scripting language designed for embedding in Go applications
- **Starlark**: Google's deterministic configuration language (used in Bazel, and others)
- **Extism**: Pure Go runtime and plugin system for executing WASM

## Installation

```bash
go get github.com/robbyt/go-polyscript@latest
```

## Quick Start

Using go-polyscript with the Risor scripting engine:

```go
package main

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript"
)

func main() {
	script := `
		// The ctx object holds the input data map
		let name = ctx.get("name")

		let p = "."
		if (ctx.get("excited")) {
			p = "!"
		}

		let message = "Hello, " + name + p

		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`

	evaluator, _ := polyscript.New[polyscript.Risor](
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
	)

	result, _ := evaluator.Eval(context.Background())
	fmt.Printf("Result: %v\n", result.Interface())
}
```

The older `FromRisorString*`, `FromStarlark*`, and `FromExtism*` constructors still work but are deprecated and slated for removal in v1.

## API at a glance

`polyscript.New[E]` is the single entry point. `E` selects the engine, the first argument is a `Source`, and the remaining arguments are `Option[E]`s.

### Engines

| Type                  | Use it for                              |
| --------------------- | --------------------------------------- |
| `polyscript.Risor`    | Risor scripts                           |
| `polyscript.Starlark` | Starlark configuration scripts          |
| `polyscript.Extism`   | WebAssembly modules via the Extism PDK  |

### Sources

| Builder                            | Backed by                                                                |
| ---------------------------------- | ------------------------------------------------------------------------ |
| `polyscript.FromString(s)`         | An in-memory script string                                               |
| `polyscript.FromBytes(b)`          | An in-memory byte slice (typically WASM); the slice is copied            |
| `polyscript.FromFile(path)`        | An absolute path on disk                                                 |
| `polyscript.FromLoader(l)`         | Any custom `loader.Loader` (e.g. an HTTP loader; see [Loading Scripts](#loading-scripts)) |

### Options

| Option                             | Applies to              | Effect                                                              |
| ---------------------------------- | ----------------------- | ------------------------------------------------------------------- |
| `WithStaticData[E](map)`           | All engines             | Bakes a fixed data map into the evaluator at construction time      |
| `WithLogHandler[E](handler)`       | All engines             | Routes diagnostic logs through the given `slog.Handler`             |
| `WithEntryPoint(name)`             | `Extism` only (compile-time enforced) | Sets the WASM function name to invoke; required for Extism |

> **Note on type arguments.** `WithStaticData` and `WithLogHandler` are generic over the engine. Go's current type inference can't always infer `E` for them when the surrounding `New[E]` call has a non-variadic `Source` parameter, so these helpers usually need an explicit type argument: `polyscript.WithStaticData[polyscript.Risor](data)`. `WithEntryPoint` is bound to `Extism` and never needs one — passing it to `New[polyscript.Risor]` or `New[polyscript.Starlark]` is a compile error rather than a silent no-op.

## Loading Scripts

`FromString` is the easiest source for embedded scripts. For other locations:

### From a file on disk

```go
evaluator, _ := polyscript.New[polyscript.Risor](
	polyscript.FromFile("/etc/polyscript/hello.risor"),
)
```

The path must be absolute. For relative paths, resolve them with `filepath.Abs` first.

### From an HTTP endpoint

`FromLoader` wraps any `loader.Loader`, including the HTTP loader that ships with go-polyscript:

```go
import (
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

httpOpts := loader.DefaultHTTPOptions().
	WithTimeout(10 * time.Second).
	WithBearerAuth("my-api-token")

httpLoader, err := loader.NewFromHTTPWithOptions(
	"https://scripts.example.com/greet.risor",
	httpOpts,
)
if err != nil {
	return err
}

evaluator, err := polyscript.New[polyscript.Risor](polyscript.FromLoader(httpLoader))
```

### Capturing diagnostic logs

```go
import (
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
)

handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

evaluator, _ := polyscript.New[polyscript.Risor](
	polyscript.FromString(script),
	polyscript.WithLogHandler[polyscript.Risor](handler),
	polyscript.WithStaticData[polyscript.Risor](data),
)
```

## Working with Data Providers

To send input data to a script, use a "data provider" implementation. There are several built-in providers, or implement your own and stack multiple with the `CompositeProvider`.

### StaticProvider

For example, attaching `WithStaticData` to a Risor evaluator wires up a `StaticProvider` internally to send the static data map into the evaluator during creation.

```go
evaluator, _ := polyscript.New[polyscript.Risor](
	polyscript.FromString(script),
	polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "cats", "excited": true}),
)
```

### ContextProvider

A constructor created without `WithStaticData` uses a `ContextProvider`, so dynamic per-request data can be threaded in through the context.

```go
evaluator, _ := polyscript.New[polyscript.Risor](polyscript.FromString(script))

runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
enrichedCtx, _ := evaluator.AddDataToContext(context.Background(), runtimeData)

// Execute with the "enriched" context containing the link to the input data
result, _ := evaluator.Eval(enrichedCtx)
```

### Combining Static and Dynamic Runtime Data

Use the following pattern for fixed configuration values and per-request data. Initial loading, parsing, and instantiating the script is relatively slow, so the example below shows how to set up the script once with static data and then reuse it many times with dynamic runtime data.

```go
staticData := map[string]any{
	"name": "User",
	"excited": true,
}

// Create the evaluator with the static data
evaluator, _ := polyscript.New[polyscript.Risor](
	polyscript.FromString(script),
	polyscript.WithStaticData[polyscript.Risor](staticData),
)

// For each request, prepare dynamic data
requestData := map[string]any{"name": "Robert"}
enrichedCtx, _ := evaluator.AddDataToContext(context.Background(), requestData)

// Execute with both static and dynamic data available
result, _ := evaluator.Eval(enrichedCtx)
```

## Architectural Design

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (disk, `io.Reader`, strings, http, etc.)
2. **Compiler**: Validates and compiles scripts into internal "bytecode"
3. **ExecutableUnit**: Compiled script bundle, ready for execution
4. **Engine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)
5. **Evaluator**: Executes compiled scripts with provided input data
6. **DataProvider**: Sends data to the engine prior to evaluation
7. **EvaluatorResponse**: The response object returned from all **Engine**s

### Note on Data Access Patterns

go-polyscript uses a two-layer approach for handling data:

1. **Data Provider Layer**: The `Provider` interface (via `AddDataToContext`) handles storage mechanisms and general type conversions. This layer is pluggable, allowing data to be stored in various backends while maintaining a consistent API.

2. **Engine-Specific Layer**: Each engine's `Evaluator` implementation handles the engine-specific conversions between the stored data and the format required by that particular scripting engine.

This separation allows scripts to access data with consistent patterns regardless of the storage mechanism or script engine. For example, data you store with `{"config": value}` will be accessible in your scripts as `ctx["config"]`, with each engine handling the specific conversions needed for its runtime.

See the [Data Providers](#working-with-data-providers) section for more details.

## Working with other Engines

### Starlark
Starlark syntax is a deterministic "python-like" language designed for complex configuration, not so much for dynamic scripting. It's high performance, but the capabilities of the language are very limited. Read more about it here: [Starlark-Go](https://github.com/google/starlark-go)

```go
scriptContent := `
# Starlark has access to ctx variable
name = ctx["name"]
message = "Hello, " + name + "!"

# Create the result dictionary
result = {"greeting": message, "length": len(message)}

# Assign to _ to return the value
_ = result
`

evaluator, _ := polyscript.New[polyscript.Starlark](
	polyscript.FromString(scriptContent),
	polyscript.WithStaticData[polyscript.Starlark](map[string]any{"name": "World"}),
)

// Execute with a context
result, _ := evaluator.Eval(context.Background())
```

### WASM with Extism

Extism uses the Wazero WASM runtime for providing WASI abstractions, and an easy input/output memory sharing data system. Read more about writing WASM plugins for the Extism/Wazero runtime using the Extism PDK here: [extism.org](https://extism.org/docs/concepts/pdk)

```go
import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
)

func main() {
	evaluator, _ := polyscript.New[polyscript.Extism](
		// pre-compiled WASM example module
		polyscript.FromBytes(wasmdata.TestModule),

		// main entrypoint function in the WASM module
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),

		// the go-polyscript Extism engine will encode the static data into
		// JSON and send it to the WASM application
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)

	// Execute, and print the result
	result, _ := evaluator.Eval(context.Background())
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Direct engine subpackage usage

Most callers should use `polyscript.New[E]`. For finer-grained control over a single engine — for example, plugging in a custom `data.Provider` — each engine subpackage exposes the same shape: `FromXxxLoader(ldr, opts...)`, with engine-scoped `WithLogHandler`, `WithStaticData`, `WithDataProvider`, and (Extism only) `WithEntryPoint` options.

```go
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

ldr, _ := loader.NewFromString(`{"name": ctx["name"]}`)

eval, _ := risor.FromRisorLoader(
	ldr,
	risor.WithStaticData(map[string]any{"name": "World"}),
)

result, _ := eval.Eval(context.Background())
```

Logging is optional: omit `WithLogHandler` and the engine inherits whatever `slog.Default()` has been configured to in the host process. Extism is the same shape but expects a WASM module loader and requires `WithEntryPoint`:

```go
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

wasmLdr, _ := loader.NewFromBytes(wasmBytes) // wasmBytes is a compiled WASM module

eval, _ := extism.FromExtismLoader(
	wasmLdr,
	extism.WithEntryPoint("greet"),
	extism.WithStaticData(map[string]any{"input": "World"}),
)

result, _ := eval.Eval(context.Background())
```

## License

Apache License 2.0
