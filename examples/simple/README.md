# Simple Execution Examples

This directory contains examples of the basic "compile and run once" pattern for script execution with go-polyscript.

## Overview

The simple execution pattern is the most straightforward way to run embedded scripts. It follows these steps:

1. Prepare script data
2. Configure script options
3. Compile and immediately execute the script in a single operation
4. Process the results

## When to Use This Pattern

This pattern is ideal for:

- One-off script executions
- Simple automation tasks
- Configuration processing
- Situations where script reuse is not needed
- When performance is not critical

## Implementation Details

These examples use the following components:

- `StaticProvider`: Provides predefined data to the script at compile time
- Script data is accessed via the `ctx` global variable within scripts
- Each subdirectory contains examples for different script engines

## Running the Examples

To run any example:

```bash
cd examples/simple/<engine>
go run main.go
```

Note: The Extism example requires a WebAssembly module (provided as `main.wasm` in the `extism` directory).

## Next Steps

For more advanced usage patterns, see:

- [Multiple Execution Examples](/examples/multiple): The "compile once, run many times" pattern for better performance
- [Data Preparation Examples](/examples/data-prep): Separating data preparation from script evaluation