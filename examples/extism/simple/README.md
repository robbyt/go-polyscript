# Simple Extism Example

This example demonstrates how to use the go-polyscript library with Extism WASM modules in a simple one-off execution pattern.

## Overview

The simple example shows how to:

1. Load an Extism WASM module from file
2. Configure the evaluator with static input data
3. Execute the script once
4. Process the result

This approach is suitable for cases where you need to run a script only once with a fixed set of inputs.

## Key Components

- **StaticProvider**: Provides fixed data known at compile time
- **polyscript.FromExtismFile**: Creates an evaluator from a WASM file
- **WithEntryPoint**: Specifies which exported function to call in the WASM module

## Usage

```go
// Create input data
inputData := map[string]any{
    "input": "World",
}
dataProvider := data.NewStaticProvider(inputData)

// Create evaluator using the functional options pattern
evaluator, err := polyscript.FromExtismFile(
    wasmFilePath, 
    options.WithLogger(handler),
    options.WithDataProvider(dataProvider),
    extism.WithEntryPoint("greet"),
)

// Evaluate the script
response, err := evaluator.Eval(ctx)
```

## Running the Example

```bash
go run main.go
```

## Testing the Example

```bash
go test
```

Note: This example requires the WASM file to be present in one of the searched locations.