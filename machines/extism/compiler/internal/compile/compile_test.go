package compile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/machines/extism/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

type TestRequest struct {
	ID        string            `json:"id"`
	Timestamp int64             `json:"timestamp"`
	Data      map[string]any    `json:"data"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	Count     int               `json:"count"`
	Active    bool              `json:"active"`
}

func readTestWasm(t *testing.T) []byte {
	t.Helper()
	wasmBytes, err := os.ReadFile("../../../testdata/examples/main.wasm")
	require.NoError(t, err)
	return wasmBytes
}

func TestCompileSuccess(t *testing.T) {
	t.Parallel()
	wasmBytes := readTestWasm(t)
	ctx := context.Background()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()
		plugin, err := CompileBytes(ctx, wasmBytes, nil)
		require.NoError(t, err)
		require.NotNil(t, plugin)
		defer func() { require.NoError(t, plugin.Close(ctx), "Failed to close plugin") }()

		// Create an instance to verify the plugin works
		pluginInstance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, pluginInstance.Close(ctx), "Failed to close plugin instance") }()

		testFunctions(t, pluginInstance)
	})

	t.Run("custom options", func(t *testing.T) {
		t.Parallel()
		opts := &Settings{
			EnableWASI: true,
			RuntimeConfig: wazero.NewRuntimeConfig().
				WithCompilationCache(wazero.NewCompilationCache()),
		}

		plugin, err := CompileBytes(ctx, wasmBytes, opts)
		require.NoError(t, err)
		require.NotNil(t, plugin)
		defer func() { require.NoError(t, plugin.Close(ctx), "Failed to close plugin") }()

		// Create an instance to verify the plugin works
		pluginInstance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, pluginInstance.Close(ctx), "Failed to close plugin instance") }()

		testFunctions(t, pluginInstance)
	})

	t.Run("base64 input default options", func(t *testing.T) {
		t.Parallel()
		wasmBase64 := base64.StdEncoding.EncodeToString(wasmBytes)
		plugin, err := CompileBase64(ctx, wasmBase64, nil)
		require.NoError(t, err)
		require.NotNil(t, plugin)
		defer func() { require.NoError(t, plugin.Close(ctx), "Failed to close plugin") }()

		// Create an instance to verify the plugin works
		pluginInstance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, pluginInstance.Close(ctx), "Failed to close plugin instance") }()

		testFunctions(t, pluginInstance)
	})

	t.Run("base64 input custom options", func(t *testing.T) {
		t.Parallel()
		opts := &Settings{
			EnableWASI:    true,
			RuntimeConfig: wazero.NewRuntimeConfig(),
		}

		wasmBase64 := base64.StdEncoding.EncodeToString(wasmBytes)
		plugin, err := CompileBase64(ctx, wasmBase64, opts)
		require.NoError(t, err)
		require.NotNil(t, plugin)
		defer func() { require.NoError(t, plugin.Close(ctx), "Failed to close plugin") }()

		// Create an instance to verify the plugin works
		pluginInstance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
			ModuleConfig: wazero.NewModuleConfig(),
		})
		require.NoError(t, err)
		defer func() { require.NoError(t, pluginInstance.Close(ctx), "Failed to close plugin instance") }()

		testFunctions(t, pluginInstance)
	})
}

func testFunctions(t *testing.T, instance adapters.PluginInstance) {
	t.Helper()

	// Test different Wasm functions
	tests := []struct {
		name       string
		funcName   string
		input      any
		assertFunc func(t *testing.T, output []byte)
	}{
		{
			name:     "greet function",
			funcName: "greet",
			input:    map[string]string{"input": "World"},
			assertFunc: func(t *testing.T, output []byte) {
				t.Helper()
				var result struct {
					Greeting string `json:"greeting"`
				}
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, "Hello, World!", result.Greeting)
			},
		},
		{
			name:     "reverse_string function",
			funcName: "reverse_string",
			input:    map[string]string{"input": "Hello"},
			assertFunc: func(t *testing.T, output []byte) {
				t.Helper()
				var result struct {
					Reversed string `json:"reversed"`
				}
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, "olleH", result.Reversed)
			},
		},
		{
			name:     "count_vowels function",
			funcName: "count_vowels",
			input:    map[string]string{"input": "Hello World"},
			assertFunc: func(t *testing.T, output []byte) {
				t.Helper()
				var result struct {
					Count  int    `json:"count"`
					Vowels string `json:"vowels"`
					Input  string `json:"input"`
				}
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, 3, result.Count) // "e", "o", "o" in "Hello World"
				assert.Equal(t, "Hello World", result.Input)
			},
		},
		{
			name:     "process_complex function",
			funcName: "process_complex",
			input: TestRequest{
				ID:        "test-123",
				Timestamp: time.Now().Unix(),
				Data: map[string]any{
					"key1": "value1",
					"key2": 42,
				},
				Tags: []string{"test", "example"},
				Metadata: map[string]string{
					"source":  "unit-test",
					"version": "1.0",
				},
				Count:  42,
				Active: true,
			},
			assertFunc: func(t *testing.T, output []byte) {
				t.Helper()
				var result struct {
					RequestID   string         `json:"request_id"`
					ProcessedAt string         `json:"processed_at"`
					Results     map[string]any `json:"results"`
					TagCount    int            `json:"tag_count"`
					MetaCount   int            `json:"meta_count"`
					IsActive    bool           `json:"is_active"`
					Summary     string         `json:"summary"`
				}
				require.NoError(t, json.Unmarshal(output, &result))
				assert.Equal(t, "test-123", result.RequestID)
				assert.Equal(t, 2, result.TagCount)
				assert.Equal(t, 2, result.MetaCount)
				assert.True(t, result.IsActive)
				assert.Contains(t, result.Summary, "test-123")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err)

			exit, output, err := instance.Call(tt.funcName, inputJSON)
			require.NoError(t, err)
			assert.Equal(t, uint32(0), exit, "Function should execute successfully")

			tt.assertFunc(t, output)
		})
	}
}

func TestCompileErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		input     []byte
		base64    string
		opts      *Settings
		useBase64 bool
		wantErr   error
	}{
		{
			name:    "nil bytes",
			input:   nil,
			wantErr: ErrContentNil,
		},
		{
			name:      "invalid base64",
			base64:    "not-base64-encoded",
			useBase64: true,
			wantErr:   ErrInvalidBinary,
		},
		{
			name:      "valid base64 but invalid wasm",
			base64:    base64.StdEncoding.EncodeToString([]byte("not-wasm-binary")),
			useBase64: true,
			wantErr:   ErrCompileFailed,
		},
		{
			name: "corrupted wasm",
			input: append(
				[]byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00},
				[]byte("corrupted")...),
			wantErr: ErrCompileFailed,
		},
		{
			name:    "empty bytes",
			input:   []byte{},
			wantErr: ErrContentNil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.useBase64 {
				_, err = CompileBase64(ctx, tt.base64, tt.opts)
			} else {
				_, err = CompileBytes(ctx, tt.input, tt.opts)
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCompileOptionsDefaults(t *testing.T) {
	opts := WithDefaultCompileSettings()
	require.NotNil(t, opts)
	assert.True(t, opts.EnableWASI)
	assert.NotNil(t, opts.RuntimeConfig)
	assert.Empty(t, opts.HostFunctions)
}

func TestCompileWithHostFunctions(t *testing.T) {
	ctx := context.Background()
	wasmBytes := readTestWasm(t)

	// Create test host function
	testHostFn := extismSDK.NewHostFunctionWithStack(
		"test_function",
		func(ctx context.Context, p *extismSDK.CurrentPlugin, stack []uint64) {
			// No-op function for testing
		},
		[]extismSDK.ValueType{extismSDK.ValueTypeI64},
		[]extismSDK.ValueType{extismSDK.ValueTypeI64},
	)

	opts := &Settings{
		EnableWASI:    true,
		RuntimeConfig: wazero.NewRuntimeConfig(),
		HostFunctions: []extismSDK.HostFunction{testHostFn},
	}

	plugin, err := CompileBytes(ctx, wasmBytes, opts)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	defer func() { require.NoError(t, plugin.Close(ctx), "Failed to close plugin") }()

	// Create an instance
	instance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig(),
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, instance.Close(ctx), "Failed to close instance") }()

	assert.NotNil(t, instance)
}
