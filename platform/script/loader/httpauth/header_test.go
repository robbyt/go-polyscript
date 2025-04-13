package httpauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHeaderAuth(t *testing.T) {
	t.Parallel()

	t.Run("Multiple custom headers", func(t *testing.T) {
		auth := NewHeaderAuth(map[string]string{
			"Authorization": "Bearer token123",
			"X-API-Key":     "secret-key",
			"X-Custom":      "value",
		})
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		err = auth.Authenticate(req)
		require.NoError(t, err)

		require.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
		require.Equal(t, "secret-key", req.Header.Get("X-API-Key"))
		require.Equal(t, "value", req.Header.Get("X-Custom"))
	})

	t.Run("Empty headers map", func(t *testing.T) {
		auth := NewHeaderAuth(map[string]string{})
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		err = auth.Authenticate(req)
		require.NoError(t, err)

		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("Nil headers map", func(t *testing.T) {
		auth := NewHeaderAuth(nil)
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		err = auth.Authenticate(req)
		require.NoError(t, err)

		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("Bearer token helper", func(t *testing.T) {
		auth := NewBearerAuth("my-test-token")
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		err = auth.Authenticate(req)
		require.NoError(t, err)

		require.Equal(t, "Bearer my-test-token", req.Header.Get("Authorization"))
	})

	t.Run("With context", func(t *testing.T) {
		auth := NewHeaderAuth(map[string]string{
			"Authorization": "Bearer token123",
		})
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)
		ctx := context.Background()

		err = auth.AuthenticateWithContext(ctx, req)
		require.NoError(t, err)

		require.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
	})

	t.Run("With cancelled context", func(t *testing.T) {
		auth := NewHeaderAuth(map[string]string{
			"Authorization": "Bearer token123",
		})
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		defer cancel()

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})

	t.Run("With timeout context", func(t *testing.T) {
		auth := NewHeaderAuth(map[string]string{
			"Authorization": "Bearer token123",
		})
		require.Equal(t, "Header", auth.Name())

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		time.Sleep(5 * time.Millisecond) // Ensure the timeout occurs
		defer cancel()

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})
}

// Test that maps.Clone is properly cloning the headers map
func TestHeaderAuthCloning(t *testing.T) {
	t.Parallel()

	// Create original map
	originalHeaders := map[string]string{
		"X-Test": "value",
	}

	// Create authenticator
	auth := NewHeaderAuth(originalHeaders)

	// Modify original map after creation
	originalHeaders["X-Test"] = "modified"
	originalHeaders["X-New"] = "added"

	// Create a request
	req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
	require.NoError(t, err)

	// Apply auth
	err = auth.Authenticate(req)
	require.NoError(t, err)

	// The authenticator should have used its own internal copy, not the modified one
	require.Equal(t, "value", req.Header.Get("X-Test"))
	require.Empty(t, req.Header.Get("X-New"))
}
