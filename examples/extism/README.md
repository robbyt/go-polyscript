# Extism Examples

This directory contains examples of using the go-polyscript library with Extism WASM modules.

## Overview

Extism is a WebAssembly (WASM) plugin system that lets you execute WASM modules from various languages. The go-polyscript library integrates with Extism to provide a consistent interface for executing WASM scripts.

Unlike other script engines, Extism/WASM requires loading compiled binary files (.wasm) rather than plain text scripts.

## Examples

There are two main patterns demonstrated:

1. [Simple](./simple/): One-off execution pattern - compiles and runs a script once
2. [Multiple](./multiple/): Compile once, run many times pattern - more efficient for repeated executions

## WASM File

Both examples use the same WASM file, which contains a simple `greet` function that takes a string input and returns a greeting. The WASM file is compiled from Go code using TinyGo.

### Building the WASM File

To build the WASM file from the source code in `machines/extism/testdata/examples/main.go`, you need TinyGo installed. Then run:

```bash
cd machines/extism/testdata
make build
```

This will generate `examples/main.wasm`, which is used by the examples.

### WASM Functions

The WASM module exports several functions, including:

- `greet`: Takes an input string and returns a greeting (e.g., "Hello, World!")
- `count_vowels`: Counts the vowels in an input string
- `reverse_string`: Reverses an input string
- `process_complex`: Processes a complex input object

## Common Components

Both examples use similar components:

- **FindWasmFile**: Helper function to locate the WASM file in various locations
- **polyscript.FromExtismFile**: Creates an evaluator from a WASM file
- **extism.WithEntryPoint**: Specifies which exported function to call in the WASM module

## Key Differences

- The **simple** example uses a **StaticProvider** with fixed input data
- The **multiple** example uses a **ContextProvider** to retrieve data dynamically from the context at runtime