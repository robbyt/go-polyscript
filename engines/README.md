# Engine Implementations

> **Note:** This documentation is intended for developers implementing new engines or understanding the internal architecture of go-polyscript. For general usage, see the [main README](../README.md) and [examples](../examples).

This package contains engine implementations for executing scripts in various languages through a consistent interface. While each supported engine has its own unique characteristics, they all implement the same interfaces and lifecycle.

## Design Philosophy

1. **Common Interface**: All engines implement the same `platform.Evaluator` interface regardless of underlying implementation
2. **Separation of Concerns**: Compilation, data preparation, and execution are distinct phases
3. **Thread-safe Evaluation**: Each engine is designed to allow concurrent execution of scripts
4. **Context-Based Data Flow**: Runtime data is accessed with a `context.Context` object (saved/loaded with a `data.Provider`)
5. **Unified Response Type**: All engines return a `platform.EvaluatorResponse` containing the execution result and metadata

## Dataflow & Architecture

1. **Compilation Stage**
   - Each engine has a `NewCompiler` function that returns a compiler instance that implements the `script.Compiler` interface
   - The `NewCompiler` function may have some engine-specific options
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
   - `NewEvaluator` takes a `script.ExecutableUnit` and returns an object that implements `platform.Evaluator`
   - At this point it can be called with `.Eval(ctx)`, however runtime data is required it must be prepared

4. **Data Preparation Stage**
   - This phase is optional, and must happen prior to evaluation when runtime data is used
   - The `Evaluator` implements the `data.Setter` interface, which has an `AddDataToContext` method
   - The `AddDataToContext` method takes a `context.Context` and a variadic list of `map[string]any`
   - `AddDataToContext` calls the `data.Provider` to store the data, somewhere accessible to the Evaluator
   - The conversion is fairly opinionated, and handled by the `data.Provider`
   - For example, it converts an `http.Request` into a `map[string]any` using the schema in `helper.RequestToMap`
   - The `AddDataToContext` method returns a new context with the data stored or linked in it

5. **Execution Stage**
   - When `Eval(ctx)` is called, the `data.Provider` first loads the runtime data into the engine
   - The engine executes the script and returns a `platform.EvaluatorResponse`

6. **Result Processing**
   - The process for building the `platform.EvaluatorResponse` is different for each engine
   - There are several type conversions, and the result is accessible with the `Interface()` method
   - The `platform.EvaluatorResponse` also contains metadata about the execution

## Engine-Specific Data Handling

While all engines receive the same `map[string]any` input data, **each engine processes and exposes this data differently** to the script runtime. Understanding these differences is important for structuring your data correctly.

### Risor Engine: `ctx` Context Wrapper

**Data Processing:** `engines/risor/internal/converters.go`
- Input data is wrapped in a global `ctx` variable
- All data is accessible via `ctx["key"]` in scripts

**Example:**
```go
// Go code
data := map[string]any{
    "name": "World",
    "config": map[string]any{"debug": true},
}

// Risor script access
let name = ctx["name"]           // "World"
let debug = ctx["config"]["debug"] // true
```

### Starlark Engine: `ctx` Context Wrapper

**Data Processing:** `engines/starlark/internal/converters.go`
- Input data is converted to Starlark types and wrapped in a `ctx` dictionary
- All data is accessible via `ctx["key"]` in scripts

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

### Extism Engine: Direct JSON Processing

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

## Script Return Value Handling

Each engine returns the script's final value to the caller, but they treat
"the final value is a callable" differently. This is intentional, not an
oversight, and reflects the source language's idioms.

### Risor: callable returns are an error

A Risor script's value is its last expression. If that expression evaluates
to a function (e.g. a trailing lambda like `x => x + 1`), evaluation returns
an error of the form `"function object returned from script: <inspect>"`.

Rationale: Risor is expression-oriented (top-level expressions are normal),
so a trailing function value is a legitimate result a script might want to
return. Auto-invoking would guess at arity and arguments, which would surprise
callers who genuinely meant "give me the function itself". If the script
intends to call the function, it should do so explicitly:

```risor
let greet = func(name) { "Hello, " + name }
greet("World")        // returns "Hello, World"
```

See `engines/risor/evaluator/evaluator.go` (`case "function":` branch).

### Starlark: callable returns are auto-invoked with no args

The Starlark evaluator pulls the script's return value out of the global
named `_` (the standard Starlark convention for "the unused/anonymous value").
If `_` is None or unset, it falls back to a global named `result`. If that
final value is a callable (a `*starlark.Function`, `*starlark.Builtin`, or
any `starlark.Callable`), the evaluator auto-invokes it with no positional
args and no kwargs, then returns the call's result. The returned value is
frozen via `val.Freeze()` to prevent further mutation.

Rationale: Starlark has no top-level expression statement, so a script body
conventionally ends in a `def` block followed by an assignment like
`_ = main`. Auto-call provides the ergonomic "run my main()" semantics that
the language otherwise lacks.

```starlark
def greet():
    return "Hello, World"
_ = greet                # script returns "Hello, World"
```

See `engines/starlark/evaluator/evaluator.go` (the `_` / `result` lookup
plus the `starlarkLib.Callable` auto-call branch).

### Extism / WASM: no callable-at-script-level concept

The Extism engine invokes the WASM module's named entry-point function and
returns whatever that function writes to its output buffer (JSON-decoded).
There is no "script value" that could be a callable — the function is the
WASM export, not a value a script can produce. This question doesn't apply.

### Why they differ

Both engines do the most natural thing for their host language. Forcing
either to match the other would harm its idiomatic use:

- Making Risor auto-invoke would surprise users who genuinely return functions.
- Making Starlark refuse to auto-invoke would break the standard `def main(): ...`
  end-of-script convention.

If you need uniform behavior across engines for a specific application, call
the function explicitly inside the script.

## Data Provider Patterns

For detailed information about data provider patterns, usage examples, and best practices, see the [platform/data documentation](../platform/data/README.md).

The `platform/data` package provides:
- **StaticProvider**: For configuration and constants that don't change
- **ContextProvider**: For thread-safe dynamic runtime data that changes per-request  
- **CompositeProvider**: For combining static configuration with dynamic runtime data

Key points for engine usage:
- **Risor/Starlark**: Data is accessible via the top-level `ctx` variable in scripts
- **Extism/WASM**: Data is passed directly as JSON to the WASM module (no `ctx` wrapper)
- Use explicit keys when adding data: `map[string]any{"request": httpRequest}`
- HTTP requests are automatically converted using `helpers.RequestToMap`
