# go-polyscript

A Go library providing a unified interface for embedding and running various scripting languages and WebAssembly modules in your Go applications.

## Overview

go-polyscript enables you to embed and execute scripts written in multiple languages within your Go applications. It offers a consistent API across different scripting engines, allowing for easy interchangeability and minimizing lock-in to a specific scripting language.

All components accept a structured logger (`slog.Handler`) for consistent logging across your application, making it easy to integrate with your existing logging infrastructure.

Currently supported scripting engines ("machines"):

- **Risor**: A simple scripting language specifically designed for embedding in Go applications
- **Starlark**: Google's configuration language (a Python dialect) used in Bazel and many other tools
- **Extism**: WebAssembly runtime for executing WASM modules as plugins

## Features

- **Unified API**: Common interfaces for all supported scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Powerful Context Passing**: Pass data from Go to scripts and vice versa
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

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/execution/script/loader"
	"github.com/robbyt/go-polyscript/machines/risor"
)

func main() {
	// Create a logger
	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)

	// Define globals that will be available to the script
	globals := []string{"ctx"}

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
	scriptLoader := loader.NewStringLoader(scriptContent)

	// Create an executable unit
	unit, err := script.NewExecutableUnit(handler, "", scriptLoader, compiler, nil)
	if err != nil {
		logger.Error("Failed to create executable unit", "error", err)
		return
	}

	// Create an evaluator for Risor scripts
	evaluator := risor.NewBytecodeEvaluator(handler)

	// Create context with input data
	ctx := context.Background()
	input := map[string]any{
		"name": "World",
	}
	ctx = context.WithValue(ctx, "eval_data", input)

	// Execute the script
	result, err := evaluator.Eval(ctx, unit)
	if err != nil {
		logger.Error("Script evaluation failed", "error", err)
		return
	}

	// Use the result
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Architecture

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (files, strings, etc.)
2. **Compiler**: Validates and compiles scripts into bytecode
3. **ExecutableUnit**: Represents a compiled script ready for execution
4. **Evaluator**: Executes compiled scripts with provided input data
5. **Machine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)

## Advanced Usage

### Using Starlark

```go
// Create a Starlark compiler
compiler := starlark.NewCompiler(handler, &starlark.BasicCompilerOptions{Globals: []string{"ctx"}})

// Rest of the code is similar to the Risor example
// Just use starlark.NewBytecodeEvaluator(handler) for evaluation
```

### Using WebAssembly with Extism

```go
// Create an Extism compiler
compiler := extism.NewCompiler(handler, &extism.BasicCompilerOptions{EntryPoint: "main"})

// Load a WASM module from file
wasmLoader := loader.NewFileLoader("/path/to/module.wasm")

// Rest of the code follows the same pattern
// Use extism.NewBytecodeEvaluator(handler) for evaluation
```

## License

Apache License 2.0