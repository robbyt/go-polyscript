# go-polyscript Examples

This directory contains examples demonstrating the core capabilities of go-polyscript with different script engines. The examples are organized by execution pattern and script engine.

## Execution Patterns

go-polyscript supports three primary execution patterns:

### 1. Simple Execution (One-Time Execution)

The simplest pattern provides both script and data at creation time:

- Uses `FromEngineWithData` functions to create an evaluator with static data
- Script and data are provided together at creation time
- Evaluator is created and executed once
- Suitable for one-off script executions with known data

**Examples:** [Risor](/examples/simple/risor), [Starlark](/examples/simple/starlark), [Extism](/examples/simple/extism)

### 2. Multiple Instantiation (Compile Once, Run Many Times)

This pattern separates compilation from execution for better performance:

- Uses `FromEngine` functions to create an evaluator without data
- Script is compiled once into an evaluator
- The same evaluator is executed multiple times with different runtime data
- Data is provided at runtime via context
- Improves performance for multiple executions of the same script

**Examples:** [Risor](/examples/multiple-instantiation/risor), [Starlark](/examples/multiple-instantiation/starlark), [Extism](/examples/multiple-instantiation/extism)

### 3. Data Preparation Pattern

This pattern separates data preparation from script evaluation:

- Uses `FromEngineWithData` functions with static configuration data
- Additional dynamic data is added to context before evaluation using `AddDataToContext`
- Combines static configuration with runtime variables
- Enables flexible data preparation workflows

**Examples:** [Risor](/examples/data-prep/risor), [Starlark](/examples/data-prep/starlark), [Extism](/examples/data-prep/extism)

## Script Engines

### Starlark

[Starlark](https://github.com/google/starlark-go) is a Python dialect designed for configuration and scripting within applications.

### Risor

[Risor](https://github.com/risor-io/risor) is a modern embedded scripting language for Go with a focus on simplicity and performance.

### Extism (WebAssembly)

[Extism](https://extism.org/) enables WebAssembly module execution within your Go application. The examples use an embedded test WebAssembly module for demonstration purposes.

## Key Components Across All Examples

### The `ctx` Global Variable

All scripts access data through the `ctx` global variable:

- **Risor & Starlark**: Access data as `ctx["key"]` 
- **Extism**: Data is automatically mapped to the WASM module's input
- This provides a consistent interface regardless of the underlying script engine

### Evaluators

Each example creates an evaluator using the appropriate function:

- **Static Data**: `FromEngineWithData(script, data, logger)` - data provided at creation
- **Dynamic Data**: `FromEngine(script, logger)` - data provided via context at runtime
- **Combined**: Use `AddDataToContext` to enrich context with additional data

### Data Flow Patterns

Examples demonstrate different data flow approaches:

- **Simple**: All data provided upfront when creating the evaluator
- **Multiple-instantiation**: Data provided dynamically via context for each execution
- **Data-prep**: Static configuration data at creation + dynamic data via context

### Result Processing

All examples show how to:
- Handle script execution results
- Validate returned data structures
- Convert between Go and script data types
- Process both successful results and errors