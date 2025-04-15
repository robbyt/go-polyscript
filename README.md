# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Go package providing a unified interface for loading and running various scripting languages and WASM in your app.

## Overview

go-polyscript democratizes different scripting engines by abstracting the loading, data handling, runtime, and results handling, allowing for interchangeability of scripting languages. This package provides interfaces and implementations for "engines", "executables", "evaluators" and the final "result". There are several tiers of public APIs, each with increasing complexity and configurability. `polyscript.go` in the root exposes the most common use cases, but is also the most opiniated.

## Features

- **Unified API**: Common interfaces and implementations for several scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Thread-safe Data Management**: Multiple ways to provide input data to scripts
- **Compilation and Evaluation Separation**: Compile once, run multiple times with different inputs
- **Data Preparation and Evaluation Separation**: Prepare data in one step/system, evaluate in another

## Engines Implemented

- **Risor**: A simple scripting language specifically designed for embedding in Go applications
- **Starlark**: Google's configuration language (a Python dialect) used in Bazel and many other tools
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
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
)

func main() {
	logHandler := slog.NewTextHandler(os.Stdout, nil)

	script := `
		// The ctx object from the Go inputData map
		name := ctx.get("name", "Roberto")

		p := "."
		if ctx.get("excited") {
			p = "!"
		}
		
		message := "Hello, " + name + p
		
		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`

	inputData := map[string]any{"name": "World"}
	
	evaluator, _ := polyscript.FromRisorStringWithData(
		script,
		inputData,
		logHandler,
	)
	
	ctx := context.Background()
	result, _ := evaluator.Eval(ctx)
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Working with Data Providers

go-polyscript enables you to send input data using a system called "data providers". There are several built-in providers, and you can implement your own or stack multiple with the `CompositeProvider`.

### StaticProvider

The `FromRisorStringWithData` function uses a `StaticProvider` to send the static data map.

```go
inputData := map[string]any{"name": "cats", "excited": true}
evaluator, _ := polyscript.FromRisorStringWithData(script, inputData, logHandler)
```

However, when using `StaticProvider`, each evaluation will always use the same input data. If you need to provide dynamic runtime data that varies per evaluation, you can use the `ContextProvider`.

### ContextProvider

The `ContextProvider` retrieves dynamic data from the context object sent to Eval. This is useful when input data changes at runtime:

```go
evaluator, _ := polyscript.FromRisorString(script, logHandler)

ctx := context.Background()
runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
enrichedCtx, _ := evaluator.AddDataToContext(ctx, runtimeData)

// Execute with the "enriched" context containing the link to the input data
result, _ := evaluator.Eval(enrichedCtx)
```

### Combining Static and Dynamic Runtime Data

This is a common pattern where you want both fixed configuration values and threadsafe per-request data to be available during evaluation:

```go
staticData := map[string]any{
    "appName": "MyApp",
    "version": "1.0",
}

// Create the evaluator with the static data
evaluator, _ := polyscript.FromRisorStringWithData(script, staticData, logHandler)

// For each request, prepare dynamic data
requestData := map[string]any{"userId": 123}
enrichedCtx, _ := evaluator.AddDataToContext(context.Background(), requestData)

// Execute with both static and dynamic data available
result, _ := evaluator.Eval(enrichedCtx)

// In scripts, all data is accessed from the context:
// appName := ctx["appName"]    // From static data: "MyApp"
// userId := ctx["userId"]      // From dynamic data: 123
```

## Architecture

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (disk, `io.Reader`, strings, http, etc.)
2. **Compiler**: Validates and compiles scripts into internal "bytecode"
3. **ExecutableUnit**: Compiled script bundle, ready for execution
4. **Engine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)
5. **Evaluator**: Executes compiled scripts with provided input data
6. **DataProvider**: Sends data to the VM prior to evaluation
7. **EvaluatorResponse**: The response object returned from all **Engine**s

### Note on Data Access Patterns

go-polyscript uses a two-layer approach for handling data:

1. **Data Provider Layer**: The `Provider` interface (via `AddDataToContext`) handles storage mechanisms and general type conversions. This layer is pluggable, allowing data to be stored in various backends while maintaining a consistent API.

2. **Engine-Specific Layer**: Each engine's `Evaluator` implementation handles the engine-specific conversions between the stored data and the format required by that particular scripting engine.

This separation allows scripts to access data with consistent patterns regardless of the storage mechanism or script engine. For example, data you store with `{"config": value}` will be accessible in your scripts as `ctx["config"]`, with each engine handling the specific conversions needed for its runtime.

See the [Data Providers](#working-with-data-providers) section for more details.

## Other Engines

### Starlark
Starlark syntax is a deterministic "python like" language designed for complex configuration, not so much for dynamic scripting. It's high performance, but the capabilities of the language are very limited. Read more about it here: [Starlark-Go](https://github.com/google/starlark-go)

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

staticData := map[string]any{"name": "World"}
evaluator, err := polyscript.FromStarlarkStringWithData(
    scriptContent,
    staticData,
    logHandler,
)

// Execute with a context
result, err := evaluator.Eval(context.Background())
```

### WASM with Extism

Extism uses the Wazero WASM runtime for providing WASI abstractions, and an easy input/output memory sharing data system. Read more about writing WASM plugins for the Extism/Wazero runtime using the Extism PDK here: [extism.org](https://extism.org/docs/concepts/pdk)

```go
// Create an Extism evaluator with static data
staticData := map[string]any{"input": "World"}
evaluator, err := polyscript.FromExtismFileWithData(
    "/path/to/module.wasm",
    staticData,
    logHandler,
    "greet",  // entryPoint
)

// Execute with a context
result, err := evaluator.Eval(context.Background())
```

## License

Apache License 2.0
