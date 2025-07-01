# Multiple Instantiation Examples

This directory contains examples demonstrating the "compile once, run many times" pattern for efficient script execution in go-polyscript.

## Overview

This pattern compiles a script once and executes it multiple times with different inputs. The evaluator instance is reused for each execution, avoiding the overhead of repeated compilation and initialization.

## When to Use This Pattern

This pattern is suitable for:

- High-throughput applications processing many requests with the same script logic
- Variable input processing executing the same logic against different data sets 
- Resource-constrained environments where minimizing memory usage and CPU overhead is important
- Latency-sensitive systems where reducing processing time is critical

## Performance Benefits

This approach provides performance improvements:

- Reduces script compilation overhead by reusing the compiled script
- Avoids repeated parsing and validation of the script
- Reduces memory allocation needs
- Improves CPU cache utilization

## Implementation Pattern

These examples follow a consistent pattern:

1. **Create an evaluator once**: 
   - Initialize with script content and a context provider
   - The context provider enables passing different data on each execution

2. **For each execution**:
   - Set up a context with specific input data (using `context.WithValue`)
   - Execute the script with the prepared context (`evaluator.Eval(ctx)`)
   - Process the results (`result.Interface()`)

## Running the Examples

Each example implements the same pattern with a different engine:

```bash
go run examples/multiple-instantiation/< engine >/main.go
```

Note: The Extism example uses an embedded WebAssembly module from the `wasmdata` package, eliminating 
the need for external WASM files.

## Related Patterns

- [Simple Examples](/examples/simple): Basic one-time execution pattern
- [Data Preparation Examples](/examples/data-prep): Pattern for separating static configuration from dynamic runtime data