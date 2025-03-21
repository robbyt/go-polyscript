# Multiple Execution Risor Example

This example demonstrates the powerful "compile once, run many times" pattern for efficient script execution with different inputs.

## Concept and Rationale

The "compile once, run many times" pattern provides significant performance benefits when you need to execute the same script repeatedly with different data. Key advantages include:

- **Reduced Overhead**: Script compilation occurs only once, eliminating redundant parsing and compilation
- **Better Resource Utilization**: Memory usage is optimized by reusing the compiled script
- **Faster Execution**: Subsequent script executions are much faster without recompilation
- **Cleaner Architecture**: Separates script compilation from execution logic

This pattern is especially valuable in high-throughput applications, data processing pipelines, or any scenario where the same logic needs to be applied to multiple data sets.

## Data Flow Architecture

The flow in this pattern is fundamentally different from the simple execution example:

1. **Provider Configuration**: A `ContextProvider` is created that will pull data from Go's context at runtime
2. **Compilation Phase**: 
   - The script is compiled once into an `ExecutableUnit`
   - An evaluator is created that wraps this executable unit
   - This compilation happens only once, regardless of how many times the script will run

3. **Execution Phase** (repeated for each data set):
   - A context is created with specific data for this execution
   - The same compiled script is executed with this context
   - The `ContextProvider` pulls data from the context and makes it available to the script via the `ctx` global
   - Results are processed for this specific execution

The `ContextProvider` is the key component enabling this pattern. Unlike the `StaticProvider` which contains fixed data, the `ContextProvider` retrieves data from the Go context at execution time, allowing the same compiled script to process different data on each run.

## Performance Implications

This approach can provide order-of-magnitude performance improvements when executing the same script multiple times, as it avoids:

- Repeated parsing of the script text
- Repeated compilation of the script to bytecode
- Repeated initialization of the execution environment

The performance benefit increases with:
- The complexity of the script
- The number of executions
- The size of the script

## When to Use This Pattern

Implement this pattern when:
- The same script logic will be executed multiple times
- The script needs to process different data sets
- Performance and resource efficiency are important
- You have a high-volume processing requirement