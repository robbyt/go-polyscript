# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A unified abstraction package for loading and running various scripting languages and WASM modules in your Go app.

## Overview

go-polyscript democratizes different scripting engines by abstracting the loading, data handling, runtime, and results handling, allowing for interchangeability of scripting languages. This package provides interfaces and implementations for "engines", "executables", "evaluators" and the final "result". There are several tiers of public APIs, each with increasing complexity and configurability. `polyscript.go` in the root exposes the most common use cases, but is also the most opinionated.

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

The top-level API is `polyscript.New[E]` where `E` selects the engine — `polyscript.Risor`, `polyscript.Starlark`, or `polyscript.Extism`. The constructor takes a `Source` (built with `FromString`, `FromBytes`, `FromFile`, or `FromLoader`) and zero or more `Option[E]`s. Engine-specific options like `WithEntryPoint` are bound to a single engine at compile time, so passing them to the wrong engine is a compile error rather than a silent no-op.

> **Note on type arguments.** `WithStaticData` and `WithLogHandler` are generic helpers that work for any engine. Go's current type inference can't always infer `E` for them when the surrounding `New[E]` call has a non-variadic `Source` parameter, so these helpers usually need an explicit type argument: `polyscript.WithStaticData[polyscript.Risor](data)`. `WithEntryPoint` is bound to `Extism` and never needs one.

The older `FromRisorString*`, `FromStarlark*`, and `FromExtism*` constructors still work but are deprecated and slated for removal in v1.

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

## License

Apache License 2.0
