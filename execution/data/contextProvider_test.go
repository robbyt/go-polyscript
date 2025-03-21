package data

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/robbyt/go-polyscript/execution/constants"
)

func TestStaticProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	provider := NewStaticProvider(map[string]any{"test": "value"})
	ctx := context.Background()

	// Static provider should reject all attempts to add data
	_, err := provider.AddDataToContext(ctx, "some data")
	assert.Error(t, err, "StaticProvider should reject all attempts to add data")
	assert.Contains(t, err.Error(), "doesn't support adding data", "Error message should explain limitation")
}

func TestContextProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contextKey  constants.ContextKey
		data        []any
		shouldError bool
	}{
		{
			name:        "empty context key",
			contextKey:  "",
			data:        []any{map[string]any{"key": "value"}},
			shouldError: true,
		},
		{
			name:        "nil data",
			contextKey:  "test_key",
			data:        []any{nil},
			shouldError: false,
		},
		{
			name:        "map data",
			contextKey:  "test_key",
			data:        []any{map[string]any{"key1": "value1", "key2": 123}},
			shouldError: false,
		},
		{
			name:        "unsupported type",
			contextKey:  "test_key",
			data:        []any{42}, // Integer type is not supported
			shouldError: true,
		},
		{
			name:       "multiple supported types",
			contextKey: "test_key",
			data: []any{
				map[string]any{"key1": "value1"},
				&http.Request{
					Method: "GET",
					URL:    &url.URL{Path: "/test"},
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewContextProvider(tt.contextKey)
			require.NotNil(t, provider)

			ctx := context.Background()
			newCtx, err := provider.AddDataToContext(ctx, tt.data...)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.contextKey != "" { // Empty key will error out above
					data, getErr := provider.GetData(newCtx)
					assert.NoError(t, getErr)
					assert.NotNil(t, data)
				}
			}
		})
	}
}

func TestCompositeProvider_AddDataToContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		providers      []Provider
		data           []any
		shouldError    bool
		allShouldError bool
	}{
		{
			name:           "empty providers",
			providers:      []Provider{},
			data:           []any{map[string]any{"key": "value"}},
			shouldError:    false,
			allShouldError: false,
		},
		{
			name: "single provider success",
			providers: []Provider{
				NewContextProvider("test_key"),
			},
			data:           []any{map[string]any{"key": "value"}},
			shouldError:    false,
			allShouldError: false,
		},
		{
			name: "single provider error",
			providers: []Provider{
				NewStaticProvider(nil), // Static provider always errors on AddDataToContext
			},
			data:           []any{map[string]any{"key": "value"}},
			shouldError:    true,
			allShouldError: true,
		},
		{
			name: "mixed providers",
			providers: []Provider{
				NewContextProvider("test_key"),
				NewStaticProvider(nil), // This one will error
			},
			data:           []any{map[string]any{"key": "value"}},
			shouldError:    false, // Should not error overall, as at least one provider succeeded
			allShouldError: false,
		},
		{
			name: "all error providers",
			providers: []Provider{
				NewStaticProvider(nil), // This one will error
				NewStaticProvider(nil), // This one will error
				&mockErrorProvider{},   // This one will error
			},
			data:           []any{map[string]any{"key": "value"}},
			shouldError:    true, // Should error when all providers fail
			allShouldError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewCompositeProvider(tt.providers...)
			require.NotNil(t, provider)

			ctx := context.Background()
			newCtx, err := provider.AddDataToContext(ctx, tt.data...)

			if tt.shouldError {
				assert.Error(t, err)
				if tt.allShouldError {
					assert.Equal(t, ctx, newCtx, "Context should remain unchanged when all providers error")
				}
			} else {
				if len(tt.providers) > 0 {
					assert.NotEqual(t, ctx, newCtx, "Context should be modified when at least one provider succeeds")
				} else {
					assert.Equal(t, ctx, newCtx, "Context should remain unchanged when no providers exist")
				}
			}
		})
	}
}
