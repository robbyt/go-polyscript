package wasmdata

import (
	"context"
	_ "embed"
	"encoding/json"
	"sync"
	"testing"
	"time"

	extismSDK "github.com/extism/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// testWasmBytes is now provided by TestModule in wasm.go
var testWasmBytes = TestModule

func TestExtismWasmIntegration(t *testing.T) {
	t.Parallel()
	// Create manifest from wasm bytes
	manifest := extismSDK.Manifest{
		Wasm: []extismSDK.Wasm{
			extismSDK.WasmData{
				Data: testWasmBytes,
			},
		},
	}

	// Create context and plugin config
	ctx := context.Background()
	cache := wazero.NewCompilationCache()
	defer func() {
		assert.NoError(t, cache.Close(ctx))
	}()

	config := extismSDK.PluginConfig{
		EnableWasi:    true, // Enable WASI since our plugin uses the PDK
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(cache),
	}

	// Create compiled plugin first
	compiledPlugin, err := extismSDK.NewCompiledPlugin(
		ctx,
		manifest,
		config,
		[]extismSDK.HostFunction{},
	)
	require.NoError(t, err, "Failed to create compiled plugin")
	defer func() {
		assert.NoError(t, compiledPlugin.Close(ctx))
	}()

	t.Run("concurrent greet calls", func(t *testing.T) {
		// Names to process concurrently
		names := []string{
			"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hank", "Ivy", "Jack",
			"Kathy", "Liam", "Mia", "Nora", "Oscar", "Pam", "Quinn", "Riley", "Sara", "Tom",
			"Uma", "Vince", "Wendy", "Xander", "Yara", "Zane", "World!",
		}

		// Create wait group for goroutines
		var wg sync.WaitGroup
		var mu sync.Mutex
		hasErrors := false

		// Process each name concurrently
		for i, name := range names {
			wg.Add(1)
			go func(instanceID int, input string) {
				defer wg.Done()

				// Create a new context for this instance with a timeout
				instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				// Create instance config
				instanceConfig := extismSDK.PluginInstanceConfig{
					ModuleConfig: wazero.NewModuleConfig(),
				}

				// Create instance for this goroutine
				instance, err := compiledPlugin.Instance(instanceCtx, instanceConfig)
				if err != nil {
					t.Logf("Failed to create instance %d: %v", instanceID, err)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}
				defer func() {
					assert.NoError(t, instance.Close(instanceCtx))
				}()

				// Create JSON input with the expected format
				inputJSON, err := json.Marshal(map[string]string{"input": input})
				if err != nil {
					t.Logf("Failed to marshal input %d: %v", instanceID, err)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}

				// Call the greet function with JSON input
				exit, output, err := instance.Call("greet", inputJSON)
				if err != nil {
					t.Logf("Instance call failed %d: %v", instanceID, err)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}

				if exit != 0 {
					t.Logf(
						"Non-zero exit code %d: %d, output: %s",
						instanceID,
						exit,
						string(output),
					)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}

				// Parse the JSON result
				var result map[string]string
				if err := json.Unmarshal(output, &result); err != nil {
					t.Logf(
						"Failed to parse result %d: %v, output: %s",
						instanceID,
						err,
						string(output),
					)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}

				// Verify expected greeting format
				expectedGreeting := "Hello, " + input + "!"
				if result["greeting"] != expectedGreeting {
					t.Logf(
						"Unexpected greeting %d: got %s, want %s",
						instanceID,
						result["greeting"],
						expectedGreeting,
					)
					mu.Lock()
					hasErrors = true
					mu.Unlock()
					return
				}
			}(i, name)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		assert.False(t, hasErrors, "Test completed with errors")
	})

	t.Run("complex JSON processing", func(t *testing.T) {
		// Create a new instance for the complex JSON test
		instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		instance, err := compiledPlugin.Instance(instanceCtx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err, "Failed to create instance for complex test")
		defer func() {
			assert.NoError(t, instance.Close(instanceCtx))
		}()

		// Create a test request
		complexRequest := map[string]any{
			"id":        "test-123",
			"timestamp": time.Now().Unix(),
			"data": map[string]any{
				"temperature": 72.5,
				"humidity":    45,
				"status":      "operational",
			},
			"tags": []string{"test", "demo", "example"},
			"metadata": map[string]string{
				"environment": "development",
				"version":     "1.0.0",
				"region":      "us-east-1",
			},
			"count":  42,
			"active": true,
		}

		// Marshal the request to JSON
		jsonBytes, err := json.Marshal(complexRequest)
		require.NoError(t, err, "Failed to marshal complex request")

		// Call the process_complex function
		exit, output, err := instance.Call("process_complex", jsonBytes)
		require.NoError(t, err, "Complex processing failed")
		require.Equal(t, uint32(0), exit, "Process_complex returned non-zero exit code")

		// Parse and verify the response
		var response map[string]any
		err = json.Unmarshal(output, &response)
		require.NoError(t, err, "Failed to parse complex response")

		assert.Equal(t, "test-123", response["request_id"])
		assert.NotEmpty(t, response["processed_at"])
		assert.Equal(t, float64(3), response["tag_count"]) // JSON numbers are float64
		assert.Equal(t, float64(3), response["meta_count"])
		assert.Contains(t, response["summary"], "test-123")
	})

	t.Run("count_vowels function", func(t *testing.T) {
		// Create a new instance
		instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		instance, err := compiledPlugin.Instance(instanceCtx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err, "Failed to create instance")
		defer func() {
			assert.NoError(t, instance.Close(instanceCtx))
		}()

		// Test input
		vowelInput := map[string]string{"input": "Hello World"}
		vowelInputJSON, err := json.Marshal(vowelInput)
		require.NoError(t, err, "Failed to marshal vowel input")

		// Call function
		exit, output, err := instance.Call("count_vowels", vowelInputJSON)
		require.NoError(t, err, "Count_vowels call failed")
		require.Equal(t, uint32(0), exit, "Count_vowels returned non-zero exit code")

		// Parse and verify result
		var vowelResult map[string]any
		err = json.Unmarshal(output, &vowelResult)
		require.NoError(t, err, "Failed to parse vowel result")

		assert.Equal(t, "Hello World", vowelResult["input"])
		assert.Equal(t, float64(3), vowelResult["count"]) // 3 vowels in "Hello World"
	})

	t.Run("reverse_string function", func(t *testing.T) {
		// Create a new instance
		instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		instance, err := compiledPlugin.Instance(instanceCtx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err, "Failed to create instance")
		defer func() {
			assert.NoError(t, instance.Close(instanceCtx))
		}()

		// Test input
		reverseInput := map[string]string{"input": "Hello World"}
		reverseInputJSON, err := json.Marshal(reverseInput)
		require.NoError(t, err, "Failed to marshal reverse input")

		// Call function
		exit, output, err := instance.Call("reverse_string", reverseInputJSON)
		require.NoError(t, err, "Reverse_string call failed")
		require.Equal(t, uint32(0), exit, "Reverse_string returned non-zero exit code")

		// Parse and verify result
		var reverseResult map[string]string
		err = json.Unmarshal(output, &reverseResult)
		require.NoError(t, err, "Failed to parse reverse result")

		assert.Equal(t, "dlroW olleH", reverseResult["reversed"])
	})

	t.Run("count_vowels_namespaced function", func(t *testing.T) {
		// Create a new instance
		instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		instance, err := compiledPlugin.Instance(instanceCtx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err, "Failed to create instance")
		defer func() {
			assert.NoError(t, instance.Close(instanceCtx))
		}()

		// Test input with namespaced structure
		namespacedInput := map[string]any{
			"data": map[string]string{
				"input": "Hello World",
			},
		}
		namespacedInputJSON, err := json.Marshal(namespacedInput)
		require.NoError(t, err, "Failed to marshal namespaced input")

		// Call function
		exit, output, err := instance.Call("count_vowels_namespaced", namespacedInputJSON)
		require.NoError(t, err, "Count_vowels_namespaced call failed")
		require.Equal(t, uint32(0), exit, "Count_vowels_namespaced returned non-zero exit code")

		// Parse and verify result
		var vowelResult map[string]any
		err = json.Unmarshal(output, &vowelResult)
		require.NoError(t, err, "Failed to parse vowel result")

		assert.Equal(t, "Hello World", vowelResult["input"])
		assert.Equal(t, float64(3), vowelResult["count"]) // 3 vowels in "Hello World"
	})

	t.Run("reverse_string_namespaced function", func(t *testing.T) {
		// Create a new instance
		instanceCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		instance, err := compiledPlugin.Instance(instanceCtx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err, "Failed to create instance")
		defer func() {
			assert.NoError(t, instance.Close(instanceCtx))
		}()

		// Test input with namespaced structure
		namespacedInput := map[string]any{
			"data": map[string]string{
				"input": "Hello World",
			},
		}
		namespacedInputJSON, err := json.Marshal(namespacedInput)
		require.NoError(t, err, "Failed to marshal namespaced input")

		// Call function
		exit, output, err := instance.Call("reverse_string_namespaced", namespacedInputJSON)
		require.NoError(t, err, "Reverse_string_namespaced call failed")
		require.Equal(t, uint32(0), exit, "Reverse_string_namespaced returned non-zero exit code")

		// Parse and verify result
		var reverseResult map[string]string
		err = json.Unmarshal(output, &reverseResult)
		require.NoError(t, err, "Failed to parse reverse result")

		assert.Equal(t, "dlroW olleH", reverseResult["reversed"])
	})
}
