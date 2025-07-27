# Data Management in go-polyscript

This package handles the flow of data to and from script evaluations in go-polyscript. It defines how data is stored, accessed, and passed between Go and script environments.

## Key Concepts

### The 2-Step Data Flow Pattern

go-polyscript uses a 2-Step Data Flow Pattern that enables the "compile once, run many" performance optimization:

**Step 1: Evaluator Creation (Compile Once)**
- Static data is provided when creating the evaluator
- This static data is embedded in the evaluator via `StaticProvider`
- The script is compiled once with this static data available
- The evaluator is ready for multiple executions

**Step 2: Runtime Execution (Run Many)**
- Dynamic per-request data is added via `AddDataToContext()`
- This dynamic data is stored in the context via `ContextProvider`
- During `Eval()`, the `CompositeProvider` merges static and dynamic data
- The merged data is passed to the script engine

### Types of Data

1. **Static Data**: 
   - Configuration values, constants, feature flags
   - Defined once at evaluator creation time
   - Remains constant across all executions
   - Embedded in the evaluator for optimal performance

2. **Dynamic Data**:
   - Per-request data: user info, HTTP requests, timestamps
   - Provided fresh for each script execution
   - Added to context via `AddDataToContext()` before each `Eval()`
   - Stored in context under the `constants.EvalData` key

Both types of data are merged and made available to scripts through the `ctx` object (in Risor/Starlark) or as direct JSON (in Extism).

### Data Flow - The 2-Step Data Flow Pattern

```
STEP 1: EVALUATOR CREATION (Compile Once)
┌─────────────────────┐
│   Static Data       │  Configuration values, constants, feature flags
│                     │  that remain the same across all executions
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  StaticProvider     │  Stores static data
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐     ┌─────────────────────┐
│  CompositeProvider  │←────│  ContextProvider    │  Ready to receive
│                     │     │  (empty)            │  dynamic data later
└──────────┬──────────┘     └─────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        ExecutableUnit                               │
│  - Contains compiled script                                         │
│  - Embeds CompositeProvider with static data                        │
│  - Ready for multiple executions                                    │
└─────────────────────────────────────────────────────────────────────┘

STEP 2: RUNTIME EXECUTION (Run Many)
┌─────────────────────┐
│   Dynamic Data      │  Per-request data: user info, HTTP requests,
│                     │  timestamps, session data, etc.
└──────────┬──────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│              evaluator.AddDataToContext(ctx, dynamicData)           │
│  - Stores dynamic data in context under constants.EvalData key      │
│  - Returns enriched context                                         │
└──────────┬──────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────┐
│   Enriched Context  │  Contains dynamic data
└──────────┬──────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      evaluator.Eval(enrichedCtx)                    │
│                                                                     │
│  CompositeProvider.GetData() merges:                                │
│  1. Static data from StaticProvider (embedded at creation)          │
│  2. Dynamic data from ContextProvider (retrieved from context)      │
│                                                                     │
│  Merged data passed to engine for script execution                  │
└──────────┬──────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Engine Execution                            │
│                                                                     │
│  Scripts access merged input data through engine-specific patterns: │
│  - Risor/Starlark: ctx["static_config"], ctx["dynamic_user"]        │
│  - Extism/WASM: Direct JSON with both static and dynamic data       │
└─────────────────────────────────────────────────────────────────────┘
```

### Code Example of the 2-Step Data Flow Pattern

```go
// STEP 1: Create evaluator with static data (happens once)
staticData := map[string]any{
    "app_version": "1.0.0",
    "environment": "production",
    "config": map[string]any{
        "timeout": 30,
        "max_retries": 3,
    },
}

evaluator, err := polyscript.FromRisorStringWithData(
    scriptContent,
    staticData,        // Embedded via StaticProvider
    logger.Handler(),
)

// STEP 2: Add dynamic data and execute (happens many times)
for _, request := range httpRequests {
    // Fresh dynamic data for this execution
    dynamicData := map[string]any{
        "user_id": request.UserID,
        "request": request,
        "timestamp": time.Now().Unix(),
    }

    // Add dynamic data to context
    enrichedCtx, err := evaluator.AddDataToContext(ctx, dynamicData)
    if err != nil {
        return err
    }

    // Execute with merged static and dynamic data
    result, err := evaluator.Eval(enrichedCtx)
    if err != nil {
        return err
    }

    // Process result...
}
```

## Data Access Patterns in Scripts

Script data access depends on the engine being used. For detailed information about how each engine processes and exposes data to scripts, see the [engines documentation](../engines/README.md#engine-specific-data-handling).

**Key principle:** When providing data to scripts, use explicit keys in your data maps for clarity:

```go
// Add HTTP request data with explicit key
enrichedCtx, _ := evaluator.AddDataToContext(ctx, map[string]any{
    "request": httpRequest
})
```

## Providers

Providers control how data is stored and accessed for script execution:

- **StaticProvider**: Returns predefined data embedded at evaluator creation time
- **ContextProvider**: Retrieves dynamic data from context at runtime
- **CompositeProvider**: Merges data from multiple providers during evaluation

### How Providers Work in the 2-Step Data Flow Pattern

When you use convenience functions like `FromRisorStringWithData()`, they automatically create:
1. A `StaticProvider` with your static data
2. A `ContextProvider` (initially empty) for dynamic data
3. A `CompositeProvider` that merges both

During evaluation, the `CompositeProvider`:
- Calls `GetData()` on each provider in sequence
- Merges the results (later providers override earlier ones)
- Passes the merged data to the script engine

Example of the provider chain:

```go
// Static configuration values
staticProvider := data.NewStaticProvider(map[string]any{
    "config": "value",
})

// Dynamic data provider for thread-safe per-request data
ctxProvider := data.NewContextProvider(constants.EvalData)

// Combine them for unified access
compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)
```

## Data Preparation and Evaluation

The `AddDataToContext` method (defined in the `data.Setter` interface) allows for a separation between:

1. Preparing the data and enriching the context
2. Evaluating the script with the prepared context

This pattern enables distributed architectures where:
- Data preparation occurs on one system (e.g., web server)
- Evaluation occurs on another system (e.g., worker node)

## Best Practices

1. Use a `ContextProvider` with the `constants.EvalData` key for dynamic per-request data
2. Use a `StaticProvider` for configuration and other static data
3. Use `CompositeProvider` when you need to merge static and dynamic data sources
4. Always use explicit keys when adding data with `AddDataToContext(ctx, map[string]any{"key": value})`
5. For HTTP requests, wrap them with a descriptive key: `map[string]any{"request": httpRequest}`