# go-polyscript Examples

This directory contains examples demonstrating the core capabilities of go-polyscript with different script engines.

## Directory Structure

The examples are organized by execution pattern and script engine:

```
examples/
├── simple/                   # Simple one-time execution examples
│   ├── risor/                # Risor simple example
│   ├── starlark/             # Starlark simple example
│   └── extism/               # Extism simple example
├── multiple/                 # Compile-once-run-many examples
│   ├── risor/                # Risor multiple execution example
│   ├── starlark/             # Starlark multiple execution example
│   └── extism/               # Extism multiple execution example
└── data-prep/                # Data preparation examples
    ├── risor/                # Risor with distributed data prep
    ├── starlark/             # Starlark with multi-step data prep
    └── extism/               # Extism with async data prep
```

## Execution Patterns

go-polyscript supports three primary execution patterns:

### 1. Simple Execution (Compile and Run Once)

The simplest pattern compiles and executes a script in a single operation:

- Script is compiled every time it runs
- All data is provided at compilation time through a `StaticProvider`
- Good for one-off script executions

**Examples:** [Risor](/examples/simple/risor), [Starlark](/examples/simple/starlark), [Extism](/examples/simple/extism)

### 2. Compile Once, Run Many Times

This advanced pattern separates compilation from execution, providing significant performance benefits:

- Script is compiled only once into an `ExecutableUnit`
- The same compiled script is executed multiple times with different data
- Data is provided at runtime through a `ContextProvider`
- Dramatically improves performance for multiple executions

**Examples:** [Risor](/examples/multiple/risor), [Starlark](/examples/multiple/starlark), [Extism](/examples/multiple/extism)

### 3. Distributed Data Preparation

This pattern separates data preparation from script evaluation:

- Prepares data on one system and evaluates on another
- Uses the `EvalDataPreparer` interface with the `PrepareContext` method
- Enables more flexible architecture and clearer separation of concerns
- Supports multi-step and asynchronous data preparation

**Examples:** [Risor](/examples/data-prep/risor), [Starlark](/examples/data-prep/starlark), [Extism](/examples/data-prep/extism)

## Script Engines

### Starlark

[Starlark](https://github.com/google/starlark-go) is a Python dialect designed for configuration and scripting within applications.

### Risor

[Risor](https://github.com/risor-io/risor) is a modern embedded scripting language for Go with a focus on simplicity and performance.

### Extism (WebAssembly)

[Extism](https://extism.org/) enables WebAssembly module execution within your Go application.

Note: The Extism examples require a WebAssembly module (provided in the `examples/simple/extism` directory).

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