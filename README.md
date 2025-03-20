# go-polyscript

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A Go package providing a unified interface for loading and running various scripting languages and WebAssembly modules in your Go applications.

## Overview

go-polyscript enables a consistent API across different scripting engines, allowing for easy interchangeability and minimizing lock-in to a specific scripting language.

Currently supported scripting engines ("machines"):

- **Risor**: A simple scripting language specifically designed for embedding in Go applications
- **Starlark**: Google's configuration language (a Python dialect) used in Bazel and many other tools
- **Extism**: WebAssembly runtime for executing WASM modules as plugins

## Features

- **Unified API**: Common interfaces for all supported scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Powerful Data Passing**: Multiple ways to provide input data to scripts
- **Comprehensive Logging**: Structured logging with `slog` support
- **Error Handling**: Robust error handling and reporting from script execution
- **Compilation and Evaluation Separation**: Compile once, run multiple times with different inputs

## Installation

```bash
go get github.com/robbyt/go-polyscript
```

## Quick Start

Here's a simple example of using go-polyscript with the Risor engine:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/risor"
)

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{constants.Ctx}

	// Create a compiler for Risor scripts
	compilerOptions := &risor.BasicCompilerOptions{Globals: globals}
	compiler := risor.NewCompiler(handler, compilerOptions)

	// Load script from a string
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
	fromString, err := loader.NewFromString(scriptContent)
	if err != nil {
		logger.Error("Failed to create string loader", "error", err)
		return
	}

	// Create an executable unit
	unit, err := script.NewExecutableUnit(handler, "", fromString, compiler, nil)
	if err != nil {
		logger.Error("Failed to create executable unit", "error", err)
		return
	}

	// Create input data provider
	inputData := map[string]any{
		"name": "World",
	}
	dataProvider := data.NewStaticProvider(inputData)

	// Create an evaluator for Risor scripts
	evaluator := risor.NewBytecodeEvaluator(handler, dataProvider)

	// Execute the script with a context
	ctx := context.Background()
	result, err := evaluator.Eval(ctx, unit)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return
	}

	// Use the result
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Working with InputDataProvider

go-polyscript uses the `InputDataProvider` interface to supply data to scripts during evaluation. Several built-in providers are available:

### ContextProvider (Backward Compatibility)

Uses a context value to retrieve script input data:

```go
// Create a context provider (this is used internally by default)
provider := data.NewContextProvider(constants.EvalData)

// Prepare context with input data
ctx := context.Background()
input := map[string]any{"name": "World"}
ctx = context.WithValue(ctx, constants.EvalData, input)

// Create evaluator with the provider
evaluator := risor.NewBytecodeEvaluator(handler, provider)
```

### StaticProvider

Provides fixed data for all evaluations:

```go
// Create a static provider with predefined data
inputData := map[string]any{"name": "World"}
provider := data.NewStaticProvider(inputData)

// Create evaluator with the provider
evaluator := risor.NewBytecodeEvaluator(handler, provider)
```

### CompositeProvider

Combines multiple providers, checking each in sequence:

```go
// Create providers for different data sources
ctxProvider := data.NewContextProvider(constants.EvalData)
defaultProvider := data.NewStaticProvider(map[string]any{"defaultKey": "value"})

// Create a composite provider that tries ctxProvider first, then defaultProvider
provider := data.NewCompositeProvider(ctxProvider, defaultProvider)

// Create evaluator with the provider
evaluator := risor.NewBytecodeEvaluator(handler, provider)
```

## Architecture

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (files, strings, http, etc.)
2. **Compiler**: Validates and compiles scripts into internal "bytecode"
3. **ExecutableUnit**: Represents a compiled script ready for execution
4. **ExecutionPackage** Contains an **ExecutableUnit** and other metadata
5. **Evaluator**: Executes compiled scripts with provided input data
6. **InputDataProvider**: Supplies data to scripts during evaluation
7. **Machine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)
8. **EvaluatorResponse** The response object returned from all **Machine**s

## Advanced Usage

### Using Starlark

```go
// Create a Starlark compiler
compiler := starlark.NewCompiler(handler, &starlark.BasicCompilerOptions{Globals: []string{constants.Ctx}})

// Load a Starlark script
content := `
    # Starlark has access to ctx variable
    name = ctx["name"]
    message = "Hello, " + name + "!"
    
    # Return a dict with our result
    {"greeting": message, "length": len(message)}
`
fromString, _ := loader.NewFromString(content)

// Create executable unit
unit, _ := script.NewExecutableUnit(handler, "", fromString, compiler, nil)

// Create data provider and evaluator
dataProvider := data.NewStaticProvider(map[string]any{"name": "World"})
evaluator := starlark.NewBytecodeEvaluator(handler, dataProvider)

// Execute with a context
result, _ := evaluator.Eval(context.Background(), unit)
```

### Using WebAssembly with Extism

```go
// Create an Extism compiler
compiler := extism.NewCompiler(handler, &extism.BasicCompilerOptions{EntryPoint: "main"})

// Load a WASM module from file
fileLoader, _ := loader.NewFromDisk("/path/to/module.wasm")

// Create executable unit
unit, _ := script.NewExecutableUnit(handler, "", fileLoader, compiler, nil)

// Create data provider and evaluator
dataProvider := data.NewStaticProvider(map[string]any{"name": "World"})
evaluator := extism.NewBytecodeEvaluator(handler, dataProvider)

// Execute with a context
result, _ := evaluator.Eval(context.Background(), unit)
```

## License

Apache License 2.0