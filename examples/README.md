# go-polyscript Examples

This directory contains examples demonstrating the core capabilities of go-polyscript with different script engines.

## Understanding Execution Patterns

go-polyscript supports two primary execution patterns:

### 1. Simple Execution (Compile and Run Once)

The simplest pattern compiles and executes a script in a single operation. This approach is straightforward but less efficient for repeated executions:

- Script is compiled every time it runs
- All data is provided at compilation time through a `StaticProvider`
- Good for one-off script executions

### 2. Compile Once, Run Many Times

This advanced pattern separates compilation from execution, providing significant performance benefits:

- Script is compiled only once into an `ExecutableUnit`
- The same compiled script is executed multiple times with different data
- Data is provided at runtime through a `ContextProvider`
- Dramatically improves performance for multiple executions
- Essential for high-throughput applications

## Script Engine Examples

### Starlark

[Starlark](https://github.com/google/starlark-go) is a Python dialect designed for configuration and scripting within applications.

**Available examples:**

- [Simple Execution](/examples/starlark/simple): Basic one-time script execution
- [Multiple Executions](/examples/starlark/multiple): Efficient "compile once, run many times" pattern 

### Risor

[Risor](https://github.com/risor-io/risor) is a modern embedded scripting language for Go with a focus on simplicity and performance.

```bash
cd examples/risor
go run main.go
```

### Extism (WebAssembly)

[Extism](https://extism.org/) enables WebAssembly module execution within your Go application.

```bash
cd examples/extism
go run main.go
```

Note: The Extism example requires a WebAssembly module (provided in the examples/extism directory).

## Key Components Across All Examples

All examples demonstrate these core go-polyscript components:

### Data Providers

Data providers control how data flows from your Go application into scripts:

- `StaticProvider`: Provides fixed data known at compile time
- `ContextProvider`: Retrieves data dynamically from the Go context at runtime
- `CompositeProvider`: Combines multiple providers for complex data flows

### The `ctx` Global

The `ctx` global variable serves as a bridge between Go and the script:

- In Go: You populate data in a provider
- In Script: The script accesses that data through the `ctx` variable
- This maintains a clean separation between your application and embedded scripts

### Evaluators

Evaluators wrap a compiled script in a standardized interface:

- They handle script compilation
- They execute the script with provided data
- They process and validate script results
- Each script engine (Starlark, Risor, Extism) has its own evaluator implementation

### Result Handling

All examples demonstrate proper techniques for:
- Processing script outputs
- Validating returned data structures  
- Converting between Go and script data types
- Error handling and reporting