package extism

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	extismSDK "github.com/extism/go-sdk"
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
	wasmBytes, err := os.ReadFile("testdata/examples/main.wasm")
	require.NoError(t, err)
	return wasmBytes
}

func TestCompileSuccess(t *testing.T) {
	wasmBytes := readTestWasm(t)

	tests := []struct {
		name      string
		opts      *compileOptions
		useBase64 bool
	}{
		{
			name:      "default options",
			opts:      nil,
			useBase64: false,
		},
		{
			name: "custom options",
			opts: &compileOptions{
				EnableWASI:    true,
				RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(wazero.NewCompilationCache()),
			},
			useBase64: false,
		},
		{
			name:      "base64 input default options",
			opts:      nil,
			useBase64: true,
		},
		{
			name: "base64 input custom options",
			opts: &compileOptions{
				EnableWASI:    true,
				RuntimeConfig: wazero.NewRuntimeConfig(),
			},
			useBase64: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var plugin compiledPlugin
			var err error

			if tt.useBase64 {
				wasmBase64 := base64.StdEncoding.EncodeToString(wasmBytes)
				plugin, err = CompileBase64(ctx, wasmBase64, tt.opts)
			} else {
				plugin, err = CompileBytes(ctx, wasmBytes, tt.opts)
			}

			require.NoError(t, err)
			require.NotNil(t, plugin)
			defer plugin.Close(ctx)

			// Create an instance to verify the plugin works
			pluginInstance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
				ModuleConfig: wazero.NewModuleConfig(),
			})
			require.NoError(t, err)
			defer pluginInstance.Close(ctx)

			testCases := []struct {
				name     string
				input    []byte
				wantErr  bool
				validate func(t *testing.T, output []byte)
			}{
				{
					name:  "greet",
					input: []byte(`{"input":"World"}`),
					validate: func(t *testing.T, output []byte) {
						var result struct {
							Greeting string `json:"greeting"`
						}
						require.NoError(t, json.Unmarshal(output, &result))
						assert.Equal(t, "Hello, World!", result.Greeting)
					},
				},
				{
					name:  "reverse_string",
					input: []byte(`{"input":"Hello"}`),
					validate: func(t *testing.T, output []byte) {
						var result struct {
							Reversed string `json:"reversed"`
						}
						require.NoError(t, json.Unmarshal(output, &result))
						assert.Equal(t, "olleH", result.Reversed)
					},
				},
				{
					name:  "count_vowels",
					input: []byte(`{"input":"Hello World"}`),
					validate: func(t *testing.T, output []byte) {
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
					name: "process_complex",
					input: func() []byte {
						req := TestRequest{
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
						}
						b, err := json.Marshal(req)
						require.NoError(t, err)
						return b
					}(),
					validate: func(t *testing.T, output []byte) {
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

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					exit, output, err := pluginInstance.Call(tc.name, tc.input)
					if tc.wantErr {
						assert.Error(t, err)
						return
					}
					require.NoError(t, err)
					assert.Equal(t, uint32(0), exit, "Function %s should execute successfully", tc.name)

					if tc.validate != nil {
						tc.validate(t, output)
					}
				})
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     []byte
		base64    string
		opts      *compileOptions
		useBase64 bool
		wantErr   string
	}{
		{
			name:    "nil bytes",
			input:   nil,
			wantErr: "wasm content is nil",
		},
		{
			name:      "invalid base64",
			base64:    "not-base64-encoded",
			useBase64: true,
			wantErr:   "invalid WASM binary (must be base64 encoded)",
		},
		{
			name:      "valid base64 but invalid wasm",
			base64:    base64.StdEncoding.EncodeToString([]byte("not-wasm-binary")),
			useBase64: true,
			wantErr:   "failed to compile plugin",
		},
		{
			name:    "corrupted wasm",
			input:   append([]byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}, []byte("corrupted")...),
			wantErr: "failed to compile plugin",
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
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCompileOptionsDefaults(t *testing.T) {
	opts := withDefaultCompileOptions()
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

	opts := &compileOptions{
		EnableWASI:    true,
		RuntimeConfig: wazero.NewRuntimeConfig(),
		HostFunctions: []extismSDK.HostFunction{testHostFn},
	}

	plugin, err := CompileBytes(ctx, wasmBytes, opts)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	defer plugin.Close(ctx)

	// Create an instance
	instance, err := plugin.Instance(ctx, extismSDK.PluginInstanceConfig{
		ModuleConfig: wazero.NewModuleConfig(),
	})
	require.NoError(t, err)
	defer instance.Close(ctx)

	assert.NotNil(t, instance)
}
