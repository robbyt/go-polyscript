package data

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoadInputData(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("nil provider returns empty map", func(t *testing.T) {
		result, err := LoadInputData(t.Context(), logger, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NotNil(t, result)
	})

	t.Run("provider returns data", func(t *testing.T) {
		provider := new(MockProvider)
		expected := map[string]any{"key": "value", "count": 42}
		provider.On("GetData", mock.Anything).Return(expected, nil)

		result, err := LoadInputData(t.Context(), logger, provider)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		provider.AssertExpectations(t)
	})

	t.Run("provider returns error", func(t *testing.T) {
		provider := new(MockProvider)
		provider.On("GetData", mock.Anything).Return(nil, assert.AnError)

		result, err := LoadInputData(t.Context(), logger, provider)
		require.Error(t, err)
		assert.Nil(t, result)
		provider.AssertExpectations(t)
	})

	t.Run("provider returns empty map", func(t *testing.T) {
		provider := new(MockProvider)
		provider.On("GetData", mock.Anything).Return(map[string]any{}, nil)

		result, err := LoadInputData(t.Context(), logger, provider)
		require.NoError(t, err)
		assert.Empty(t, result)
		provider.AssertExpectations(t)
	})

	t.Run("nil logger does not panic", func(t *testing.T) {
		provider := new(MockProvider)
		expected := map[string]any{"key": "value"}
		provider.On("GetData", mock.Anything).Return(expected, nil)

		result, err := LoadInputData(t.Context(), nil, provider)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		provider.AssertExpectations(t)
	})

	t.Run("nil logger and nil provider does not panic", func(t *testing.T) {
		result, err := LoadInputData(t.Context(), nil, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NotNil(t, result)
	})
}
