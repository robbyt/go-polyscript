package loader

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/platform/script/loader/httpauth"
	"github.com/stretchr/testify/require"
)

func TestNewFromHTTP(t *testing.T) {
	t.Parallel()

	t.Run("Valid HTTPS URL", func(t *testing.T) {
		// Set up TLS server
		tlsServer := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(FunctionContent))
				require.NoError(t, err)
			}),
		)
		defer tlsServer.Close()

		testURL := tlsServer.URL + "/script.js"

		// Set InsecureSkipVerify to make test work with self-signed cert
		options := DefaultHTTPOptions()
		options.InsecureSkipVerify = true
		loader, err := NewFromHTTPWithOptions(testURL, options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// Verify loader properties
		require.Equal(t, testURL, loader.url)
		require.NotNil(t, loader.sourceURL)
		require.Equal(t, testURL, loader.sourceURL.String())
		require.NotNil(t, loader.client)
		require.NotNil(t, loader.options)

		// Use TLS server's client to accept its certificate
		loader.client = tlsServer.Client()

		// Additional verification
		verifyLoader(t, loader, testURL)
	})

	t.Run("Valid HTTP URL", func(t *testing.T) {
		// Set up HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(FunctionContent))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// Verify loader properties
		require.Equal(t, testURL, loader.url)
		require.NotNil(t, loader.sourceURL)
		require.Equal(t, testURL, loader.sourceURL.String())
		require.NotNil(t, loader.client)
		require.NotNil(t, loader.options)

		// Additional verification
		verifyLoader(t, loader, testURL)
	})

	t.Run("Invalid URL scheme", func(t *testing.T) {
		testURL := "file:///path/to/script.js"

		loader, err := NewFromHTTP(testURL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported scheme")
		require.Nil(t, loader)
	})

	t.Run("Invalid URL format", func(t *testing.T) {
		testURL := "://invalid-url"

		loader, err := NewFromHTTP(testURL)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unable to parse URL")
		require.Nil(t, loader)
	})
}

func TestNewFromHTTPWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		optionsModifier func(options *HTTPOptions) *HTTPOptions
		validateOption  func(t *testing.T, loader *FromHTTP)
		expectError     bool
		errorContains   string
	}{
		{
			name: "Custom timeout",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				return options.WithTimeout(60 * time.Second)
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				t.Helper()
				require.Equal(t, 60*time.Second, loader.options.Timeout)
			},
		},
		{
			name: "Basic auth",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				return options.WithBasicAuth("user", "pass")
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				t.Helper()
				auth, ok := loader.options.Authenticator.(*httpauth.BasicAuth)
				require.True(t, ok, "Expected BasicAuth authenticator")
				require.Equal(t, "user", auth.Username)
				require.Equal(t, "pass", auth.Password)
			},
		},
		{
			name: "Bearer auth",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				return options.WithBearerAuth("token123")
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				t.Helper()
				auth, ok := loader.options.Authenticator.(*httpauth.HeaderAuth)
				require.True(t, ok, "Expected HeaderAuth authenticator")
				require.Equal(t, "Bearer token123", auth.Headers["Authorization"])
			},
		},
		{
			name: "Custom headers",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				options.Headers["X-Custom"] = "TestValue"
				options.Headers["User-Agent"] = "Test-Agent"
				return options
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				t.Helper()
				require.Equal(t, "TestValue", loader.options.Headers["X-Custom"])
				require.Equal(t, "Test-Agent", loader.options.Headers["User-Agent"])
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Create test server for this test case
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(FunctionContent))
					require.NoError(t, err)
				}),
			)
			defer server.Close()

			testURL := server.URL + "/script.js"

			// Start with default options and apply modifier if provided
			options := DefaultHTTPOptions()
			if tc.optionsModifier != nil {
				options = tc.optionsModifier(options)
			}

			loader, err := NewFromHTTPWithOptions(testURL, options)
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)
			require.Equal(t, testURL, loader.url)
			require.NotNil(t, loader.sourceURL)
			require.Equal(t, testURL, loader.sourceURL.String())
			require.NotNil(t, loader.client)
			require.NotNil(t, loader.options)

			if tc.validateOption != nil {
				tc.validateOption(t, loader)
			}

			// Use helper for further validation
			verifyLoader(t, loader, testURL)
		})
	}
}

func TestFromHTTP_TLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("with insecure skip verify", func(t *testing.T) {
		// Create test server for this test
		server := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(FunctionContent))
				require.NoError(t, err)
			}),
		)
		defer server.Close()

		testURL := server.URL + "/script.js"

		options := DefaultHTTPOptions()
		options.InsecureSkipVerify = true

		loader, err := NewFromHTTPWithOptions(testURL, options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// Extract the transport to verify TLS settings
		transport, ok := loader.client.(*http.Client).Transport.(*http.Transport)
		require.True(t, ok, "Expected *http.Transport")
		require.NotNil(t, transport.TLSClientConfig)
		require.True(t, transport.TLSClientConfig.InsecureSkipVerify,
			"InsecureSkipVerify should be true")
	})

	t.Run("with custom TLS config", func(t *testing.T) {
		// Create test server for this test
		server := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(FunctionContent))
				require.NoError(t, err)
			}),
		)
		defer server.Close()

		testURL := server.URL + "/script.js"

		options := DefaultHTTPOptions()
		customTLS := &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Add custom ciphers
			CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
			// Use InsecureSkipVerify for the test server
			InsecureSkipVerify: true,
		}
		options.TLSConfig = customTLS

		loader, err := NewFromHTTPWithOptions(testURL, options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// Extract the transport to verify TLS settings
		transport, ok := loader.client.(*http.Client).Transport.(*http.Transport)
		require.True(t, ok, "Expected *http.Transport")
		require.NotNil(t, transport.TLSClientConfig)
		require.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
		require.Contains(t, transport.TLSClientConfig.CipherSuites,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384)
	})

	t.Run("TLSConfig takes precedence over InsecureSkipVerify", func(t *testing.T) {
		// Create test server for this test
		server := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(FunctionContent))
				require.NoError(t, err)
			}),
		)
		defer server.Close()

		testURL := server.URL + "/script.js"

		options := DefaultHTTPOptions()
		options.InsecureSkipVerify = true
		customTLS := &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS13,
		}
		options.TLSConfig = customTLS

		loader, err := NewFromHTTPWithOptions(testURL, options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// Extract the transport to verify TLS settings
		transport, ok := loader.client.(*http.Client).Transport.(*http.Transport)
		require.True(t, ok, "Expected *http.Transport")
		require.NotNil(t, transport.TLSClientConfig)

		// Should use the TLSConfig value (false) rather than the InsecureSkipVerify field (true)
		require.False(t, transport.TLSClientConfig.InsecureSkipVerify,
			"TLSConfig should override InsecureSkipVerify")
		require.Equal(t, uint16(tls.VersionTLS13), transport.TLSClientConfig.MinVersion)

		// For testing purposes, replace the client with one that accepts the test server's certificate
		loader.client = server.Client()
	})

	t.Run("no TLS modifications when neither option is set", func(t *testing.T) {
		// Create test server for this test
		server := httptest.NewTLSServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(FunctionContent))
				require.NoError(t, err)
			}),
		)
		defer server.Close()

		testURL := server.URL + "/script.js"

		options := DefaultHTTPOptions()

		loader, err := NewFromHTTPWithOptions(testURL, options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// When no TLS options are set, the client uses the default transport without modifications
		client, ok := loader.client.(*http.Client)
		require.True(t, ok, "Expected http.Client")

		// The http.Client only initializes Transport when needed, so it might be nil at this point
		// We should check that it's nil when neither TLS option is set
		require.Nil(t, client.Transport, "Expected Transport to be nil when no TLS options are set")

		// For testing purposes, replace the client with one that accepts the test server's certificate
		loader.client = server.Client()
	})
}

func TestFromHTTP_GetReader(t *testing.T) {
	t.Parallel()

	const testScript = FunctionContent

	// Test with simple basic mocks instead of complex HTTP validation
	t.Run("successful read", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testScript))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Use real server with helper
		reader, err := loader.GetReader()
		require.NoError(t, err)
		verifyReaderContent(t, reader, testScript)
	})

	t.Run("unauthorized error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte("Unauthorized"))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/auth"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.Error(t, err)
		require.Contains(t, err.Error(), "HTTP 401")
		require.Nil(t, reader)
	})

	t.Run("not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Not Found"))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/not-found"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		reader, err := loader.GetReader()
		require.Error(t, err)
		require.Contains(t, err.Error(), "HTTP 404")
		require.Nil(t, reader)
	})

	t.Run("network error", func(t *testing.T) {
		// Use any URL since we'll replace the client with a mock
		testURL := "https://localhost:8080/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Replace with client that returns an error
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}
		loader.client = mockClient

		reader, err := loader.GetReader()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to execute HTTP request")
		require.Nil(t, reader)
	})
}

func TestFromHTTP_GetReaderWithContext(t *testing.T) {
	t.Parallel()
	const testScript = FunctionContent

	t.Run("Success - Background Context", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testScript))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		ctx := context.Background()
		reader, err := loader.GetReaderWithContext(ctx)
		require.NoError(t, err)
		require.NotNil(t, reader)
		verifyReaderContent(t, reader, testScript)
	})

	t.Run("Failure - Cancelled Context", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testScript))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Use mock client to ensure we're testing context cancellation
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Should fail with context error
				return nil, req.Context().Err()
			},
		}
		loader.client = mockClient

		reader, err := loader.GetReaderWithContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context canceled")
		require.Nil(t, reader)
	})

	t.Run("Failure - Timeout Context", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delay to ensure timeout happens
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(testScript))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Use mock client to ensure we're testing context timeout
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Small sleep to ensure context times out
				time.Sleep(1 * time.Millisecond)
				return nil, req.Context().Err()
			},
		}
		loader.client = mockClient

		reader, err := loader.GetReaderWithContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context deadline exceeded")
		require.Nil(t, reader)
	})
}

func TestFromHTTP_String(t *testing.T) {
	t.Parallel()

	t.Run("successful string representation", func(t *testing.T) {
		// Create test server that returns content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("test script content"))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		str := loader.String()
		require.Contains(t, str, "loader.FromHTTP{URL:")
		require.Contains(t, str, testURL)
		require.Contains(t, str, "SHA256:")
	})

	t.Run("string representation with network error", func(t *testing.T) {
		// Create server that deliberately fails connections (invalid port)
		testURL := "http://localhost:1" // This port is unlikely to be listening

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		str := loader.String()
		require.Contains(t, str, "loader.FromHTTP{URL:")
		require.Contains(t, str, testURL)
		require.NotContains(t, str, "SHA256")
	})

	t.Run("string representation with HTTP error", func(t *testing.T) {
		// Create test server that returns an error status
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte("Not Found"))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		str := loader.String()
		require.Contains(t, str, "loader.FromHTTP{URL:")
		require.Contains(t, str, testURL)
		require.NotContains(t, str, "SHA256")
	})
}

func TestDefaultHTTPOptions(t *testing.T) {
	t.Parallel()

	options := DefaultHTTPOptions()
	require.NotNil(t, options)
	require.Equal(t, 30*time.Second, options.Timeout)
	require.False(t, options.InsecureSkipVerify)
	require.NotNil(t, options.Authenticator)
	require.Equal(t, "None", options.Authenticator.Name())
	require.NotNil(t, options.Headers)
	require.Empty(t, options.Headers)
}

func TestHTTPOptionsWithMethods(t *testing.T) {
	t.Parallel()

	t.Run("option method chaining", func(t *testing.T) {
		// Test chaining of option methods
		options := DefaultHTTPOptions().
			WithTimeout(60*time.Second).
			WithBasicAuth("user", "pass")

		require.Equal(t, 60*time.Second, options.Timeout)

		// Check authenticator
		auth, ok := options.Authenticator.(*httpauth.BasicAuth)
		require.True(t, ok, "Expected BasicAuth authenticator")
		require.Equal(t, "user", auth.Username)
		require.Equal(t, "pass", auth.Password)
	})

	t.Run("with header auth", func(t *testing.T) {
		headers := map[string]string{
			"X-API-Key":    "api-key-123",
			"X-Custom-Key": "custom-value",
		}

		options := DefaultHTTPOptions().WithHeaderAuth(headers)

		auth, ok := options.Authenticator.(*httpauth.HeaderAuth)
		require.True(t, ok, "Expected HeaderAuth authenticator")
		require.Equal(t, "api-key-123", auth.Headers["X-API-Key"])
		require.Equal(t, "custom-value", auth.Headers["X-Custom-Key"])
	})

	t.Run("with no auth", func(t *testing.T) {
		// Start with basic auth
		options := DefaultHTTPOptions().WithBasicAuth("user", "pass")

		// Then switch to no auth
		noAuthOptions := options.WithNoAuth()

		// Check that authenticator is NoAuth
		require.Equal(t, "None", noAuthOptions.Authenticator.Name())
		require.NotEqual(t, options.Authenticator, noAuthOptions.Authenticator)
	})
}

func TestFromHTTP_GetSourceURL(t *testing.T) {
	t.Parallel()

	t.Run("source URL", func(t *testing.T) {
		// Create test server for this test
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(FunctionContent))
			require.NoError(t, err)
		}))
		defer server.Close()

		testURL := server.URL + "/script.js"

		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		sourceURL := loader.GetSourceURL()
		require.NotNil(t, sourceURL)
		require.Equal(t, testURL, sourceURL.String())
	})
}

func TestFromHTTP_ImplementsLoader(t *testing.T) {
	var _ Loader = (*FromHTTP)(nil)
}
