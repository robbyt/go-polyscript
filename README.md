# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A unified abstraction package for loading and running various scripting languages and WASM in your Go app.

## Overview

go-polyscript democratizes different scripting engines by abstracting the loading, data handling, runtime, and results handling, allowing for interchangeability of scripting languages. This package provides interfaces and implementations for "engines", "executables", "evaluators" and the final "result". There are several tiers of public APIs, each with increasing complexity and configurability. `polyscript.go` in the root exposes the most common use cases, but is also the most opiniated.

## Features

- **Unified Abstraction API**: Common interfaces and implementations for several scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Thread-safe Data Management**: Multiple ways to provide input data to scripts
- **Compilation, Evaluation, and Data Handling**: Compile scripts once with static data when creating the evaluator instance, then run multiple evaluation executions with variable input.


## Engines Implemented

- **Risor**: A scripting language specifically designed for embedding in Go applications
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	script := `
		// The ctx object from the Go inputData map
		name := ctx.get("name")

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
		logger.Handler(),
	)
	
	ctx := context.Background()
	result, _ := evaluator.Eval(ctx)
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Working with Data Providers

To send input data to a script, use a "data provider" implementation. There are several built-in providers, or implement your own and stack multiple with the `CompositeProvider`.

### StaticProvider

For example, when working with Risor, the `FromRisorStringWithData` constructor function uses a `StaticProvider` to send the static data map into the evaluator during creation.

```go
inputData := map[string]any{"name": "cats", "excited": true}
evaluator, _ := polyscript.FromRisorStringWithData(script, inputData, logger.Handler())
```

However, when using `StaticProvider`, each evaluation will always use the same input data. Use the `ContextProvider` to provide dynamic runtime data that varies per evaluation.

### ContextProvider

The `ContextProvider` retrieves dynamic data from the context object sent to Eval.

```go
evaluator, _ := polyscript.FromRisorString(script, logger.Handler())

ctx := context.Background()
runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
enrichedCtx, _ := evaluator.AddDataToContext(ctx, runtimeData)

// Execute with the "enriched" context containing the link to the input data
result, _ := evaluator.Eval(enrichedCtx)
```

### Combining Static and Dynamic Runtime Data

Use the following pattern for fixed configuration values and threadsafe per-request data. Loading, parsing and instantiating the script is relatively slow, so the example below shows how to compile the script once with static data, then reuse it multiple times with different dynamic data.

```go
staticData := map[string]any{
    "appName": "MyApp",
    "version": "1.0",
}

// Create the evaluator with the static data
evaluator, _ := polyscript.FromRisorStringWithData(script, staticData, logger.Handler())

// For each request, prepare dynamic data
requestData := map[string]any{"userId": 123}
enrichedCtx, _ := evaluator.AddDataToContext(context.Background(), requestData)

// Execute with both static and dynamic data available
result, _ := evaluator.Eval(enrichedCtx)

// In Risor and Starlark scripts, data is accessed from the ctx variable:
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
    logger.Handler(),
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
    logger.Handler(),
    "greet",  // entryPoint
)

// Execute with a context
result, err := evaluator.Eval(context.Background())
```

## License

Apache License 2.0
