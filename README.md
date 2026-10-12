# go-polyscript

<!-- Each Go code block below is a copy of tested code under examples/docs, named by the snippet comment above it. Edit that code first, then copy it here; docs_test.go fails when the two differ. -->

[![Go Reference](https://pkg.go.dev/badge/github.com/robbyt/go-polyscript.svg)](https://pkg.go.dev/github.com/robbyt/go-polyscript)
[![Go Report Card](https://goreportcard.com/badge/github.com/robbyt/go-polyscript)](https://goreportcard.com/report/github.com/robbyt/go-polyscript)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=robbyt_go-polyscript&metric=coverage)](https://sonarcloud.io/summary/new_code?id=robbyt_go-polyscript)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

A unified abstraction package for loading and running various scripting languages and WASM modules in your Go app.

## Overview

go-polyscript democratizes different scripting engines by abstracting the loading, data handling, runtime, and results handling, allowing for interchangeability of scripting languages. The library provides interfaces and implementations for "engines", "executables", "evaluators", and the final "result".

The top-level package (`polyscript.go`) exposes a single generic constructor — `polyscript.New[E]` — that handles the common cases for all three engines. For finer-grained control (custom compilers, custom data providers, etc.), the per-engine sub-packages under `engines/` are also exported.

## Features

- **Unified Abstraction API**: Common interfaces and implementations for several scripting languages
- **Flexible Engine Selection**: Easily switch between different script engines
- **Thread-safe Data Management**: Multiple ways to provide input data to scripts
- **Compilation, Evaluation, and Data Handling**: Compile scripts once with static data when creating the evaluator instance, then run multiple evaluation executions with variable runtime input.

## Engines Implemented

- **Risor**: A fast scripting language designed for embedding in Go applications
- **Starlark**: Google's deterministic configuration language (used in Bazel, and others)
- **Extism**: Pure Go runtime and plugin system for executing WASM

## Installation

```bash
go get github.com/robbyt/go-polyscript@latest
```

## Quick Start

Using go-polyscript with the Risor scripting engine:

<!-- snippet: examples/docs/quickstart/main.go -->
```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/robbyt/go-polyscript"
)

func main() {
	script := `
		// The ctx object holds the input data map
		let name = ctx.get("name")

		let p = "."
		if (ctx.get("excited")) {
			p = "!"
		}

		let message = "Hello, " + name + p

		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`

	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	result, err := evaluator.Eval(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result.Interface())
}
```

The older `FromRisorString*`, `FromStarlark*`, and `FromExtism*` constructors still work but are deprecated and slated for removal in v1.

## API at a glance

`polyscript.New[E]` is the single entry point. `E` selects the engine, the first argument is a `context.Context` (used while loading and compiling the script, and bounding the evaluator's lifetime), the second is a `Source`, and the remaining arguments are `Option[E]`s.

Release an evaluator you no longer need with `defer evaluator.Close(ctx)`, placed after the error check. For an engine that holds resources, `Close` stops new `Eval` calls at once and frees what the engine holds; for one that holds none, `Close` changes nothing and `Eval` keeps working. See [engines/README.md](engines/README.md) for what each engine does. `Close` waits for in-flight `Eval` calls only until its `ctx` is done: if that comes first, `Close` returns the ctx's error and the resources are released when the last in-flight `Eval` returns. With nothing in flight it releases them at once, even with a cancelled `ctx`. Like `http.Server.Shutdown`, `Close` is safe to call more than once and concurrently: the resources are released once, and each call waits on its own `ctx`.

### Evaluator lifetime

The `ctx` passed to `New` also bounds the evaluator's lifetime: when it is done, the evaluator closes itself as if `Close` had been called. For an engine that holds resources, an evaluator built with a per-request or timeout `ctx` is therefore closed when that `ctx` ends; for one that holds none, the end of the lifetime changes nothing. For compile-once-run-many, pass a long-lived `ctx`, or keep a short one for loading and compiling and pass the long-lived one with `WithLifetime[E]`. Call `Close` or let the lifetime end: an evaluator that is neither closed nor bound to a lifetime that ends keeps what its engine holds, since nothing releases it when the evaluator is garbage collected. If the lifetime `ctx` is already done when construction finishes, `New` closes what it built and returns an error; a lifetime that ends later, even before `New` returns, closes the evaluator as described above.

<!-- snippet: examples/docs/snippets/lifetime.go#lifetime -->
```go
import (
	"context"
	"time"

	"github.com/robbyt/go-polyscript"
)

// Give loading and compiling a deadline, but keep the evaluator
// open until appCtx ends or Close is called.
loadCtx, cancel := context.WithTimeout(appCtx, 10*time.Second)
defer cancel()

evaluator, err := polyscript.New[polyscript.Extism](
	loadCtx,
	polyscript.FromBytes(wasmBytes),
	polyscript.WithEntryPoint("greet"),
	polyscript.WithLifetime[polyscript.Extism](appCtx),
)
if err != nil {
	return err
}
defer evaluator.Close(appCtx)
```

### Engines

| Type                  | Use it for                              |
| --------------------- | --------------------------------------- |
| `polyscript.Risor`    | Risor scripts                           |
| `polyscript.Starlark` | Starlark configuration scripts          |
| `polyscript.Extism`   | WebAssembly modules via the Extism PDK  |

### Sources

| Builder                            | Backed by                                                                |
| ---------------------------------- | ------------------------------------------------------------------------ |
| `polyscript.FromString(s)`         | An in-memory script string                                               |
| `polyscript.FromBytes(b)`          | An in-memory byte slice (typically WASM); the slice is copied            |
| `polyscript.FromFile(path)`        | An absolute path on disk                                                 |
| `polyscript.FromLoader(l)`         | Any custom `loader.Loader` (e.g. an HTTP loader; see [Loading Scripts](#loading-scripts)) |

### Options

| Option                             | Applies to              | Effect                                                              |
| ---------------------------------- | ----------------------- | ------------------------------------------------------------------- |
| `WithStaticData[E](map)`           | All engines             | Bakes a fixed data map into the evaluator at construction time      |
| `WithLogHandler[E](handler)`       | All engines             | Routes diagnostic logs through the given `slog.Handler`             |
| `WithLifetime[E](ctx)`             | All engines             | Bounds the evaluator's lifetime by `ctx` instead of the `ctx` passed to `New` |
| `WithEntryPoint(name)`             | `Extism` only (compile-time enforced) | Sets the WASM function name to invoke; required for Extism |
| `WithContextTermination(bool)`     | `Extism` only (compile-time enforced) | Whether cancelling the `Eval` ctx stops a running guest. On by default; wazero's termination checks roughly double per-call time on small modules, so pass `false` for trusted modules on hot paths that never need a timeout |

> **Note on type arguments.** `WithStaticData`, `WithLogHandler` and `WithLifetime` are generic over the engine. Go's current type inference can't always infer `E` for them when the surrounding `New[E]` call has a non-variadic `Source` parameter, so these helpers usually need an explicit type argument: `polyscript.WithStaticData[polyscript.Risor](data)`. `WithEntryPoint` is bound to `Extism` and never needs one — passing it to `New[polyscript.Risor]` or `New[polyscript.Starlark]` is a compile error rather than a silent no-op.

## Loading Scripts

`FromString` is the easiest source for embedded scripts. For other locations:

### From a file on disk

<!-- snippet: examples/docs/snippets/loading.go#from-file -->
```go
ctx := context.Background()

// scriptPath is an absolute path, such as "/etc/polyscript/hello.risor"
evaluator, err := polyscript.New[polyscript.Risor](
	ctx,
	polyscript.FromFile(scriptPath),
)
```

The path must be absolute. For relative paths, resolve them with `filepath.Abs` first.

### From an HTTP endpoint

`FromLoader` wraps any `loader.Loader`, including the HTTP loader that ships with go-polyscript:

<!-- snippet: examples/docs/snippets/http.go#http -->
```go
import (
	"context"
	"time"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

ctx := context.Background()

httpOpts := loader.DefaultHTTPOptions().
	WithTimeout(10 * time.Second).
	WithBearerAuth("my-api-token")

// scriptURL is the script's location, such as "https://scripts.example.com/greet.risor"
httpLoader, err := loader.NewFromHTTPWithOptions(scriptURL, httpOpts)
if err != nil {
	return err
}

evaluator, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromLoader(httpLoader))
```

The HTTP loader follows redirects only within the original origin (same scheme, host and port), plus an http-to-https upgrade on the same host. Any other redirect fails with `loader.ErrRedirectNotAllowed`, so custom auth headers are never sent to another host. `WithAllowCrossOriginRedirects(true)` lifts that restriction. When a followed redirect leaves the original origin, the loader strips every header it set (custom `Headers` and anything the authenticator added) plus `Referer`, keeping only `User-Agent` and the names passed to `WithRedirectForwardHeaders(...)` (which can include `Authorization`), and restores them if a later redirect returns to the original origin. Redirects from https to http are always refused.

### Capturing diagnostic logs

<!-- snippet: examples/docs/snippets/logging.go#logging -->
```go
import (
	"context"
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript"
)

ctx := context.Background()
handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

evaluator, err := polyscript.New[polyscript.Risor](
	ctx,
	polyscript.FromString(script),
	polyscript.WithLogHandler[polyscript.Risor](handler),
	polyscript.WithStaticData[polyscript.Risor](data),
)
```

## Working with Data Providers

To send input data to a script, use a "data provider" implementation. There are several built-in providers, or implement your own and stack multiple with the `CompositeProvider`.

### StaticProvider

For example, attaching `WithStaticData` to a Risor evaluator wires up a `StaticProvider` internally to send the static data map into the evaluator during creation.

<!-- snippet: examples/docs/snippets/providers.go#static-provider -->
```go
ctx := context.Background()

evaluator, err := polyscript.New[polyscript.Risor](
	ctx,
	polyscript.FromString(script),
	polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "cats", "excited": true}),
)
```

### ContextProvider

A constructor created without `WithStaticData` uses a `ContextProvider`, so dynamic per-request data can be threaded in through the context.

<!-- snippet: examples/docs/snippets/providers.go#context-provider -->
```go
ctx := context.Background()

evaluator, err := polyscript.New[polyscript.Risor](ctx, polyscript.FromString(script))
if err != nil {
	return err
}

runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
if err != nil {
	return err
}

// Execute with the "enriched" context containing the link to the input data
result, err := evaluator.Eval(enrichedCtx)
```

#### Concurrency and thread-safety

`ContextProvider.AddDataToContext` is safe for concurrent use. Each call returns a derived context independent of its parent: data merged into the derived context never affects the parent or any sibling derived chain.

Existing data from the parent context is deep-copied when forming the derived context, so concurrent enrichments from goroutines that share a parent context — even one already populated with nested maps — do not race on shared inner map storage.

The typical pattern is one derived-context chain per request, with the returned context threaded forward for any subsequent enrichment:

<!-- snippet: examples/docs/snippets/providers.go#derived-contexts -->
```go
ctx, err := provider.AddDataToContext(parent, requestData)
if err != nil {
	return err
}
ctx, err = provider.AddDataToContext(ctx, moreData) // build on the returned ctx
if err != nil {
	return err
}
```

For the details of which value kinds are deep-copied versus stored by reference, see the [`AddDataToContext` godoc](https://pkg.go.dev/github.com/robbyt/go-polyscript/platform/data#ContextProvider.AddDataToContext).

### Combining Static and Dynamic Runtime Data

Use the following pattern for fixed configuration values and per-request data. Initial loading, parsing, and instantiating the script is relatively slow, so the example below shows how to set up the script once with static data and then reuse it many times with dynamic runtime data.

<!-- snippet: examples/docs/snippets/providers.go#static-and-dynamic -->
```go
ctx := context.Background()

staticData := map[string]any{
	"name":    "User",
	"excited": true,
}

// Create the evaluator with the static data
evaluator, err := polyscript.New[polyscript.Risor](
	ctx,
	polyscript.FromString(script),
	polyscript.WithStaticData[polyscript.Risor](staticData),
)
if err != nil {
	return err
}

// For each request, prepare dynamic data
requestData := map[string]any{"name": "Robert"}
enrichedCtx, err := evaluator.AddDataToContext(ctx, requestData)
if err != nil {
	return err
}

// Execute with both static and dynamic data available
result, err := evaluator.Eval(enrichedCtx)
```

## Architectural Design

go-polyscript is structured around a few key concepts:

1. **Loader**: Loads script content from various sources (disk, `io.Reader`, strings, http, etc.)
2. **Compiler**: Validates and compiles scripts into internal "bytecode"
3. **ExecutableUnit**: Compiled script bundle, ready for execution
4. **Engine**: A specific implementation of a scripting engine (Risor, Starlark, Extism)
5. **Evaluator**: Executes compiled scripts with provided input data
6. **DataProvider**: Sends data to the engine prior to evaluation
7. **EvaluatorResponse**: The response object returned from all **Engine**s

### Note on Data Access Patterns

go-polyscript uses a two-layer approach for handling data:

1. **Data Provider Layer**: The `Provider` interface (via `AddDataToContext`) handles storage mechanisms and general type conversions. This layer is pluggable, allowing data to be stored in various backends while maintaining a consistent API.

2. **Engine-Specific Layer**: Each engine's `Evaluator` implementation handles the engine-specific conversions between the stored data and the format required by that particular scripting engine.

This separation allows scripts to access data with consistent patterns regardless of the storage mechanism or script engine. For example, data you store with `{"config": value}` will be accessible in your scripts as `ctx["config"]`, with each engine handling the specific conversions needed for its runtime.

See the [Data Providers](#working-with-data-providers) section for more details.

## Working with other Engines

### Starlark
Starlark syntax is a deterministic "python-like" language designed for complex configuration, not so much for dynamic scripting. It's high performance, but the capabilities of the language are very limited. Read more about it here: [Starlark-Go](https://github.com/google/starlark-go)

<!-- snippet: examples/docs/snippets/starlark.go#starlark -->
```go
scriptContent := `
# Starlark has access to ctx variable
name = ctx["name"]
message = "Hello, " + name + "!"

# Create the result dictionary
result = {"greeting": message, "length": len(message)}

# Assign to _ to return the value
_ = result
`

ctx := context.Background()

evaluator, err := polyscript.New[polyscript.Starlark](
	ctx,
	polyscript.FromString(scriptContent),
	polyscript.WithStaticData[polyscript.Starlark](map[string]any{"name": "World"}),
)
if err != nil {
	return err
}

// Execute with a context
result, err := evaluator.Eval(ctx)
```

### WASM with Extism

Extism uses the Wazero WASM runtime for providing WASI abstractions, and an easy input/output memory sharing data system. Read more about writing WASM plugins for the Extism/Wazero runtime using the Extism PDK here: [extism.org](https://extism.org/docs/concepts/pdk)

> **Known issue:** wazero v1.12.0 has a data race that `go test -race` can report when wazero runtimes are created concurrently. go-polyscript's Extism compiler works around it, but apps that also create wazero runtimes directly can still hit it. See [engines/README.md](engines/README.md#known-issue-wazero-data-race-under--race) for the cause and workarounds.

<!-- snippet: examples/docs/extism/main.go -->
```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
)

func main() {
	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Extism](
		ctx,

		// pre-compiled WASM example module
		polyscript.FromBytes(wasmdata.TestModule),

		// main entrypoint function in the WASM module
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),

		// the go-polyscript Extism engine will encode the static data into
		// JSON and send it to the WASM application
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	// release the compiled module's wazero runtime when done
	defer evaluator.Close(ctx)

	// Execute, and print the result
	result, err := evaluator.Eval(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result.Interface())
}
```

## Direct engine subpackage usage

Most callers should use `polyscript.New[E]`. For finer-grained control over a single engine — for example, plugging in a custom `data.Provider` — each engine subpackage exposes the same shape: `FromXxxLoader(ctx, ldr, opts...)`, with engine-scoped `WithLogHandler`, `WithStaticData`, `WithDataProvider`, `WithLifetime`, and (Extism only) `WithEntryPoint` options.

<!-- snippet: examples/docs/snippets/direct_risor.go#direct-risor -->
```go
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

ctx := context.Background()

ldr, err := loader.NewFromString(`{"name": ctx["name"]}`)
if err != nil {
	return err
}

eval, err := risor.FromRisorLoader(
	ctx,
	ldr,
	risor.WithStaticData(map[string]any{"name": "World"}),
)
if err != nil {
	return err
}

result, err := eval.Eval(ctx)
```

Logging is optional: omit `WithLogHandler` and the engine inherits whatever `slog.Default()` has been configured to in the host process. Extism is the same shape but expects a WASM module loader and requires `WithEntryPoint`:

<!-- snippet: examples/docs/snippets/direct_extism.go#direct-extism -->
```go
import (
	"context"

	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

ctx := context.Background()

wasmLdr, err := loader.NewFromBytes(wasmBytes) // wasmBytes is a compiled WASM module
if err != nil {
	return err
}

eval, err := extism.FromExtismLoader(
	ctx,
	wasmLdr,
	extism.WithEntryPoint("greet"),
	extism.WithStaticData(map[string]any{"input": "World"}),
)
if err != nil {
	return err
}

result, err := eval.Eval(ctx)
```

## License

Apache License 2.0
