# Multiple Extism Example

This example demonstrates how to use the go-polyscript library with Extism WASM modules in a "compile once, run many times" pattern.

## Overview

The multiple example shows how to:

1. Compile an Extism WASM module once
2. Execute the compiled script multiple times with different input data
3. Process the results from each execution

This approach is more efficient when you need to run the same script multiple times with different inputs, as it avoids recompiling the WASM module for each execution.

## Key Components

- **ContextProvider**: Retrieves data dynamically from the Go context at runtime
- **polyscript.ExecutableUnit**: A compiled script that can be executed multiple times
- **constants.EvalData**: Context key used to retrieve data from the context

## Usage

```go
// Compile phase
dataProvider := data.NewContextProvider(constants.EvalData)
compiler, err := polyscript.FromExtismFile(
    wasmFilePath,
    options.WithLogger(handler),
    options.WithDataProvider(dataProvider),
    extism.WithEntryPoint("greet"),
)
execUnit, err := compiler.Compile()

// Execution phase (can be done multiple times)
inputData := map[string]any{
    "input": "World",
}
runCtx := context.WithValue(ctx, constants.EvalData, inputData)
response, err := execUnit.Eval(runCtx)
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