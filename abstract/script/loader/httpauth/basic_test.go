package httpauth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicAuth(t *testing.T) {
	t.Parallel()

	t.Run("Valid credentials", func(t *testing.T) {
		username := "testuser"
		password := "testpass"

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		auth := NewBasicAuth(username, password)
		require.Equal(t, "Basic", auth.Name())

		err = auth.Authenticate(req)
		require.NoError(t, err)

		actualUsername, actualPassword, ok := req.BasicAuth()
		require.True(t, ok, "Basic auth header should be present")
		require.Equal(t, username, actualUsername)
		require.Equal(t, password, actualPassword)
	})

	t.Run("Empty username (no auth applied)", func(t *testing.T) {
		username := ""
		password := "testpass"

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		auth := NewBasicAuth(username, password)
		require.Equal(t, "Basic", auth.Name())

		err = auth.Authenticate(req)
		require.NoError(t, err)

		_, _, ok := req.BasicAuth()
		require.False(t, ok, "Basic auth header should not be present")
		require.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("With context", func(t *testing.T) {
		username := "testuser"
		password := "testpass"

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)
		ctx := context.Background()

		auth := NewBasicAuth(username, password)
		require.Equal(t, "Basic", auth.Name())

		err = auth.AuthenticateWithContext(ctx, req)
		require.NoError(t, err)

		actualUsername, actualPassword, ok := req.BasicAuth()
		require.True(t, ok, "Basic auth header should be present")
		require.Equal(t, username, actualUsername)
		require.Equal(t, password, actualPassword)
	})

	t.Run("With cancelled context", func(t *testing.T) {
		username := "testuser"
		password := "testpass"

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		defer cancel()

		auth := NewBasicAuth(username, password)
		require.Equal(t, "Basic", auth.Name())

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})

	t.Run("With timeout context", func(t *testing.T) {
		username := "testuser"
		password := "testpass"

		req, err := http.NewRequest(http.MethodGet, "http://localhost/test", nil)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		time.Sleep(5 * time.Millisecond) // Ensure the timeout occurs
		defer cancel()

		auth := NewBasicAuth(username, password)
		require.Equal(t, "Basic", auth.Name())

		err = auth.AuthenticateWithContext(ctx, req)
		require.Error(t, err)
	})
}
