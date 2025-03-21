# Simple Starlark Example

This example demonstrates the basic usage pattern for executing a Starlark script with go-polyscript.

## Concept and Rationale

This example illustrates the most straightforward way to run an embedded script using go-polyscript. In this pattern, the script is compiled and executed in a single operation, making it ideal for:

- One-off script executions
- Simple automation tasks
- Configuration processing
- Quick data transformations

While simple, this approach establishes the foundation for understanding how go-polyscript integrates with your Go application.

## Data Flow

The script execution flow follows this pattern:

1. **Data Preparation**: The Go application prepares static data (`{"name": "World"}`) to be made available to the script
2. **Provider Creation**: A `StaticProvider` is created that serves as a bridge between your Go data and the script
3. **Engine Configuration**: A Starlark engine is configured with appropriate settings (globals, imports)
4. **Script Compilation and Execution**: The script is compiled and immediately executed with the provided data
5. **Result Handling**: The script returns a structured result that is processed by the Go application

The `StaticProvider` used in this example is designed for simple cases where all data is known at compile time. The script receives this data through the `ctx` global variable, which acts as a bridge between your Go application and the embedded script.

## When to Use This Pattern

Use this approach when:

- Your script only needs to execute once
- All input data is known in advance
- You don't need to reuse the compiled script
- Performance is not critical

For scenarios requiring repeated execution of the same script with different inputs, see the "Multiple Execution" example which demonstrates the more efficient "compile once, run many times" pattern.