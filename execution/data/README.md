# Data Management in go-polyscript

This package handles the flow of data to and from script evaluations in go-polyscript. It defines how data is stored, accessed, and passed between Go and script environments.

## Key Concepts

### Types of Data

There are two main types of data in go-polyscript:

1. **Script Data (static)**: 
   - Defined once at script compile/load time
   - Remains constant for all executions of a compiled script
   - Often contains configuration values, utility functions, or constants
   - Stored under the `"script_data"` key in the context

2. **Evaluation Data (dynamic)**:
   - Provided fresh for each script execution
   - Contains runtime-specific data like request parameters, user inputs, etc.
   - Changes between different executions of the same script
   - Added to the context via `PrepareContext` method

Both types of data are made available to scripts as part of the top-level `ctx` variable, which is injected into the script's global scope.

### Data Flow

```
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Script Data     │    │   Runtime Data    │    │     Provider      │
│   (static)        │    │   (dynamic)       │    │                   │
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
│    "script_data": { ... },  // Static data from script compilation  │
│    "request": { ... },      // HTTP request data (if available)     │
│    ...                      // Other runtime data                   │
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
│    ctx["script_data"]["config"]                                     │
│    ctx["request"]["params"]                                         │
└─────────────────────────────────────────────────────────────────────┘
```

## Providers

Providers control how data is stored and accessed for script execution:

- **StaticProvider**: Returns predefined data, useful for fixed values and testing
- **ContextProvider**: Retrieves data from context using a specific key, useful for dynamic data
- **CompositeProvider**: Chains multiple providers, combining their results

## Data Preparation and Evaluation

The `PrepareContext` method (defined in the `EvalDataPreparer` interface) allows for a separation between:

1. Preparing the data and enriching the context
2. Evaluating the script with the prepared context

This pattern enables distributed architectures where:
- Data preparation occurs on one system (e.g., web server)
- Evaluation occurs on another system (e.g., worker node)

The `PrepareContextHelper` function provides a consistent implementation that all VM implementations use.

## Best Practices

1. Use a `ContextProvider` with the `constants.EvalData` key for dynamic data
2. Keep `script_data` small and focused on configuration
3. Use the data preparation pattern for complex or distributed architectures
4. Access data consistently in scripts via the `ctx` variable
5. Use `CompositeProvider` when you need to combine data from multiple sources