# Data Preparation Examples

This directory contains examples demonstrating how to separate data preparation from script evaluation in go-polyscript.

## Overview

The data preparation pattern separates static configuration data from dynamic runtime data, making code more modular and flexible. This pattern uses a composite data provider combining static and dynamic data sources, with a clear separation between configuration, preparation, and evaluation phases.

## When to Use This Pattern

This pattern is valuable for:

- **Distributed Architecture**: Prepare data on one system (e.g., web server) and evaluate on another (e.g., worker)
- **Separation of Concerns**: Clearly separate data processing from script execution
- **Layered Data Access**: Combine static configuration with dynamic runtime data
- **Performance Optimization**: Prepare data asynchronously while other operations are happening

## Implementation Pattern

These examples follow a consistent three-phase pattern:

1. **Create evaluator with static data**: Set up an evaluator with configuration data that remains constant across executions
2. **Prepare runtime data**: Add dynamic data to the context using the `PrepareContext` method
3. **Evaluate script**: Execute the script with the enriched context

## Running the Examples

Each example follows the same pattern but demonstrates it with a different script engine:

```bash
go run examples/data-prep/<engine>/main.go
```

Note: The Extism example requires a WASM file. If not found automatically, you may need to compile it using the `Makefile` in `/machines/extism/testdata/`.

## Related Patterns

- [Simple Examples](/examples/simple): Basic one-time execution pattern
- [Multiple Examples](/examples/multiple-instantiation): Compile-once-run-many pattern