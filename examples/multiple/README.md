# Multiple Execution Examples

This directory contains examples of the "compile once, run many times" pattern for efficient script execution with go-polyscript.

## Overview

This pattern separates script compilation from execution, providing significant performance benefits when running the same script repeatedly with different inputs. The process follows these steps:

1. Compile the script once into an `ExecutableUnit`
2. Create an evaluator for this compiled script
3. For each execution:
   - Prepare context with specific data for this run
   - Execute the compiled script with the prepared context
   - Process the results

## When to Use This Pattern

This pattern is recommended when:

- The same script logic will be executed multiple times
- The script needs to process different data sets
- Performance and resource efficiency are important
- You're building high-throughput applications

## Implementation Details

These examples use the following components:

- `ContextProvider`: Retrieves data dynamically from the Go context at runtime
- Script data is accessed via the `ctx` global variable within scripts
- The compiled script is reused for each execution

## Performance Benefits

This approach provides significant performance improvements by avoiding:

- Repeated parsing and compilation of the script
- Repeated initialization of the execution environment
- Redundant memory allocation

The performance benefit increases with script complexity, size, and execution frequency.

## Running the Examples

To run any example:

```bash
cd examples/multiple/<engine>
go run main.go
```

Note: The Extism example requires a WebAssembly module (provided as `main.wasm` in the `extism` directory).

## Next Steps

For more advanced usage patterns, see:

- [Data Preparation Examples](/examples/data-prep): Separating data preparation from script evaluation