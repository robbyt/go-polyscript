# Logging philosophy

This document explains how `go-polyscript` handles diagnostic logging — what
the library guarantees, what hosts can rely on, and why those decisions were
made. It also lays the choices alongside what other modern Go libraries do,
so you can see they're not idiosyncratic.

The short version: **a host that does nothing gets diagnostic logs through
`slog.Default()`. A host that wires its own `slog.Handler` gets that handler
honored at every layer. The library never writes to `os.Stdout`/`os.Stderr`
on its own.**

---

## Principles

### 1. Logging is opt-in, never required

No constructor demands a `slog.Handler`. Every public option that takes one
also accepts nil:

- `polyscript.WithLogHandler[E](nil)` — top-level, generic
- `engines/{risor,starlark,extism}.WithLogHandler(nil)` — per-engine
- `engines/{risor,starlark,extism}/compiler.WithLogHandler(nil)` — compiler-level
- `engines/{risor,starlark,extism}/compiler.WithLogger(nil)` — compiler-level

Calling any of them with nil is **a true no-op** — semantically equivalent to
not calling the option at all. It does not error, does not panic, and does
not clear state set by a prior call. A host that supplies no handler is
treated identically to a host that supplies `nil`.

### 2. nil means "inherit from `slog.Default()`"

Every internal construction site routes through one canonical helper:

```go
// internal/helpers/logger.go
func SetupLogger(handler slog.Handler, engineName, groupName string) (slog.Handler, *slog.Logger)
```

When `handler` is nil, `SetupLogger` calls `slog.Default().Handler()` and
wraps it with `engineName` as a `slog.Group`, so a record from the Risor
evaluator looks like `risor.Evaluator.<…>` regardless of whether the host
configured a handler. When `handler` is non-nil it's returned unchanged —
the host configured it the way they want and the library doesn't second-guess
the grouping.

This means a host can call `slog.SetDefault(...)` once at startup and have
every engine inherit it — no per-engine wiring, no per-call argument passing.

### 3. There is exactly one fallback path

Every optional-handler path in the package routes through `helpers.SetupLogger`.
There are no `slog.NewTextHandler(os.Stdout, …)` fallbacks. There are no
`if handler == nil { handler = … }` branches scattered through constructors.
There is no "if you forgot to wire a handler we'll print to stderr to be
helpful" behavior.

Two consequences:

- A misconfigured host never has the library writing to stdout/stderr behind
  its back, polluting structured-log pipelines.
- Changing the default-handler behavior is a one-line change in `SetupLogger`,
  not a migration across every engine constructor.

The four loader-internal `slog.Default().Debug(...)` cleanup-error logs in
`platform/script/loader/{fromHTTP,fromDisk}.go` are the one apparent
exception — they live in code paths that don't carry a logger. They still
inherit from `slog.Default()` (same fallback target as `SetupLogger`), so
the principle holds.

### 4. Functional options are silent on nil

This is a Go idiom: passing `nil` to an optional functional option is
equivalent to not calling it at all. Erroring on nil violates the contract
and forces every call site into a defensive `if x != nil { … opt(x) … }`
wrap.

We held the wrong contract for a while — the per-engine
`compiler.WithLogHandler(nil)` and `compiler.WithLogger(nil)` returned
`fmt.Errorf("log handler cannot be nil")`. That decision propagated outward:
six call sites in `polyscript.go` and `engines/{risor,starlark,extism}/new.go`
gained a `cfg.handler != nil` guard solely to avoid tripping the error
during normal construction. PR #113 (issue #111) fixed the option layer; PR
#114 (issue #112) removed the now-redundant guards.

### 5. Group names follow `(engine, struct)` shape

Every `SetupLogger` call uses `(engineName, structName)` form:

```go
helpers.SetupLogger(handler, "extism", "Compiler")
helpers.SetupLogger(handler, "extism", "Evaluator")
helpers.SetupLogger(handler, "extism", "execResult")
helpers.SetupLogger(handler, "script", "ExecutableUnit")
```

This produces records prefixed `<engine>.<struct>.<field>=…`, which is what
a host querying their logs expects: filter by `engine=extism` to see the
WASM path; filter by `script=ExecutableUnit` to see the loader/compile
machinery; etc.

---

## How this aligns with prior art

The choices above aren't novel — they line up with conventions that are by
now mainstream in modern Go.

### Standard library `log/slog`

`slog.NewTextHandler(w, nil)` accepts a nil `*HandlerOptions` and falls back
to defaults. The pattern is: **constructors accept nil with sensible
defaults; only fully-built logger objects panic on nil** (e.g. `slog.New(nil)`
panics, but that's an already-built thing — different layer).

`go-polyscript` mirrors this at the option layer: passing nil never panics,
never errors, falls back to a sensible default.

`slog.Default()` is the canonical hook for "let the host configure logging
once, library code inherits." We use it as our nil-handler target for
exactly that reason.

### `log/slog` discard handler

Go 1.24 added `slog.DiscardHandler` (issue golang/go#62005) so libraries
have a clean "log nothing" sink without instantiating a handler against
`io.Discard`. We don't use it as a default — `slog.Default()` is more
useful, since the host can swap it. But the same philosophy applies:
**nil-checks in business logic are a smell; have a sentinel sink instead.**

### OpenTelemetry `otelslog`

`otelslog.WithLoggerProvider` defaults to the global `LoggerProvider` when
the option is omitted. Absence == default. Same shape as our
`WithLogHandler`: omit it, and the global default takes over.

### `go-logr/logr`

The `logr` ecosystem documents `logr.Discard()` so callers never need to
nil-check a logger field. The principle: **nil logger should not be a
foot-gun.** Pre-#85 this library had nil-handlers writing to `os.Stdout`;
post-#85/#99 nil routes through `slog.Default()`. Different mechanism, same
intent.

### Functional options pattern (Cheney/Pike lineage)

Dave Cheney's foundational write-ups on functional options consistently
treat the option as **optional**: calling `Server(WithTimeout(...))` opts
into a non-default value; calling `Server()` accepts the default. Erroring
on `WithTimeout(0)` would be a contract violation. Same logic applies to
our `WithLogHandler(nil)` — it's the option layer's job to accept any value
the type allows; the constructor's defaults take over when the value isn't
informative.

### Stdlib `database/sql`

`sql.Open(driver, dsn)` doesn't take a logger. Drivers that want to log
configure it via their own setters or rely on `log.Default()` /
`slog.Default()`. The pattern is: **the host owns the logging surface; the
library doesn't fight it.**

We follow the same pattern with `slog.Default()` as the inheritance point.

---

## What we explicitly don't do

These are choices that look reasonable on paper but have specific failure
modes we avoided:

- **No `os.Stdout` / `os.Stderr` fallback writes.** Pre-#99 several
  evaluator constructors did `slog.NewTextHandler(os.Stdout, nil)` when
  given nil. That's hostile to a host running under systemd-journal,
  Docker, or anything that captures stdout for output rather than logs. The
  library has no business deciding where logs go.

- **No "Handler is nil, using default" warning at construction.** Same
  evaluators emitted a one-time `Warn` log when falling back. That's noise
  the host didn't ask for, especially during tests where nil is the
  intentional choice.

- **No panic on nil at the option layer.** `slog.New(nil)` panics, and
  that's correct for *constructors*, but functional options aren't
  constructors — they're configuration knobs, and a nil knob is "use the
  default."

- **No mandatory handler argument on `polyscript.New`.** The host's
  ergonomic baseline is `polyscript.New[Risor](polyscript.FromString(src))`
  — nothing else. Adding a required handler argument would force every
  caller to either write `slog.Default()` explicitly (boilerplate) or pass
  nil (which we'd then have to handle anyway).

- **No "library defines its own logger interface."** `logr` predates
  `slog`; we don't. There's no value in re-abstracting `slog.Handler`
  behind a `Logger` interface for our use case.

---

## Architectural map

If you're reading the source, the chain is:

```
host code
  └─ polyscript.New[E](src, polyscript.WithLogHandler[E](h))
       └─ engines/<engine>.FromXxxLoader(ldr, <engine>.WithLogHandler(h))
            ├─ engines/<engine>/compiler.New(compiler.WithLogHandler(h))
            │    └─ helpers.SetupLogger(h, "<engine>", "Compiler")
            └─ engines/<engine>/evaluator.New(h, execUnit)
                 └─ helpers.SetupLogger(h, "<engine>", "Evaluator")
                      and execResult uses "<engine>.execResult"

platform/script.NewExecutableUnit(h, ...) — separately:
  └─ helpers.SetupLogger(h, "script", "ExecutableUnit")
```

Every leaf is `helpers.SetupLogger`. Every layer accepts nil. Every layer's
nil-path inherits from `slog.Default()`.

---

## How we got here (history)

Four PRs (filed against issues #85, #99, #111, #112) converged the package
on the philosophy above:

| Issue | PR | What changed |
|---|---|---|
| #85 | #105 | Made `slog.Handler` optional in engine subpackages; introduced `helpers.SetupLogger` as the canonical fallback. |
| #99 | #110 | Replaced the last three `slog.NewTextHandler(os.Stdout, nil)` fallbacks (in `engines/*/evaluator/response.go`) with `SetupLogger`. Dropped three stale logging-related TODOs. |
| #111 | #113 | Relaxed `WithLogHandler(nil)` and `WithLogger(nil)` across the three engines from "errors on nil" to true no-op (skip set, skip clear-of-the-other-field). |
| #112 | #114 | Dropped the six `cfg.handler != nil` guards in `polyscript.go` and `engines/*/new.go` that existed solely to work around the pre-#111 nil-rejection. Tightened the top-level `WithLogHandler` godoc. |

If you're considering adding a new construction site that takes an
`slog.Handler`, the rule is simple: **call `helpers.SetupLogger` with
`(engineName, structName)`. Don't write a nil-check. Don't construct
fallback handlers. Don't panic.**
