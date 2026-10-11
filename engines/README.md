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

7. **Shutdown**
   - `Evaluator.Close(ctx)` releases resources held by the compiled script by calling `ExecutableUnit.Close`, which closes the `ExecutableContent` if it implements `Close(context.Context) error`. The unit calls the content's `Close` at most once, through a `sync.Once`: the first close, whether from `Close`, the lifetime hook or `BindLifetime` finding the lifetime already ended, runs it with its own `ctx`, and every other `Close`, concurrent or later, waits for it and returns its result
   - The evaluator's lifetime is bound to a ctx: the one passed to `FromXxxLoader` (or `polyscript.New`), or the one given to `WithLifetime`. The engine constructor calls `ExecutableUnit.BindLifetime`, which registers a `context.AfterFunc` hook that closes the content when that ctx is done; only the hook's stop func is kept, and `Close` stops the hook first, so it never fires after `Close`. If the lifetime ctx is already done when construction finishes, the constructor closes what it built and returns an error wrapping the ctx's error
   - The lifetime hook holds the unit's close guard only through a `weak.Pointer`, so a long-lived lifetime ctx keeps neither the `ExecutableUnit` nor its content in memory. The weak pointer only goes nil once the unit has been garbage collected, so until then the end of the lifetime still closes the content, even for a dropped unit whose content a caller retained. When a unit is garbage collected, a `runtime.AddCleanup` on its close guard unregisters the hook, and the end of the lifetime no longer closes the content. The unit never closes content on garbage collection: after that, a caller that retained the content (from `GetContent`) keeps using it and owns its release. A caller that needs the content to outlive the unit should not bind it to a lifetime that may end first. Bytecode taken from the content need not keep the content alive; retain the content itself
   - Content that holds resources releases them from its own cleanup when it becomes unreachable without being closed, like `os.File`. Extism: `compiler.NewExecutable` registers a `runtime.AddCleanup` on the `Executable` that releases the compiled plugin and its wazero runtime once the `Executable` is garbage collected without having been released; the release runs at most once, so a closed `Executable` is not released again. `Close` releases it sooner. A plugin obtained from `GetExtismByteCode` does not keep the `Executable` alive; use `WithPlugin`
   - Extism: `Close` closes the compiled plugin and its wazero runtime. `Eval` calls made once `Close` has been called (even while it waits) fail fast with `compiler.ErrExecutableClosed` without calling the data provider. `Close` waits for in-flight `Eval` calls, including any that are still loading data from the provider, until its `ctx` is done. If they return first, the last one releases the runtime and `Close` returns the release error; if the `ctx` is done first, `Close` returns the ctx's error and the runtime is released when the last in-flight `Eval` returns. With nothing in flight, `Close` releases the runtime and returns the release error, even with a cancelled `ctx`. The release never uses the caller's cancelled `ctx`, and the `Executable` runs it exactly once through its own `sync.Once`. `compiler.Executable.IsClosed` reports `true` once the runtime has been released, not while `Close` is still waiting
   - Extism: when the lifetime ctx ends, the evaluator closes itself the same way, so an evaluator built with a request or timeout ctx stops working when that ctx ends. Build a reused evaluator with a long-lived ctx, or pass one with `WithLifetime`
   - Cancelling an `Eval`'s `ctx` shortens the wait only if the guest stops and the data provider honours cancellation
   - A data provider must not call `Eval` on the same evaluator, or `Close` with a `ctx` that is never done: `Close` would wait for the `Eval` that is calling it
   - Risor and Starlark hold no releasable resources, so `Close` and the end of the lifetime ctx are no-ops and `Eval` keeps working. A lifetime ctx that is already done when construction finishes still makes the constructor return an error
   - `Close` is safe to call more than once and concurrently (only the first close's `ctx` bounds the wait); call it (e.g. `defer evaluator.Close(ctx)`) when an evaluator is discarded, especially for evaluators built dynamically at runtime

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

### Known Issue: wazero Data Race Under `-race`

wazero v1.12.0, the WASM runtime under the Extism engine, looks up and caches
its own version the first time a runtime is created. If two runtimes are
created at the same moment (for example, parallel tests that each compile an
Extism module), both write that cache at once and Go's race detector reports a
`DATA RACE` in `wazero/internal/version.GetWazeroVersion`. The write is
harmless in practice, but it fails `go test -race`.

This is fixed upstream in
[tetratelabs/wazero#2536](https://github.com/tetratelabs/wazero/pull/2536)
(commit [`6edbb8c`](https://github.com/tetratelabs/wazero/commit/6edbb8c01a)),
which is not in a wazero release yet. Until it is, go-polyscript works around
it: the Extism compiler caches wazero's version once per process, under a
`sync.Once`, before it creates its first runtime (`internal/wazerorace`). So
concurrent Extism compiles through go-polyscript are safe under `-race`.

Your app can still hit the race if it creates its own wazero runtimes (or
calls the Extism SDK directly) at the same moment as go-polyscript's first
compile. Two workarounds:

- Create and close one wazero runtime before the parallel work starts, e.g. in
  `TestMain`. Each test package is its own binary, so every package that
  needs it must do this. go-polyscript's own tests do this only in
  `engines/extism/wasmdata`, which calls the Extism SDK directly:
  ```go
  func TestMain(m *testing.M) {
      ctx := context.Background()
      if err := wazero.NewRuntime(ctx).Close(ctx); err != nil {
          fmt.Fprintln(os.Stderr, "wazero warm-up failed:", err)
          os.Exit(1)
      }
      os.Exit(m.Run())
  }
  ```
- Require an unreleased wazero commit that includes the fix in your own
  `go.mod`, e.g. `go get github.com/tetratelabs/wazero@6edbb8c01a`.

`engines/extism/wazero_race_test.go` fails as soon as the wazero dependency
changes, as a reminder to remove this note, `internal/wazerorace` and its
callers once a release with the fix lands.

### Testing: Mock Plugins and the Garbage Collector

A `compiler.Executable` releases its plugin from its own `runtime.AddCleanup` when it is garbage collected without being closed, the way `os.File` closes a file descriptor. So any test that builds an `Executable` around a mock plugin and never closes it will see `Close` called on that mock at some point after the test.

- A testify mock panics on a call it was not told to expect, and a panic inside a cleanup crashes the whole test binary, not just the test that leaked.
- It only happens when the garbage collector runs at the wrong moment, so it shows up as a random failure, often in a later test or package, sometimes only after a rebase.

**Rules for Extism tests:**

- Give every mock plugin that is not explicitly closed a `Close` expectation that tolerates the cleanup:
  ```go
  mockPlugin.On("Close", mock.Anything).Return(nil).Maybe()
  ```
  Or close the `Executable` (or its evaluator) in the test, with a matching `Close` expectation.
- Run new or changed Extism tests once with the garbage collector forced to run constantly, which makes a missing expectation crash every time:
  ```bash
  GOGC=1 go test -race -count=3 ./engines/extism/... .
  ```
- Never use `recover()` to hide the panic; fix the expectation.

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
