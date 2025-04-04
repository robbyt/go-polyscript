# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Go package providing a unified interface for loading and running various scripting languages and WebAssembly modules in your Go applications.

## Overview

go-polyscript provides a consistent API across different scripting engines, allowing for easy interchangeability and minimizing lock-in to a specific scripting language. This is achieved through a low-overhead abstraction of "machines," "executables," and the final "result". The input/output and runtime features provided by the scripting engines are standardized, which simplifies combining or swapping scripting engines in your application.

Currently supported scripting engines ("machines"):

- **Risor**: A simple scripting language specifically designed for embedding in Go applications
- **Starlark**: Google's configuration language (a Python dialect) used in Bazel and many other tools
- **Extism**: WebAssembly runtime and plugin system for executing WASM modules

## Features

- **Unified API**: Common interfaces for all supported scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Powerful Data Passing**: Multiple ways to provide input data to scripts
- **Comprehensive Logging**: Structured logging with `slog` support
- **Error Handling**: Robust error handling and reporting from script execution
- **Compilation and Evaluation Separation**: Compile once, run multiple times with different inputs
- **Data Preparation and Evaluation Separation**: Prepare data in one step/system, evaluate in another

## Installation

```bash
go get github.com/robbyt/go-polyscript@latest
```

## Quick Start

Here's a simple example of using go-polyscript with the Risor scripting engine:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/risor"
	"github.com/robbyt/go-polyscript/options"
)

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)

	// Script content
	scriptContent := `
		// Script has access to ctx variable passed from Go
		name := ctx["name"]
		message := "Hello, " + name + "!"
		
		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`
	
	// Input data
	inputData := map[string]any{"name": "World"}
	dataProvider := data.NewStaticProvider(inputData)
	
	// Create evaluator with functional options
	evaluator, err := polyscript.FromRisorString(
		scriptContent,
		options.WithDefaults(),
		options.WithLogger(handler),
		options.WithDataProvider(dataProvider),
		risor.WithGlobals([]string{"ctx"}),
	)
	if err != nil {
		logger.Error("Failed to create evaluator", "error", err)
		return
	}
	
	// Execute the script with a context
	ctx := context.Background()
	result, err := evaluator.Eval(ctx)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return
	}
	
	// Use the result
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Working with Data Providers

go-polyscript uses data providers to supply information to scripts during evaluation. Depending on your use case, you can choose from several built-in providers or combine them for more flexibility.

### StaticProvider

The `StaticProvider` supplies fixed data for all evaluations. This is ideal for scenarios where the input data remains constant across evaluations:

```go
// Create a static provider with predefined data
configData := map[string]any{"name": "World", "timeout": 30}
provider := data.NewStaticProvider(configData)

// Create evaluator with the provider
evaluator, err := polyscript.FromRisorString(script, options.WithDataProvider(provider))

// In scripts, static data is accessed directly:
// name := ctx["name"]  // "World"
```

However, when using `StaticProvider`, each evaluation will always use the same input data. If you need to provide dynamic runtime data that varies per evaluation, you can use the `ContextProvider`.

### ContextProvider

The `ContextProvider` retrieves dynamic data from the context and makes it available to scripts. This is useful for scenarios where input data changes at runtime:

```go
// Create a context provider for dynamic data
provider := data.NewContextProvider(constants.EvalData)

// Prepare context with runtime data
ctx := context.Background()
userData := map[string]any{"userId": 123, "preferences": {"theme": "dark"}}
enrichedCtx, _ := provider.AddDataToContext(ctx, userData)

// Create evaluator with the provider
evaluator, err := polyscript.FromRisorString(script, options.WithDataProvider(provider))

// In scripts, dynamic data is accessed via input_data:
// userId := ctx["input_data"]["userId"]  // 123
```

### CompositeProvider

To combine static data with dynamic runtime data, you can use the `CompositeProvider`. This allows you to stack a `StaticProvider` with a `ContextProvider`, enabling both fixed and variable data to be available during evaluation:

```go
// Create providers for static and dynamic data
staticProvider := data.NewStaticProvider(map[string]any{
    "appName": "MyApp",
    "version": "1.0",
})
ctxProvider := data.NewContextProvider(constants.EvalData)

// Combine them using CompositeProvider
provider := data.NewCompositeProvider(staticProvider, ctxProvider)

// Create evaluator with the composite provider
evaluator, err := polyscript.FromRisorString(script, options.WithDataProvider(provider))

// In scripts, data can be accessed from both locations:
// appName := ctx["appName"]  // Static data: "MyApp"
// userId := ctx["input_data"]["userId"]  // Dynamic data: 123
```

By using the `CompositeProvider`, you can ensure that your scripts have access to both constant configuration values and per-evaluation runtime data, making your evaluations more flexible and powerful.

## Architecture

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (files, strings, http, etc.)
2. **Compiler**: Validates and compiles scripts into internal "bytecode"
3. **ExecutableUnit**: Represents a compiled script ready for execution
4. **ExecutionPackage**: Contains an **ExecutableUnit** and other metadata
5. **Evaluator**: Executes compiled scripts with provided input data
6. **EvalDataPreparer**: Prepares data for evaluation (can be separated from evaluation)
7. **Provider**: Supplies data to scripts during evaluation
8. **Machine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)
9. **EvaluatorResponse**: The response object returned from all **Machine**s

### Note on Data Access Patterns

go-polyscript uses a unified `Provider` interface to supply data to scripts. The library has standardized on storing dynamic runtime data under the `input_data` key (previously `script_data`). For maximum compatibility, scripts should handle two data access patterns:

1. Top-level access for static data: `ctx["config_value"]`
2. Nested access for dynamic data: `ctx["input_data"]["user_data"]`
3. HTTP request data access: `ctx["input_data"]["request"]["method"]` (request objects are always stored under input_data)

See the [Data Providers](#working-with-data-providers) section for more details.

## Preparing Data Separately from Evaluation

go-polyscript provides the `EvalDataPreparer` interface to separate data preparation from script evaluation, which is useful for distributed architectures and multi-step data processing:

```go
// Create an evaluator (implements EvaluatorWithPrep interface)
evaluator, err := polyscript.FromRisorString(script, options...)
if err != nil {
    // handle error
}

// Prepare context with data (could happen on a web server)
requestData := map[string]any{"name": "World"}
enrichedCtx, err := evaluator.PrepareContext(ctx, requestData)
if err != nil {
    // handle error
}

// Later, or on a different system, evaluate with the prepared context
result, err := evaluator.Eval(enrichedCtx)
if err != nil {
    // handle error
}
```

For more detailed examples of this pattern, see the [data-prep examples](examples/data-prep/).

## Advanced Usage

### Using Starlark

```go
// Create a Starlark evaluator with options
evaluator, err := polyscript.FromStarlarkString(
    `
    # Starlark has access to ctx variable
    name = ctx["name"]
    message = "Hello, " + name + "!"
    
    # Create the result dictionary
    result = {"greeting": message, "length": len(message)}
    
    # Assign to _ to return the value
    _ = result
    `,
    options.WithDefaults(),
    options.WithDataProvider(data.NewStaticProvider(map[string]any{"name": "World"})),
    starlark.WithGlobals([]string{constants.Ctx}),
)

// Execute with a context
result, err := evaluator.Eval(context.Background())
```

### Using WebAssembly with Extism

```go
// Create an Extism evaluator
evaluator, err := polyscript.FromExtismFile(
    "/path/to/module.wasm",
    options.WithDefaults(),
    options.WithDataProvider(data.NewStaticProvider(map[string]any{"input": "World"})),
    extism.WithEntryPoint("greet"),
)

// Execute with a context
result, err := evaluator.Eval(context.Background())
```

## License

Apache License 2.0
