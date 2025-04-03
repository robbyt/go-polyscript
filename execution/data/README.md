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
   - Added to the context via `PrepareContext` method
   - Stored under the `"input_data"` key (previously `"script_data"`) in the context

Both types of data are made available to scripts as part of the top-level `ctx` variable, which is injected into the script's global scope.

### Data Flow

```
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Static Data     │    │   Dynamic Data    │    │     Provider      │
│                   │    │                   │    │                   │
│                   │    │                   │    │  GetData()        │
│ - Config values   │    │ - Request params  │    │  AddDataToContext()│
│ - Constants       │    │ - User inputs     │    │                   │
└─────────┬─────────┘    └─────────┬─────────┘    └─────────┬─────────┘
          │                        │                        │
          │                        │                        │
          ▼                        ▼                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Context                                     │
│                                                                     │
│  Data stored under constants.EvalData key with structure:           │
│  {                                                                  │
│    // Static data (from StaticProvider) is at top level             │
│    "config_value1": ...,                                            │
│    "config_value2": ...,                                            │
│                                                                     │
│    // Dynamic data (from ContextProvider) is nested                 │
│    "input_data": { ... },  // Dynamic data added at runtime         │
│    "request": { ... },     // HTTP request data (if available)      │
│  }                                                                  │
└─────────────────────────────────────┬───────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        VM Execution                                  │
│                                                                     │
│  - VM implementations access data through the Provider interface    │
│  - Each VM makes the data available as a global `ctx` variable      │
│                                                                     │
│  Script accesses via top-level `ctx` variable:                      │
│    ctx["config_value1"]                 // Static data (direct)     │
│    ctx["input_data"]["user_input"]      // Dynamic data (nested)    │
│    ctx["request"]["params"]             // HTTP request data        │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Access Patterns in Scripts

Scripts must handle different data access patterns based on the Provider:

1. **Top-level access** for static data from `StaticProvider`:
   ```
   name := ctx["name"]  // Direct access
   ```

2. **Nested access** for dynamic data from `ContextProvider`:
   ```
   name := ctx["input_data"]["name"]  // Nested under input_data
   ```

3. **Hybrid approach** for maximum compatibility:
   ```
   var name = ""
   if ctx["name"] != nil {
       name = ctx["name"]  // Try direct access first
   } else if ctx["input_data"] != nil && ctx["input_data"]["name"] != nil {
       name = ctx["input_data"]["name"]  // Fall back to nested access
   }
   ```

## Providers

Providers control how data is stored and accessed for script execution:

- **StaticProvider**: Returns predefined data at top level, useful for configuration and static values
- **ContextProvider**: Retrieves data from context using a specific key, stores dynamic data under `input_data`
- **CompositeProvider**: Chains multiple providers, combining static and dynamic data sources

A common pattern is to combine a StaticProvider for configuration with a ContextProvider for runtime data:

```go
// Static configuration values (available at top level)
staticProvider := data.NewStaticProvider(map[string]any{
    "config": "value",
})

// Runtime data provider (data will be nested under input_data)
ctxProvider := data.NewContextProvider(constants.EvalData)

// Combine them for unified access
compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)
```

## Data Preparation and Evaluation

The `PrepareContext` method (defined in the `EvalDataPreparer` interface) allows for a separation between:

1. Preparing the data and enriching the context
2. Evaluating the script with the prepared context

This pattern enables distributed architectures where:
- Data preparation occurs on one system (e.g., web server)
- Evaluation occurs on another system (e.g., worker node)

## Best Practices

1. Use a `ContextProvider` with the `constants.EvalData` key for dynamic data
2. Use a `StaticProvider` for configuration and other static data
3. Use `CompositeProvider` when you need to combine static and dynamic data sources
4. For maximum compatibility, scripts should check both direct and nested data access patterns
5. In new code, use the `input_data` key for dynamic data (replacing the older `script_data`)
6. Avoid relying on specific data locations without checking alternative locations
7. Use the data preparation pattern for complex or distributed architectures