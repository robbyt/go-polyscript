# Machine Implementations

This package contains virtual machine implementations for executing scripts in various languages through a consistent interface.

## Data Flow

All machine implementations follow a uniform data flow pattern:

1. **Compilation Stage**
   - Source code is parsed and compiled to bytecode
   - Compile-time errors are captured and returned
   - Machine-specific optimizations are applied

2. **Data Preparation**
   - `PrepareContext` enriches context with runtime data
   - Data providers handle accessing and storing execution data
   - Context is the primary vehicle for data transfer between components

3. **Execution Stage**
   - Bytecode is executed with data from context
   - A consistent global variable (`ctx`) provides access to input data
   - Context cancellation signals terminate execution

4. **Result Processing**
   - VM-specific results are converted to Go types
   - Type conversions maintain semantic equivalence
   - Execution metrics (timing, etc.) are captured consistently

## Implementation Requirements

Each machine implementation must:

1. Implement the `engine.EvaluatorWithPrep` interface
2. Handle context cancellation properly
3. Use the executable unit's data provider for input/output
4. Maintain consistent error wrapping patterns
5. Track execution timing in a uniform way
6. Pass data to scripts through a consistent interface

## Testing Guidelines

Virtual machine implementations should include tests that verify:

1. Proper handling of various input data types
2. Context cancellation behavior
3. Error conditions and edge cases
4. Type conversion correctness
5. Integration with the executable unit lifecycle

By following these guidelines, we ensure consistent behavior across all machine implementations while allowing for VM-specific optimizations.