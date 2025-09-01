package engines

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript"
	extismEngine "github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	risorEngine "github.com/robbyt/go-polyscript/engines/risor"
	starlarkEngine "github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineDataHandlingIntegration tests that identical data structures work across all engines
// but demonstrates how each engine exposes the data differently to scripts
func TestEngineDataHandlingIntegration(t *testing.T) {
	t.Parallel()

	// Common test data structure
	testData := map[string]any{
		"name":    "Integration Test",
		"version": "1.0.0",
		"config": map[string]any{
			"debug":   true,
			"timeout": 30,
		},
		"tags": []any{
			"test",
			"integration",
			"multi-engine",
		}, // Use []any for Starlark compatibility
	}

	// Create a test logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Reduce noise in tests
	})

	t.Run(
		"risor_engine",
		func(t *testing.T) {
			// Risor script that accesses data via ctx variable
			risorScript := `
// Access data through ctx variable
name := ctx["name"]
version := ctx["version"]
debug := ctx["config"]["debug"]
timeout := ctx["config"]["timeout"]
tags := ctx["tags"]

// Create result
{
	"engine": "risor",
	"name": name,
	"version": version,
	"debug": debug,
	"timeout": timeout,
	"tag_count": len(tags),
	"first_tag": tags[0]
}
`

			// Create evaluator with test data
			evaluator, err := polyscript.FromRisorStringWithData(
				risorScript,
				testData,
				handler,
			)
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(t.Context())
			require.NoError(t, err)

			// Verify results
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "risor", resultMap["engine"])
			assert.Equal(t, "Integration Test", resultMap["name"])
			assert.Equal(t, "1.0.0", resultMap["version"])
			assert.Equal(t, true, resultMap["debug"])
			assert.Equal(t, int64(30), resultMap["timeout"]) // Risor converts to int64
			assert.Equal(t, int64(3), resultMap["tag_count"])
			assert.Equal(t, "test", resultMap["first_tag"])
		},
	)

	t.Run(
		"starlark_engine",
		func(t *testing.T) {
			// Starlark script that accesses data via ctx dictionary
			starlarkScript := `
# Access data through ctx dictionary
name = ctx["name"]
version = ctx["version"]
debug = ctx["config"]["debug"]
timeout = ctx["config"]["timeout"]
tags = ctx["tags"]

# Create result dictionary
result = {
	"engine": "starlark",
	"name": name,
	"version": version,
	"debug": debug,
	"timeout": timeout,
	"tag_count": len(tags),
	"first_tag": tags[0]
}

# Return the result
_ = result
`

			// Create evaluator with test data
			evaluator, err := polyscript.FromStarlarkStringWithData(
				starlarkScript,
				testData,
				handler,
			)
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(t.Context())
			require.NoError(t, err)

			// Verify results
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "starlark", resultMap["engine"])
			assert.Equal(t, "Integration Test", resultMap["name"])
			assert.Equal(t, "1.0.0", resultMap["version"])
			assert.Equal(t, true, resultMap["debug"])
			assert.Equal(t, int64(30), resultMap["timeout"]) // Starlark converts to int64
			assert.Equal(t, int64(3), resultMap["tag_count"])
			assert.Equal(t, "test", resultMap["first_tag"])
		},
	)

	t.Run("extism_engine", func(t *testing.T) {
		// For Extism, we need to structure the data to match what the WASM module expects
		// The test WASM module expects {"input": "some_value"} structure
		extismData := map[string]any{
			"input":   "Integration Test",
			"version": "1.0.0",
			"config": map[string]any{
				"debug":   true,
				"timeout": 30,
			},
			"tags": []any{"test", "integration", "multi-engine"},
		}

		// Create evaluator with WASM-specific data structure
		evaluator, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			extismData,
			handler,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)

		// Execute
		result, err := evaluator.Eval(t.Context())
		require.NoError(t, err)

		// Verify results - the greet function returns {"greeting": "Hello, <input>!"}
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		greeting, exists := resultMap["greeting"]
		require.True(t, exists)
		assert.Equal(t, "Hello, Integration Test!", greeting)
	})
}

// TestEngineDataHandlingWithDynamicData tests dynamic data handling across engines
func TestEngineDataHandlingWithDynamicData(t *testing.T) {
	t.Parallel()

	// Static configuration data
	staticData := map[string]any{
		"app_name":    "TestApp",
		"app_version": "2.0.0",
		"environment": "test",
	}

	// Dynamic runtime data
	runtimeData := map[string]any{
		"user_id":   "user123",
		"action":    "process_request",
		"timestamp": "2023-01-01T00:00:00Z",
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	t.Run("risor_dynamic_data", func(t *testing.T) {
		risorScript := `
// Access both static and dynamic data
app_name := ctx["app_name"]
user_id := ctx["user_id"]
action := ctx["action"]

{
	"result": "processed",
	"app": app_name,
	"user": user_id,
	"action": action
}
`

		// Create evaluator with static data
		evaluator, err := polyscript.FromRisorStringWithData(
			risorScript,
			staticData,
			handler,
		)
		require.NoError(t, err)

		// Add dynamic data to context
		ctx := t.Context()
		enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
		require.NoError(t, err)

		// Execute with enriched context
		result, err := evaluator.Eval(enrichedCtx)
		require.NoError(t, err)

		// Verify both static and dynamic data are accessible
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "TestApp", resultMap["app"])            // Static data
		assert.Equal(t, "user123", resultMap["user"])           // Dynamic data
		assert.Equal(t, "process_request", resultMap["action"]) // Dynamic data
	})

	t.Run("starlark_dynamic_data", func(t *testing.T) {
		starlarkScript := `
# Access both static and dynamic data
app_name = ctx["app_name"]
user_id = ctx["user_id"]
action = ctx["action"]

result = {
	"result": "processed",
	"app": app_name,
	"user": user_id,
	"action": action
}

_ = result
`

		// Create evaluator with static data
		evaluator, err := polyscript.FromStarlarkStringWithData(
			starlarkScript,
			staticData,
			handler,
		)
		require.NoError(t, err)

		// Add dynamic data to context
		ctx := t.Context()
		enrichedCtx, err := evaluator.AddDataToContext(ctx, runtimeData)
		require.NoError(t, err)

		// Execute with enriched context
		result, err := evaluator.Eval(enrichedCtx)
		require.NoError(t, err)

		// Verify both static and dynamic data are accessible
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "TestApp", resultMap["app"])            // Static data
		assert.Equal(t, "user123", resultMap["user"])           // Dynamic data
		assert.Equal(t, "process_request", resultMap["action"]) // Dynamic data
	})

	t.Run("extism_dynamic_data", func(t *testing.T) {
		// For Extism, we need to structure data for the WASM module
		// Using the greet entrypoint which expects {"input": "value"}
		extismStaticData := map[string]any{
			"input":       "TestApp",
			"app_version": "2.0.0",
			"environment": "test",
		}

		extismRuntimeData := map[string]any{
			"input":  "user123", // This will override the static input
			"action": "process_request",
		}

		// Create evaluator with static data
		evaluator, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			extismStaticData,
			handler,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)

		// Add dynamic data to context
		ctx := t.Context()
		enrichedCtx, err := evaluator.AddDataToContext(ctx, extismRuntimeData)
		require.NoError(t, err)

		// Execute with enriched context
		result, err := evaluator.Eval(enrichedCtx)
		require.NoError(t, err)

		// Verify the WASM module processed the data
		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		greeting, exists := resultMap["greeting"]
		require.True(t, exists)
		// Should use the dynamic data that overrode the static data
		assert.Equal(t, "Hello, user123!", greeting)
	})
}

// TestEngineDataStructureDocumentation demonstrates the documented examples work correctly
func TestEngineDataStructureDocumentation(t *testing.T) {
	t.Parallel()

	// This is the exact example from the README documentation
	data := map[string]any{
		"name":   "World",
		"config": map[string]any{"debug": true},
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	t.Run("risor_documentation_example", func(t *testing.T) {
		script := `
name := ctx["name"]
debug := ctx["config"]["debug"]

{
	"name": name,
	"debug": debug
}
`

		evaluator, err := polyscript.FromRisorStringWithData(script, data, handler)
		require.NoError(t, err)

		result, err := evaluator.Eval(t.Context())
		require.NoError(t, err)

		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "World", resultMap["name"])
		assert.Equal(t, true, resultMap["debug"])
	})

	t.Run("starlark_documentation_example", func(t *testing.T) {
		script := `
name = ctx["name"]
debug = ctx["config"]["debug"]

result = {
	"name": name,
	"debug": debug
}

_ = result
`

		evaluator, err := polyscript.FromStarlarkStringWithData(script, data, handler)
		require.NoError(t, err)

		result, err := evaluator.Eval(t.Context())
		require.NoError(t, err)

		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		assert.Equal(t, "World", resultMap["name"])
		assert.Equal(t, true, resultMap["debug"])
	})

	t.Run("extism_documentation_example", func(t *testing.T) {
		// For Extism, the data structure must match what the WASM module expects
		// The documentation example shows the JSON that gets passed directly
		extismData := map[string]any{
			"input":  "World", // The greet function expects this structure
			"config": map[string]any{"debug": true},
		}

		evaluator, err := polyscript.FromExtismBytesWithData(
			wasmdata.TestModule,
			extismData,
			handler,
			wasmdata.EntrypointGreet,
		)
		require.NoError(t, err)

		result, err := evaluator.Eval(t.Context())
		require.NoError(t, err)

		resultMap, ok := result.Interface().(map[string]any)
		require.True(t, ok)

		// The greet function returns {"greeting": "Hello, <input>!"}
		greeting, exists := resultMap["greeting"]
		require.True(t, exists)
		assert.True(t, strings.Contains(greeting.(string), "World"))
	})
}

// TestDataProviderPatterns tests the data provider patterns from platform/data/README.md
func TestDataProviderPatterns(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	t.Run("context_provider_usage", func(t *testing.T) {
		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"risor_context_provider",
			func(t *testing.T) {
				script := `
config := ctx["config"]
user_data := ctx["user_data"]

{
	"config": config,
	"user_data": user_data
}
`

				// Create evaluator using context provider pattern
				ctxProvider := data.NewContextProvider(constants.EvalData)
				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := risorEngine.NewEvaluator(handler, scriptLoader, ctxProvider)
				require.NoError(t, err)

				// Add data using context provider
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"config": map[string]any{
						"debug":   true,
						"timeout": 30,
					},
					"user_data": map[string]any{
						"name": "TestUser",
						"id":   123,
					},
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				config, exists := resultMap["config"].(map[string]any)
				require.True(t, exists)
				assert.Equal(t, true, config["debug"])
				assert.Equal(t, int64(30), config["timeout"])

				userData, exists := resultMap["user_data"].(map[string]any)
				require.True(t, exists)
				assert.Equal(t, "TestUser", userData["name"])
				assert.Equal(t, int64(123), userData["id"])
			},
		)

		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"starlark_context_provider",
			func(t *testing.T) {
				script := `
config = ctx["config"]
user_data = ctx["user_data"]

result = {
	"config": config,
	"user_data": user_data
}

_ = result
`

				// Create evaluator using context provider pattern
				ctxProvider := data.NewContextProvider(constants.EvalData)
				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := starlarkEngine.NewEvaluator(handler, scriptLoader, ctxProvider)
				require.NoError(t, err)

				// Add data using context provider
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"config": map[string]any{
						"debug":   true,
						"timeout": 30,
					},
					"user_data": map[string]any{
						"name": "TestUser",
						"id":   123,
					},
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				config, exists := resultMap["config"].(map[string]any)
				require.True(t, exists)
				assert.Equal(t, true, config["debug"])
				assert.Equal(t, int64(30), config["timeout"])

				userData, exists := resultMap["user_data"].(map[string]any)
				require.True(t, exists)
				assert.Equal(t, "TestUser", userData["name"])
				assert.Equal(t, int64(123), userData["id"])
			},
		)

		t.Run("extism_context_provider", func(t *testing.T) {
			// Create evaluator using context provider pattern
			ctxProvider := data.NewContextProvider(constants.EvalData)
			scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
			require.NoError(t, err)
			evaluator, err := extismEngine.NewEvaluator(
				handler,
				scriptLoader,
				ctxProvider,
				wasmdata.EntrypointGreet,
			)
			require.NoError(t, err)

			// Add data using context provider - structure for greet function
			ctx := t.Context()
			enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
				"input": "ContextProvider",
				"config": map[string]any{
					"debug": true,
				},
			})
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(enrichedCtx)
			require.NoError(t, err)

			// Verify
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			greeting, exists := resultMap["greeting"]
			require.True(t, exists)
			assert.Equal(t, "Hello, ContextProvider!", greeting)
		})
	})

	t.Run("static_provider_usage", func(t *testing.T) {
		staticData := map[string]any{
			"config": map[string]any{
				"app_name": "TestApp",
				"version":  "1.0.0",
			},
			"constants": map[string]any{
				"max_retries": 3,
				"timeout":     30,
			},
		}

		t.Run("risor_static_provider", func(t *testing.T) {
			script := `
app_name := ctx["config"]["app_name"]
version := ctx["config"]["version"]
max_retries := ctx["constants"]["max_retries"]

{
	"app_name": app_name,
	"version": version,
	"max_retries": max_retries
}
`

			// Create evaluator using static provider
			staticProvider := data.NewStaticProvider(staticData)
			scriptLoader, err := loader.NewFromString(script)
			require.NoError(t, err)
			evaluator, err := risorEngine.NewEvaluator(handler, scriptLoader, staticProvider)
			require.NoError(t, err)

			// Execute without additional context data
			result, err := evaluator.Eval(t.Context())
			require.NoError(t, err)

			// Verify
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "TestApp", resultMap["app_name"])
			assert.Equal(t, "1.0.0", resultMap["version"])
			assert.Equal(t, int64(3), resultMap["max_retries"])
		})

		t.Run("starlark_static_provider", func(t *testing.T) {
			script := `
app_name = ctx["config"]["app_name"]
version = ctx["config"]["version"]
max_retries = ctx["constants"]["max_retries"]

result = {
	"app_name": app_name,
	"version": version,
	"max_retries": max_retries
}

_ = result
`

			// Create evaluator using static provider
			staticProvider := data.NewStaticProvider(staticData)
			scriptLoader, err := loader.NewFromString(script)
			require.NoError(t, err)
			evaluator, err := starlarkEngine.NewEvaluator(handler, scriptLoader, staticProvider)
			require.NoError(t, err)

			// Execute without additional context data
			result, err := evaluator.Eval(t.Context())
			require.NoError(t, err)

			// Verify
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			assert.Equal(t, "TestApp", resultMap["app_name"])
			assert.Equal(t, "1.0.0", resultMap["version"])
			assert.Equal(t, int64(3), resultMap["max_retries"])
		})

		t.Run("extism_static_provider", func(t *testing.T) {
			// Create evaluator using static provider
			staticProvider := data.NewStaticProvider(map[string]any{
				"input": "StaticProvider",
				"config": map[string]any{
					"app_name": "TestApp",
					"version":  "1.0.0",
				},
			})
			scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
			require.NoError(t, err)
			evaluator, err := extismEngine.NewEvaluator(
				handler,
				scriptLoader,
				staticProvider,
				wasmdata.EntrypointGreet,
			)
			require.NoError(t, err)

			// Execute without additional context data
			result, err := evaluator.Eval(t.Context())
			require.NoError(t, err)

			// Verify
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			greeting, exists := resultMap["greeting"]
			require.True(t, exists)
			assert.Equal(t, "Hello, StaticProvider!", greeting)
		})
	})

	t.Run("composite_provider_usage", func(t *testing.T) {
		// Static configuration values
		staticProvider := data.NewStaticProvider(map[string]any{
			"config": map[string]any{
				"app_name": "TestApp",
				"version":  "1.0.0",
			},
		})

		// Runtime data provider for thread-safe per-request data
		ctxProvider := data.NewContextProvider(constants.EvalData)

		// Combine them for unified access
		compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)

		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"risor_composite_provider", func(t *testing.T) {
				script := `
app_name := ctx["config"]["app_name"]
version := ctx["config"]["version"]
user_id := ctx["user_id"]
request_id := ctx["request_id"]

{
	"app_name": app_name,
	"version": version,
	"user_id": user_id,
	"request_id": request_id
}
`

				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := risorEngine.NewEvaluator(handler, scriptLoader, compositeProvider)
				require.NoError(t, err)

				// Add runtime data to context
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"user_id":    "user123",
					"request_id": "req456",
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify both static and dynamic data are accessible
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				assert.Equal(t, "TestApp", resultMap["app_name"])  // Static data
				assert.Equal(t, "1.0.0", resultMap["version"])     // Static data
				assert.Equal(t, "user123", resultMap["user_id"])   // Dynamic data
				assert.Equal(t, "req456", resultMap["request_id"]) // Dynamic data
			})

		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"starlark_composite_provider", func(t *testing.T) {
				script := `
app_name = ctx["config"]["app_name"]
version = ctx["config"]["version"]
user_id = ctx["user_id"]
request_id = ctx["request_id"]

result = {
	"app_name": app_name,
	"version": version,
	"user_id": user_id,
	"request_id": request_id
}

_ = result
`

				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := starlarkEngine.NewEvaluator(
					handler,
					scriptLoader,
					compositeProvider,
				)
				require.NoError(t, err)

				// Add runtime data to context
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"user_id":    "user123",
					"request_id": "req456",
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify both static and dynamic data are accessible
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				assert.Equal(t, "TestApp", resultMap["app_name"])  // Static data
				assert.Equal(t, "1.0.0", resultMap["version"])     // Static data
				assert.Equal(t, "user123", resultMap["user_id"])   // Dynamic data
				assert.Equal(t, "req456", resultMap["request_id"]) // Dynamic data
			})

		t.Run("extism_composite_provider", func(t *testing.T) {
			// Static configuration values for Extism
			staticProvider := data.NewStaticProvider(map[string]any{
				"input": "CompositeProvider",
				"config": map[string]any{
					"app_name": "TestApp",
					"version":  "1.0.0",
				},
			})

			// Runtime data provider
			ctxProvider := data.NewContextProvider(constants.EvalData)

			// Combine them
			compositeProvider := data.NewCompositeProvider(staticProvider, ctxProvider)

			scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
			require.NoError(t, err)
			evaluator, err := extismEngine.NewEvaluator(
				handler,
				scriptLoader,
				compositeProvider,
				wasmdata.EntrypointGreet,
			)
			require.NoError(t, err)

			// Add runtime data to context (will override static "input")
			ctx := t.Context()
			enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
				"input": "DynamicComposite",
			})
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(enrichedCtx)
			require.NoError(t, err)

			// Verify dynamic data overrode static data
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			greeting, exists := resultMap["greeting"]
			require.True(t, exists)
			assert.Equal(t, "Hello, DynamicComposite!", greeting)
		})
	})
}

// TestHttpRequestDataAccess tests HTTP request processing patterns from platform/data/README.md
func TestHttpRequestDataAccess(t *testing.T) {
	t.Parallel()

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	// Helper function to create a test request for each test to avoid race conditions
	createTestRequest := func() *http.Request {
		req := httptest.NewRequest(
			"POST",
			"/api/test?param=value",
			strings.NewReader("request body"),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")
		return req
	}

	t.Run("http_request_processing", func(t *testing.T) {
		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"risor_http_request", func(t *testing.T) {
				script := `
// HTTP request data as documented in platform/data/README.md
request_method := ctx["request"]["Method"]
url_path := ctx["request"]["URL_Path"]
request_body := ctx["request"]["Body"]
content_type := ctx["request"]["Headers"]["Content-Type"][0]

{
	"method": request_method,
	"path": url_path,
	"body": request_body,
	"content_type": content_type
}
`

				// Create evaluator with context provider
				ctxProvider := data.NewContextProvider(constants.EvalData)
				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := risorEngine.NewEvaluator(handler, scriptLoader, ctxProvider)
				require.NoError(t, err)

				// Add HTTP request data with explicit key (as documented)
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"request": createTestRequest(),
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify HTTP request data is accessible
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				assert.Equal(t, "POST", resultMap["method"])
				assert.Equal(t, "/api/test", resultMap["path"])
				assert.Equal(t, "request body", resultMap["body"])
				assert.Equal(t, "application/json", resultMap["content_type"])
			})

		t.Run( //nolint:dupl // Each engine test demonstrates different syntax and behavior
			"starlark_http_request", func(t *testing.T) {
				script := `
# HTTP request data as documented in platform/data/README.md
request_method = ctx["request"]["Method"]
url_path = ctx["request"]["URL_Path"]
request_body = ctx["request"]["Body"]
content_type = ctx["request"]["Headers"]["Content-Type"][0]

result = {
	"method": request_method,
	"path": url_path,
	"body": request_body,
	"content_type": content_type
}

_ = result
`

				// Create evaluator with context provider
				ctxProvider := data.NewContextProvider(constants.EvalData)
				scriptLoader, err := loader.NewFromString(script)
				require.NoError(t, err)
				evaluator, err := starlarkEngine.NewEvaluator(handler, scriptLoader, ctxProvider)
				require.NoError(t, err)

				// Add HTTP request data with explicit key (as documented)
				ctx := t.Context()
				enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
					"request": createTestRequest(),
				})
				require.NoError(t, err)

				// Execute
				result, err := evaluator.Eval(enrichedCtx)
				require.NoError(t, err)

				// Verify HTTP request data is accessible
				resultMap, ok := result.Interface().(map[string]any)
				require.True(t, ok)

				assert.Equal(t, "POST", resultMap["method"])
				assert.Equal(t, "/api/test", resultMap["path"])
				assert.Equal(t, "request body", resultMap["body"])
				assert.Equal(t, "application/json", resultMap["content_type"])
			})

		t.Run("extism_http_request", func(t *testing.T) {
			// For Extism, we need to structure the HTTP request data appropriately
			// This demonstrates how to structure HTTP request data for WASM processing
			ctxProvider := data.NewContextProvider(constants.EvalData)
			scriptLoader, err := loader.NewFromBytes(wasmdata.TestModule)
			require.NoError(t, err)
			evaluator, err := extismEngine.NewEvaluator(
				handler,
				scriptLoader,
				ctxProvider,
				wasmdata.EntrypointGreet,
			)
			require.NoError(t, err)

			// Add HTTP request data - the greet function will use the request body as input
			ctx := t.Context()
			enrichedCtx, err := evaluator.AddDataToContext(ctx, map[string]any{
				"input":   "request body",      // Use the request body as input
				"request": createTestRequest(), // Include full request for potential processing
			})
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(enrichedCtx)
			require.NoError(t, err)

			// Verify the WASM module processed the request data
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			greeting, exists := resultMap["greeting"]
			require.True(t, exists)
			assert.Equal(t, "Hello, request body!", greeting)
		})
	})

	t.Run("explicit_key_wrapping", func(t *testing.T) {
		// This tests the recommended pattern from platform/data/README.md
		// "Always use explicit keys when adding data with AddDataToContext"

		t.Run("risor_explicit_keys", func(t *testing.T) {
			script := `
request_data := ctx["request"]
user_data := ctx["user"]
config_data := ctx["config"]

{
	"has_request": request_data != nil,
	"has_user": user_data != nil,
	"has_config": config_data != nil,
	"request_method": request_data["Method"],
	"user_name": user_data["name"]
}
`

			ctxProvider := data.NewContextProvider(constants.EvalData)
			scriptLoader, err := loader.NewFromString(script)
			require.NoError(t, err)
			evaluator, err := risorEngine.NewEvaluator(handler, scriptLoader, ctxProvider)
			require.NoError(t, err)

			// Use explicit keys as recommended in the documentation
			ctx := t.Context()
			enrichedCtx, err := evaluator.AddDataToContext(ctx,
				map[string]any{"request": createTestRequest()},
				map[string]any{"user": map[string]any{"name": "TestUser", "id": 123}},
				map[string]any{"config": map[string]any{"debug": true}},
			)
			require.NoError(t, err)

			// Execute
			result, err := evaluator.Eval(enrichedCtx)
			require.NoError(t, err)

			// Verify explicit key wrapping worked
			resultMap, ok := result.Interface().(map[string]any)
			require.True(t, ok)

			assert.Equal(t, true, resultMap["has_request"])
			assert.Equal(t, true, resultMap["has_user"])
			assert.Equal(t, true, resultMap["has_config"])
			assert.Equal(t, "POST", resultMap["request_method"])
			assert.Equal(t, "TestUser", resultMap["user_name"])
		})
	})
}
