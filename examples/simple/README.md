# Simple Execution Examples

This directory contains examples demonstrating the basic "compile and execute once" pattern for script execution in go-polyscript.

## Overview

The simple execution pattern is the most straightforward way to run embedded scripts with go-polyscript. It creates an evaluator and immediately executes the script in a single operation, providing a clean, concise approach for one-off script executions.

## When to Use This Pattern

This pattern is ideal for:

- **Single-Use Scripts**: Scripts that will only be executed once
- **Configuration Processing**: Loading and validating configuration files
- **Simple Automation Tasks**: One-time data transformations or validations
- **Prototype Development**: Quick implementations for testing concepts
- **Isolated Operations**: When script execution is independent of other operations

## Implementation Pattern

These examples follow a consistent pattern:

1. **Create the input data**: Build a map with values the script needs
2. **Configure the data provider**: Use `data.NewStaticProvider(input)` to make data available to the script
3. **Configure the evaluator**: Set up with options like `WithGlobals` to expose the `ctx` variable
4. **Execute immediately**: Call `evaluator.Eval(ctx)` to run the script
5. **Process results**: Convert the response to the expected type with `result.Interface()`

## Script Access Pattern

In all examples, scripts access data using a `ctx` global variable:
- Starlark: `name = ctx["name"]`
- Risor: `name := ctx["name"]`
- Extism: Input data is available directly to the WASM module

## Running the Examples

Each example follows the same pattern but demonstrates it with a different script engine:

```bash
go run examples/simple/<engine>/main.go
```

Note: The Extism example requires a WebAssembly module. It uses the `FindWasmFile` function to locate the module in various directories.

## Related Patterns

- [Multiple Instantiation Examples](/examples/multiple-instantiation): Compile once, run many times pattern
- [Data Preparation Examples](/examples/data-prep): Separating static configuration from dynamic runtime data