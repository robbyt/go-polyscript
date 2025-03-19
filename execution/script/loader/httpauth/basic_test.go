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

	tests := []struct {
		name        string
		username    string
		password    string
		setup       func() (*http.Request, context.Context, context.CancelFunc)
		expectError bool
		verifyReq   func(t *testing.T, req *http.Request)
	}{
		{
			name:     "Valid credentials",
			username: "testuser",
			password: "testpass",
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				username, password, ok := req.BasicAuth()
				require.True(t, ok, "Basic auth header should be present")
				require.Equal(t, "testuser", username)
				require.Equal(t, "testpass", password)
			},
		},
		{
			name:     "Empty username (no auth applied)",
			username: "",
			password: "testpass",
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				return req, nil, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				_, _, ok := req.BasicAuth()
				require.False(t, ok, "Basic auth header should not be present")
				require.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name:     "With context",
			username: "testuser",
			password: "testpass",
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx := context.Background()
				return req, ctx, nil
			},
			verifyReq: func(t *testing.T, req *http.Request) {
				username, password, ok := req.BasicAuth()
				require.True(t, ok, "Basic auth header should be present")
				require.Equal(t, "testuser", username)
				require.Equal(t, "testpass", password)
			},
		},
		{
			name:     "With cancelled context",
			username: "testuser",
			password: "testpass",
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return req, ctx, nil
			},
			expectError: true,
		},
		{
			name:     "With timeout context",
			username: "testuser",
			password: "testpass",
			setup: func() (*http.Request, context.Context, context.CancelFunc) {
				req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				time.Sleep(5 * time.Millisecond) // Ensure the timeout occurs
				return req, ctx, cancel
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			auth := NewBasicAuth(tt.username, tt.password)
			require.Equal(t, "Basic", auth.Name())

			req, ctx, cancel := tt.setup()
			if cancel != nil {
				defer cancel()
			}

			var err error
			if ctx != nil {
				err = auth.AuthenticateWithContext(ctx, req)
			} else {
				err = auth.Authenticate(req)
			}

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.verifyReq != nil {
				tt.verifyReq(t, req)
			}
		})
	}
}
