# Benchmark Documentation

This directory contains benchmarking tools and historical benchmark records for go-polyscript performance analysis. Benchmarks serve as documentation of optimization effectiveness and protection against performance regressions.

## Performance Optimization Patterns

### 1. Compile Once, Run Many Times

The benchmarks show that pre-loading and parsing scripts is ~35% faster than recompiling for each execution:

```go
// COMPILATION PHASE (expensive, do once)
evaluator, err := polyscript.FromRisorString(script, options...)

// EXECUTION PHASE (inexpensive, do many times)
result1, _ := evaluator.Eval(ctx1)
result2, _ := evaluator.Eval(ctx2)
result3, _ := evaluator.Eval(ctx3)
```

### 2. Data Preparation Separation

For distributed architectures, separate data preparation from evaluation to improve system architecture design:

```go
// PREPARE DATA (can happen on system A)
enrichedCtx, _ := evaluator.AddDataToContext(ctx, inputData)

// EVALUATE (can happen on system B)
result, _ := evaluator.Eval(enrichedCtx)
```

### 3. Provider Performance Comparison

The benchmarks show performance differences between `data.Provider` implementations:

- **StaticProvider**: Fastest overall (~5-10% faster than other providers) - use when input data is static
- **ContextProvider**: Needed for request-specific data that varies per execution
- **CompositeProvider**: Small overhead but enables both static configuration and dynamic request data

### 4. Script Engine Performance Characteristics

Performance characteristics vary significantly by implementation:

- **Risor**: Generally fastest for general-purpose scripting with good Go interoperability
- **Starlark**: Optimized for configuration processing with Python-like syntax
- **Extism/WASM**: Best for security isolation with pre-compiled modules

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
go test -bench=BenchmarkVMComparison -benchmem -benchtime=10x ./engine
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
