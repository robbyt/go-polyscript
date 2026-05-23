# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Generic top-level constructor `polyscript.New[E]` covering all three engines
  with a unified `Source` and `Option[E]` API, replacing twelve legacy
  `FromXxx` functions. ([#103](https://github.com/robbyt/go-polyscript/pull/103))
- `WithMaxBodySize` option on the HTTP loader, plus `DefaultMaxBodySize`
  (10 MiB) and an `ErrScriptTooLarge` sentinel, so a malicious or
  misbehaving server can no longer exhaust memory through unbounded
  response bodies. ([#107](https://github.com/robbyt/go-polyscript/pull/107))
- Two `Eval()` error-path subtests (`load input data error`,
  `convert to extism format error`) plus a small `mockMapProvider` helper,
  raising `engines/extism/evaluator` coverage from ~93% to 95.5%.
  ([#119](https://github.com/robbyt/go-polyscript/pull/119))
- `docs/LOGGING.md` describing the package's logging philosophy: nil
  handler inherits from `slog.Default()`, options are silent on nil, and
  every construction site routes through `helpers.SetupLogger`.
  ([#115](https://github.com/robbyt/go-polyscript/pull/115))

### Changed
- `slog.Handler` is now optional throughout. A nil handler at any layer
  inherits from `slog.Default()` via the canonical `helpers.SetupLogger`
  helper. The new `polyscript.New[E]` constructor takes no handler
  positionally; the deprecated `FromXxx` constructors still accept one
  in their signatures but nil is now valid.
  ([#105](https://github.com/robbyt/go-polyscript/pull/105),
  [#110](https://github.com/robbyt/go-polyscript/pull/110),
  [#113](https://github.com/robbyt/go-polyscript/pull/113),
  [#114](https://github.com/robbyt/go-polyscript/pull/114))
- Compiler-level `WithLogHandler(nil)` and `WithLogger(nil)` are now true
  no-ops (skip both the field set and the clear-of-the-other-field)
  instead of returning an error. Calling either with nil is equivalent to
  not calling it at all. ([#113](https://github.com/robbyt/go-polyscript/pull/113))
- `WithGlobals` is now additive in both Risor and Starlark — each call
  appends to the compiler's existing globals, deduplicating names already
  present. Order-of-call no longer matters when combined with
  `WithCtxGlobal`. ([#118](https://github.com/robbyt/go-polyscript/pull/118))
- **BREAKING**: Renamed `machine`→`engine` throughout the type-system API.
  `ExecutableContent.GetMachineType()` and `ExecutableUnit.GetMachineType()`
  are now `EngineType()`. `engines/types` exports `ErrInvalidEngineType`,
  `GetEngineTypeFromString`, and `GetEngineTypeFromPath`.
  ([#137](https://github.com/robbyt/go-polyscript/pull/137))
- **BREAKING**: Unexported all six fields on `platform/script.ExecutableUnit`.
  Construct via `NewExecutableUnit` only; read via the existing getters.
  ([#139](https://github.com/robbyt/go-polyscript/pull/139))
- **BREAKING**: Tightened `platform.EvaluatorResponse`. `GetScriptExeID()`
  and `GetExecTime()` are now `ScriptExeID()` and `ExecTime() time.Duration`.
  Added `AsMap() (map[string]any, error)`; on type mismatch the error wraps
  a per-engine `evaluator.ErrAsMapTypeMismatch` sentinel.
  ([#141](https://github.com/robbyt/go-polyscript/pull/141))
- **BREAKING**: `data.Setter.AddDataToContext` no longer takes a variadic.
  Signature is now `AddDataToContext(ctx, data map[string]any) (context.Context, error)`.
  Callers with multiple sources must merge them first.

### Deprecated
- The twelve legacy top-level constructors (`FromRisorFile`,
  `FromRisorString`, `FromStarlarkFile`, `FromStarlarkString`,
  `FromExtismFile`, `FromExtismBytes`, and their `*WithData` variants).
  Each carries a `// Deprecated:` doc comment pointing to the
  `polyscript.New[E]` equivalent.
  ([#103](https://github.com/robbyt/go-polyscript/pull/103))

### Removed
- **BREAKING**: Removed four exported mock types (`loader.MockLoader` with
  `NewMockLoaderWithContent`, `script.MockCompiler`,
  `script.MockExecutableContent`, and the entire `engines/mocks` package).
  `testify` is no longer reachable from any production import path.
  ([#138](https://github.com/robbyt/go-polyscript/pull/138))
- The redundant `cfg.handler != nil` guards in `polyscript.go` and
  `engines/{risor,starlark,extism}/new.go` that existed solely to work
  around the pre-#113 nil-rejection in `WithLogHandler`. Each pair of
  lines collapses to one unconditional append.
  ([#114](https://github.com/robbyt/go-polyscript/pull/114))
- The `slog.NewTextHandler(os.Stdout, …)` fallbacks in three
  `engines/*/evaluator/response.go` `newEvalResult` constructors. The
  library no longer writes to stdout on its own when given a nil handler.
  ([#110](https://github.com/robbyt/go-polyscript/pull/110))
- The dead `url.Parse(u.String())` round-trip in
  `internal/helpers/requestToMap.go:resolveURL`. The unexported helper's
  signature simplified from `(*url.URL, error)` to `*url.URL`.
  ([#118](https://github.com/robbyt/go-polyscript/pull/118))

### Documentation
- Documented the intentional Risor vs Starlark divergence for
  callable-returning scripts: Risor errors, Starlark auto-invokes. New
  "Script Return Value Handling" section in `engines/README.md`.
  ([#140](https://github.com/robbyt/go-polyscript/pull/140))

### Fixed
- `RequestToMap` no longer mutates the caller's `*http.Request`. The URL
  is now resolved through a local sentinel (`resolveURL`) and the body is
  read without write-back. Body remains consume-once, documented in the
  godoc. ([#106](https://github.com/robbyt/go-polyscript/pull/106))
- `FixJSONNumberTypes` now converts bare top-level `json.Number` values.
  Previously a WASM module returning a plain integer at the top level
  (e.g. `42`) leaked `json.Number("42")` to the caller's
  `result.Interface()` instead of arriving as `int(42)` — the recursive
  walker's leaf rule now applies at the root.
  ([#117](https://github.com/robbyt/go-polyscript/pull/117))

## [0.7.0] - 2026-05-04

### Changed
- **BREAKING**: Upgraded the Risor engine to v2. This is the first
  release that no longer supports the Risor v1 language; scripts must
  be updated. ([#74](https://github.com/robbyt/go-polyscript/pull/74))
- Bumped minimum Go version to 1.26.2.
  ([#77](https://github.com/robbyt/go-polyscript/pull/77))
- Extracted `loadInputData` and `AddDataToContext` helpers into the
  shared `platform/data` package, removing engine-by-engine duplication.
  ([#82](https://github.com/robbyt/go-polyscript/pull/82))
- Simplified `FixJSONNumberTypes`; dropped the fragile field-name
  heuristic in favor of straight type-driven conversion.
  ([#83](https://github.com/robbyt/go-polyscript/pull/83))
- Updated `go.starlark.net` to digest `b62fd89`, then `fadfc96`.
  ([#75](https://github.com/robbyt/go-polyscript/pull/75),
  [#76](https://github.com/robbyt/go-polyscript/pull/76))
- Updated `SonarSource/sonarqube-scan-action` to v8.
  ([#79](https://github.com/robbyt/go-polyscript/pull/79))
- Misc dependency refresh.
  ([#80](https://github.com/robbyt/go-polyscript/pull/80))

### Fixed
- Starlark goroutine leak in `exec()` resolved by switching to
  `context.AfterFunc`.
  ([#81](https://github.com/robbyt/go-polyscript/pull/81))

## [0.6.0] - 2026-02-10

> Note: this is the last release supporting the Risor v1 language. The
> next release (`0.7.0`) upgrades to Risor v2, which is incompatible.

### Changed
- Bumped minimum Go version to 1.25.4, then 1.25.5.
  ([#63](https://github.com/robbyt/go-polyscript/pull/63),
  [#71](https://github.com/robbyt/go-polyscript/pull/71))
- Adjusted lint config and resolved several lint warnings in tests.
  ([#64](https://github.com/robbyt/go-polyscript/pull/64))
- Updated `github.com/tetratelabs/wazero` to v1.10.1, then v1.11.0.
  ([#66](https://github.com/robbyt/go-polyscript/pull/66),
  [#69](https://github.com/robbyt/go-polyscript/pull/69))
- Updated `actions/checkout` to v6.
  ([#67](https://github.com/robbyt/go-polyscript/pull/67))
- Updated `golangci/golangci-lint-action` to v9.
  ([#62](https://github.com/robbyt/go-polyscript/pull/62))
- Updated `SonarSource/sonarqube-scan-action` to v7.
  ([#68](https://github.com/robbyt/go-polyscript/pull/68))
- Updated `go.starlark.net` digest several times: `6d2315c`, `7849196`,
  `be02852`, `15019ee`, `3fee463`.
  ([#60](https://github.com/robbyt/go-polyscript/pull/60),
  [#61](https://github.com/robbyt/go-polyscript/pull/61),
  [#65](https://github.com/robbyt/go-polyscript/pull/65),
  [#70](https://github.com/robbyt/go-polyscript/pull/70),
  [#72](https://github.com/robbyt/go-polyscript/pull/72))

## [0.5.0] - 2025-10-19

### Changed
- Bumped minimum Go version to 1.25.3.
- Upgraded `google.golang.org/protobuf`.
  ([#59](https://github.com/robbyt/go-polyscript/pull/59))

[Unreleased]: https://github.com/robbyt/go-polyscript/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/robbyt/go-polyscript/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/robbyt/go-polyscript/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/robbyt/go-polyscript/compare/v0.4.0...v0.5.0
