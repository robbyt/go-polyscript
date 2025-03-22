# Data Preparation Examples

This directory contains examples demonstrating the `EvalDataPreparer` interface for separating data preparation from script evaluation in go-polyscript.

## Overview

The data preparation pattern enables a more flexible architecture by allowing data preparation to occur independently from script evaluation. This pattern introduces:

- The `EvalDataPreparer` interface with the `PrepareContext` method
- The `EvaluatorWithPrep` combined interface

## When to Use This Pattern

This pattern is valuable for:

1. **Distributed Architecture**: Prepare data on one system (e.g., web server) and evaluate on another (e.g., worker)
2. **Separation of Concerns**: Clearly separate data processing from script execution
3. **Multi-step Preparation**: Enrich a context across multiple services or systems
4. **Performance Optimization**: Prepare data asynchronously while other operations are happening

## Implementation Details

### Interface Overview

```go
// EvalDataPreparer is an interface for preparing data before evaluation
type EvalDataPreparer interface {
    // PrepareContext takes a context and variadic data arguments and returns an enriched context
    PrepareContext(ctx context.Context, data ...any) (context.Context, error)
}

// EvaluatorWithPrep combines Evaluator and EvalDataPreparer interfaces
type EvaluatorWithPrep interface {
    Evaluator
    EvalDataPreparer
}
```

All factory functions now return implementations of the `EvaluatorWithPrep` interface.

### Example Patterns

Each subdirectory demonstrates a different usage pattern:

- **risor/**: Simple distributed architecture pattern
- **starlark/**: Multi-step data preparation
- **extism/**: Asynchronous data preparation

### Key Concepts

The `PrepareContext` method handles:
- Converting raw Go types to machine-specific formats
- Storing data in the context using the data provider
- Error handling and validation

## Running the Examples

To run any example:

```bash
cd examples/data-prep/<engine>
go run main.go
```

Note: The Extism example requires a WASM file. If not found automatically, you may need to compile it using the `Makefile` in `/machines/extism/testdata/`.