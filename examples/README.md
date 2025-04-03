# go-polyscript Examples

This directory contains examples demonstrating the core capabilities of go-polyscript with different script engines.

## Directory Structure

The examples are organized by execution pattern and script engine:

```
examples/
├── simple/                        # Simple one-time execution examples
│   ├── risor/                     # Risor simple example
│   ├── starlark/                  # Starlark simple example
│   └── extism/                    # Extism simple example
├── multiple-instantiation/        # Compile-once-run-many examples
│   ├── risor/                     # Risor multiple execution example
│   ├── starlark/                  # Starlark multiple execution example
│   └── extism/                    # Extism multiple execution example
└── data-prep/                     # Data preparation examples
    ├── risor/                     # Risor with data preparation
    ├── starlark/                  # Starlark with data preparation
    └── extism/                    # Extism with data preparation
```

## Execution Patterns

go-polyscript supports three primary execution patterns:

### 1. Simple Execution (Compile and Run Once)

The simplest pattern compiles and executes a script in a single operation:

- Script is compiled every time it runs
- All data is provided at compilation time through a `StaticProvider`
- Suitable for one-off script executions

**Examples:** [Risor](/examples/simple/risor), [Starlark](/examples/simple/starlark), [Extism](/examples/simple/extism)

### 2. Multiple Instantiation (Compile Once, Run Many Times)

This pattern separates compilation from execution for better performance:

- Script is compiled only once into an `ExecutableUnit`
- The same compiled script is executed multiple times with different data
- Data is provided at runtime through a `ContextProvider`
- Improves performance for multiple executions of the same script

**Examples:** [Risor](/examples/multiple-instantiation/risor), [Starlark](/examples/multiple-instantiation/starlark), [Extism](/examples/multiple-instantiation/extism)

### 3. Data Preparation Pattern

This pattern separates data preparation from script evaluation:

- Uses a combination of static and dynamic data providers
- Static data (configuration) is provided at compile time
- Dynamic data (runtime variables) is injected at execution time
- Enables flexibility in how data is prepared and passed to scripts

**Examples:** [Risor](/examples/data-prep/risor), [Starlark](/examples/data-prep/starlark), [Extism](/examples/data-prep/extism)

## Script Engines

### Starlark

[Starlark](https://github.com/google/starlark-go) is a Python dialect designed for configuration and scripting within applications.

### Risor

[Risor](https://github.com/risor-io/risor) is a modern embedded scripting language for Go with a focus on simplicity and performance.

### Extism (WebAssembly)

[Extism](https://extism.org/) enables WebAssembly module execution within your Go application.

Note: The Extism examples require a WebAssembly module (main.wasm).

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
- This maintains a clear separation between your application and embedded scripts

### Evaluators

Evaluators wrap a compiled script in a standardized interface:

- They handle script compilation and execution
- They execute the script with provided data
- They process script results
- Each script engine (Starlark, Risor, Extism) has its own evaluator implementation

### Result Handling

All examples demonstrate techniques for:
- Processing script outputs
- Validating returned data structures  
- Converting between Go and script data types
- Error handling and reporting