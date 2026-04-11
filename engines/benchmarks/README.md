# Benchmark Documentation

This directory contains benchmarking tools and historical benchmark records for go-polyscript performance analysis. Benchmarks serve as documentation of optimization effectiveness and protection against performance regressions.

## Performance Optimization Patterns

The numbers below are from `BenchmarkEvaluationPatterns`, `BenchmarkDataProviders`, and `BenchmarkEngineComparison` on an Apple M5 Max (darwin/arm64) using the trivial greeting script in [`benchmark_test.go`](./benchmark_test.go). Your absolute numbers will differ, but the relative deltas are the signal. See [`results/latest.txt`](./results/latest.txt) for the authoritative most-recent run.

### 1. Compile Once, Run Many Times

Reusing a compiled evaluator is **~2.25x faster** than recompiling the script on every call, and allocates **~44% less memory** / **~65% fewer objects**:

| Pattern              |   ns/op | B/op    | allocs/op |
|----------------------|--------:|--------:|----------:|
| SingleExecution      |  48,323 | 120,751 |       713 |
| CompileOnceRunMany   |  21,435 |  67,682 |       252 |

```go
// COMPILATION PHASE (expensive, do once)
evaluator, err := polyscript.FromRisorString(script, options...)

// EXECUTION PHASE (cheap, do many times)
result1, _ := evaluator.Eval(ctx1)
result2, _ := evaluator.Eval(ctx2)
result3, _ := evaluator.Eval(ctx3)
```

The single-execution path pays parser + compiler + globals-validation cost per call. If you're running the same script more than once, create the evaluator once and hold onto it.

### 2. Data Preparation Separation

For distributed architectures, separate data preparation from evaluation to improve system architecture design:

```go
// PREPARE DATA (can happen on system A)
enrichedCtx, _ := evaluator.AddDataToContext(ctx, inputData)

// EVALUATE (can happen on system B)
result, _ := evaluator.Eval(enrichedCtx)
```

### 3. Provider Performance Comparison

On `BenchmarkDataProviders` (Risor engine), all three `data.Provider` implementations land within ~3.5% of each other — raw provider throughput is **not** a meaningful selection criterion. Choose based on data flow shape, not speed:

| Provider           |   ns/op | B/op   | allocs/op |
|--------------------|--------:|-------:|----------:|
| StaticProvider     |  22,354 | 67,382 |       251 |
| ContextProvider    |  21,604 | 66,283 |       243 |
| CompositeProvider  |  21,966 | 67,382 |       253 |

- **StaticProvider** — data is fixed at evaluator creation (config, constants, feature flags).
- **ContextProvider** — data varies per-call; carried via `context.Context`.
- **CompositeProvider** — the backing store for the 2-step pattern (static config + dynamic per-request data).

### 4. Script Engine Performance Characteristics

On the same greeting script, `BenchmarkEngineComparison` shows Starlark is **~5.4x faster** than Risor for raw per-call overhead on small scripts, with **~8.6x less memory** and **~3.5x fewer allocations**:

| Engine    |   ns/op | B/op   | allocs/op |
|-----------|--------:|-------:|----------:|
| Risor     |  22,121 | 67,382 |       251 |
| Starlark  |   4,119 |  7,806 |        71 |

Caveats: the benchmark script is trivial (two variables + a map literal), so this measures per-call VM setup more than real work. Interpret by use case, not by the numbers alone:

- **Starlark** — lowest per-call overhead; deterministic, Python-like, designed for configuration. Limited language capabilities (no loops as iteration, no stdlib I/O). Best when you have many fast, simple evaluations.
- **Risor** — richer stdlib (`math`, `rand`, `regexp` in v2), TypeScript-aligned syntax (arrow functions, `try/catch`, optional chaining), friendlier for general scripting and data munging. Pays a higher fixed per-call cost.
- **Extism/WASM** — language-agnostic isolation via pre-compiled modules. Choose when you need to run untrusted code, support multiple languages, or get true sandbox isolation.

## Running Benchmarks

To benchmark go-polyscript performance in your environment:

```bash
# Run all benchmarks and generate reports
make bench

# Run specific benchmark pattern
./benchmarks/run.sh BenchmarkEvaluationPatterns

# Quick benchmark without reports
make bench-quick

# Benchmark with specific iterations
go test -bench=BenchmarkEngineComparison -benchmem -benchtime=10x ./engines/benchmarks
```

## Historical Results

Benchmark results are stored in the `results/` directory with timestamps. This allows tracking performance changes over time and identifying performance regressions:

- `benchmark_YYYY-MM-DD_HH-MM-SS.json` - Raw benchmark data
- `benchmark_YYYY-MM-DD_HH-MM-SS.txt` - Human-readable summary
- `comparison.txt` - Comparison of current results with previous runs
- `latest.txt` - Most recent benchmark summary

## Benchmarking Infrastructure

The benchmarking infrastructure in go-polyscript automatically:

1. Runs comprehensive benchmarks across all script engines
2. Compares execution patterns (one-time vs. compile-once-run-many)
3. Measures data provider performance differences
4. Records results for historical comparison
5. Generates human-readable reports

For more detailed examples of optimized script usage patterns, see the [examples directory](../examples/).
