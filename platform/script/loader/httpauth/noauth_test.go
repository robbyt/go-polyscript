package httpauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNoAuth(t *testing.T) {
	t.Parallel()

	// Create the authenticator
	auth := NewNoAuth()

	// Check the name
	require.Equal(t, "None", auth.Name())

	t.Run("Basic authentication", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		err = auth.Authenticate(req)
		require.NoError(t, err)

		// No-auth shouldn't add any headers
		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("With context authentication", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)
		ctx := t.Context()

		err = auth.AuthenticateWithContext(ctx, req)
		require.NoError(t, err)

		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("With cancelled context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately
		defer cancel()

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})

	t.Run("With timeout context", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Nanosecond)
		time.Sleep(5 * time.Millisecond) // Ensure the timeout occurs
		defer cancel()

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})
}
