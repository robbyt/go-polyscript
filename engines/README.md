# Machine Implementations

This package contains virtual machine implementations for executing scripts in various languages through a consistent interface. While each supported VM has its own unique characteristics, they all follow a standardized flow pattern.

## Design Philosophy

1. **Common Interface**: All VMs present the same interface (`evaluationEvaluator`) regardless of underlying implementation
2. **Separation of Concerns**: Compilation, data preparation, and execution are distinct phases
3. **Thread-safe Evaluation**: Each VM is designed to allow concurrent execution of scripts
3. **Context-Based Data Flow**:  Runtime data is accessed with a `context.Context` object (saved/loaded with a `data.Provider`) 
4. **Execution Results**: All VMs return the same `evaluation.EvaluatorResponse` object, which contains the execution result and metadata

## Dataflow & Architecture

1. **Compilation Instantiation**
   - Each VM has a `NewCompiler` function that returns a compiler instance that implements the `script.Compiler` interface
   - The `NewCompiler` function may have some VM-specific options
   - The `Compiler` object includes a `Compile` method that takes a `loader.Loader` implementation
   - `loader.Loader` is a generic way to load script content from various sources
   - Compile-time errors are captured and returned to the caller
   - A `script.ExecutableContent` is returned by `Compile`

2. **Executable Creation Stage**
   - The `script.ExecutableUnit` is a wrapper around the `script.ExecutableContent`
   - `NewExecutableUnit` receives a `Compiler` and several other objects
   - Calls the `script.Compiler` to compile the script, storing the result in the `ExecutableContent`
   - The `ExecutableUnit` is responsible for managing the lifecycle of the script execution

3. **Evaluator Creation**
   - `NewEvaluator` takes a `script.ExecutableUnit` and returns an object that implements `evaluationEvaluator`
   - At this point it can be called with `.Eval(ctx)`, however input data is required it must be prepared

4. **Data Preparation Stage**
   - This phase is optional, and must happen prior to evaluation when runtime input data is used
   - The `Evaluator` implements the `data.Setter` interface, which has an `AddDataToContext` method
   - The `AddDataToContext` method takes a `context.Context` and a variadic list of `map[string]any`
   - `AddDataToContext` calls the `data.Provider` to store the data, somewhere accessible to the Evaluator
   - The conversion is fairly opinionated, and handled by the `data.Provider`
   - For example, it converts an `http.Request` into a `map[string]any` using the schema in `helper.RequestToMap`
   - The `AddDataToContext` method returns a new context with the data stored or linked in it

5. **Execution Stage**
   - When `Eval(ctx)` is called, the `data.Provider` first loads the input data into the VM
   - The VM executes the script and returns an `evaluation.EvaluatorResponse`

6. **Result Processing**
   - The process for building the `evaluation.EvaluatorResponse` is different for each VM
   - There are several type conversions, and the result is accessible with the `Interface()` method
   - The `evaluation.EvaluatorResponse` also contains metadata about the execution

## Engine-Specific Data Handling

While all engines receive the same `map[string]any` input data, **each engine processes and exposes this data differently** to the script runtime. Understanding these differences is important for structuring your data correctly.

### Risor Engine: `ctx` Variable Wrapper

**Data Processing:** `engines/risor/internal/converters.go`
- Input data is wrapped in a global `ctx` variable
- All data becomes accessible via `ctx["key"]` in scripts

**Example:**
```go
// Go code
data := map[string]any{
    "name": "World",
    "config": map[string]any{"debug": true},
}

// Risor script access
name := ctx["name"]           // "World"
debug := ctx["config"]["debug"] // true
```

### Starlark Engine: `ctx` Dictionary Wrapper

**Data Processing:** `engines/starlark/internal/converters.go`
- Input data is converted to Starlark types and wrapped in a `ctx` dictionary
- All data becomes accessible via `ctx["key"]` in scripts

**Example:**
```go
// Go code
data := map[string]any{
    "name": "World",
    "config": map[string]any{"debug": true},
}

// Starlark script access
name = ctx["name"]           # "World"
debug = ctx["config"]["debug"] # true
```

### Extism Engine: Direct JSON Pass-Through

**Data Processing:** `engines/extism/internal/converters.go`
- Input data is marshaled directly to JSON and passed to the WASM module
- **No wrapper variable** - the WASM module receives the raw JSON structure
- **Data structure must exactly match what your WASM module expects**

**Example:**
```go
// Go code
data := map[string]any{
    "name": "World",
    "config": map[string]any{"debug": true},
}

// WASM module receives JSON directly:
// {"name": "World", "config": {"debug": true}}
```

### Key Implications

1. **Risor/Starlark**: Any data structure works - everything is accessible via `ctx["key"]`
2. **Extism/WASM**: Data structure must match your WASM module's expectations exactly
3. **Flexibility**: WASM modules have complete control over their input format
4. **Consistency**: Risor/Starlark provide a standardized `ctx` interface

### Troubleshooting WASM Data Structure Issues

If your WASM module reports errors like "input string is empty" or "missing field":

1. **Check the expected JSON structure** in your WASM module's input parsing code
2. **Structure your Go data** to match exactly what the WASM module expects
3. **Use the debug logging** in development to verify the JSON being passed

**Example for a WASM module expecting `{"request": {"Body": "text"}, "static_data": {...}}`:**
```go
data := map[string]any{
    "request": map[string]any{
        "Body": "text to process",
    },
    "static_data": map[string]any{
        "search_characters": "aeiou",
        "case_sensitive": false,
    },
}
```

## Data Provider Patterns

For detailed information about data provider patterns, usage examples, and best practices, see the [platform/data documentation](../platform/data/README.md).

The `platform/data` package provides:
- **StaticProvider**: For configuration and constants that don't change
- **ContextProvider**: For thread-safe dynamic runtime data that changes per request  
- **CompositeProvider**: For combining static configuration with dynamic runtime data

Key points for engine usage:
- **Risor/Starlark**: Data is accessible via the top-level `ctx` variable in scripts
- **Extism/WASM**: Data is passed directly as JSON to the WASM module (no `ctx` wrapper)
- Use explicit keys when adding data: `map[string]any{"request": httpRequest}`
- HTTP requests are automatically converted using `helpers.RequestToMap`
