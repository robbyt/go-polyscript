## Benchmarking VMs, and general optimization

This directory contains benchmarking tools and historical benchmark records to track performance 
characteristics of go-polyscript. These benchmarks serve as both documentation of optimization 
effectiveness and protection against performance regressions. The data collected here demonstrates 
the impact of various performance optimizations implemented throughout the package.

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
enrichedCtx, _ := evaluator.PrepareContext(ctx, inputData)

// EVALUATE (can happen on system B)
result, _ := evaluator.Eval(enrichedCtx)
```

### 3. Provider Selection

The benchmarks also show performance differences between `data.Provider` implementations:

- **StaticProvider**: Fastest overall - use when runtime data is not needed, and input data is static
- **ContextProvider**: Needed for request-specific data that varies per execution, data stored in local memory
- **CompositeProvider**: Meta-provider, enabling multiple `data.Provider` objects in a series

### 4. VM Selection Trade-offs

Performance characteristics vary by VM implementation:

- **Risor**: Fast general-purpose scripting with broad Go compatibility
- **Starlark**: Excellent for configuration processing with Python-like syntax
- **Extism/WASM**: Best security isolation with pre-compiled modules

### Running Benchmarks

To benchmark go-polyscript performance in your environment:

```bash
# Run all benchmarks and generate reports
make bench

# Run specific benchmark pattern
./benchmark.sh BenchmarkEvaluationPatterns

# Quick benchmark without reports
make bench-quick

# Benchmark with specific iterations
go test -bench=BenchmarkVMComparison -benchmem -benchtime=10x ./engine
```
