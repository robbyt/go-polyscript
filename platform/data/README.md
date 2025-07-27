# Data Management in go-polyscript

This package handles the flow of data to and from script evaluations in go-polyscript. It defines how data is stored, accessed, and passed between Go and script environments.

## Key Concepts

### Types of Data

There are two main types of data in go-polyscript:

1. **Static Data**: 
   - Defined once at script compile/load time
   - Remains constant for all executions of a compiled script
   - Often contains configuration values, utility functions, or constants
   - Provided via `StaticProvider` and accessed directly at the top level of the `ctx` variable

2. **Dynamic Data**:
   - Provided fresh for each script execution
   - Contains runtime-specific data like request parameters, user inputs, etc.
   - Changes between different executions of the same script
   - Added to the context via `AddDataToContext` method
   - Stored directly at the root level of the context

Both types of data are made available to scripts, though the exact format depends on the engine (see [engines documentation](../engines/README.md#engine-specific-data-handling) for details).

### Data Flow

```
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Static Data     │    │   Dynamic Data    │    │     Provider      │
│                   │    │                   │    │                   │
│                   │    │                   │    │ GetData()         │
│ - Config values   │    │ - Request params  │    │ AddDataToContext()│
│ - Constants       │    │ - User inputs     │    │                   │
└─────────┬─────────┘    └─────────┬─────────┘    └─────────┬─────────┘
          │                        │                        │
          │                        │                        │
          ▼                        ▼                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Context                                    │
│                                                                     │
│  Data stored under constants.EvalData key with structure:           │
│  {                                                                  │
│    // All data is at the top level of the context                   │
│    "config_value1": ...,   // Static data (from StaticProvider)     │
│    "config_value2": ...,   // Static data (from StaticProvider)     │
│    "user_data": ...,       // Dynamic data (user-provided)          │
│    "request": { ... },     // HTTP request data (if available)      │
│  }                                                                  │
└─────────────────────────────────────┬───────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Engine Execution                                │
│                                                                     │
│ - Engine implementations access data through the Provider interface │
│ - Each engine exposes data differently to scripts (see engines docs)│
│                                                                     │
│  Script data access (format varies by engine):                      │
│    - Risor/Starlark: ctx["config_value1"], ctx["user_data"]         │
│    - Extism/WASM: Direct JSON structure passed to WASM module       │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Access Patterns in Scripts

Script data access depends on the engine being used:

**Risor/Starlark engines:**
```
// Static configuration
config := ctx["config_name"]

// Dynamic user data
userData := ctx["user_data"]

// HTTP request data
requestMethod := ctx["request"]["Method"]
urlPath := ctx["request"]["URL_Path"]
requestBody := ctx["request"]["Body"]
```

**Extism/WASM engine:**
Data is passed directly as JSON to the WASM module without a `ctx` wrapper. See [engines documentation](../engines/README.md#engine-specific-data-handling) for details.

When providing data to scripts, use explicit keys in your data maps for clarity:

```go
// Add HTTP request data with explicit key
enrichedCtx, _ := evaluator.AddDataToContext(ctx, map[string]any{
    "request": httpRequest
})
```

## Providers

Providers control how data is stored and accessed for script execution:

- **StaticProvider**: Returns predefined data, useful for configuration and static values
- **ContextProvider**: Used for storing and retrieving thread-safe dynamic runtime data
- **CompositeProvider**: Chains multiple providers, combining static and dynamic data sources

A common pattern is to combine a StaticProvider for configuration with a ContextProvider for runtime data:

```go
// Static configuration values
staticProvider := data.NewStaticProvider(map[string]any{
    "config": "value",
})

// Runtime data provider for thread-safe per-request data
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

1. Use a `ContextProvider` with the `constants.EvalData` key for dynamic request-specific data
2. Use a `StaticProvider` for configuration and other static data
3. Use `CompositeProvider` when you need to combine static and dynamic data sources
4. Always use explicit keys when adding data with `AddDataToContext(ctx, map[string]any{"key": value})`
5. For HTTP requests, wrap them with a descriptive key: `map[string]any{"request": httpRequest}`