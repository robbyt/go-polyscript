package loader

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/execution/script/loader/httpauth"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient implements the httpRequester interface for testing
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, errors.New("doFunc not implemented")
}

// mockResponseBody implements io.ReadCloser for testing
type mockResponseBody struct {
	io.Reader
	closeFunc func() error
	closed    bool
}

func (m *mockResponseBody) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true

	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// newMockResponse creates a new mock HTTP response
func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       &mockResponseBody{Reader: bytes.NewBufferString(body)},
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
	}
}

func TestNewFromHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		url           string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid HTTPS URL",
			url:         "https://example.com/script.js",
			expectError: false,
		},
		{
			name:        "Valid HTTP URL",
			url:         "http://example.com/script.js",
			expectError: false,
		},
		{
			name:          "Invalid URL scheme",
			url:           "file:///path/to/script.js",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
		{
			name:          "Invalid URL format",
			url:           "://invalid-url",
			expectError:   true,
			errorContains: "unable to parse URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewFromHTTP(tt.url)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)
			require.Equal(t, tt.url, loader.url)
			require.NotNil(t, loader.sourceURL)
			require.Equal(t, tt.url, loader.sourceURL.String())
			require.NotNil(t, loader.client)
			require.NotNil(t, loader.options)
		})
	}
}

func TestNewFromHTTPWithOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		url             string
		optionsModifier func(options *HTTPOptions) *HTTPOptions
		validateOption  func(t *testing.T, loader *FromHTTP)
		expectError     bool
		errorContains   string
	}{
		{
			name: "Custom timeout",
			url:  "https://example.com/script.js",
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
			url:  "https://example.com/script.js",
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
			url:  "https://example.com/script.js",
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
			url:  "https://example.com/script.js",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with default options and apply modifier if provided
			options := DefaultHTTPOptions()
			if tt.optionsModifier != nil {
				options = tt.optionsModifier(options)
			}

			loader, err := NewFromHTTPWithOptions(tt.url, options)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loader)
			require.Equal(t, tt.url, loader.url)
			require.NotNil(t, loader.sourceURL)
			require.Equal(t, tt.url, loader.sourceURL.String())
			require.NotNil(t, loader.client)
			require.NotNil(t, loader.options)

			if tt.validateOption != nil {
				tt.validateOption(t, loader)
			}
		})
	}
}

func TestFromHTTP_TLSConfig(t *testing.T) {
	t.Parallel()

	t.Run("with insecure skip verify", func(t *testing.T) {
		options := DefaultHTTPOptions()
		options.InsecureSkipVerify = true

		loader, err := NewFromHTTPWithOptions("https://example.com/script.js", options)
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
		options := DefaultHTTPOptions()
		customTLS := &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Add custom ciphers
			CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		}
		options.TLSConfig = customTLS

		loader, err := NewFromHTTPWithOptions("https://example.com/script.js", options)
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
		options := DefaultHTTPOptions()
		options.InsecureSkipVerify = true
		customTLS := &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS13,
		}
		options.TLSConfig = customTLS

		loader, err := NewFromHTTPWithOptions("https://example.com/script.js", options)
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
	})

	t.Run("no TLS modifications when neither option is set", func(t *testing.T) {
		options := DefaultHTTPOptions()

		loader, err := NewFromHTTPWithOptions("https://example.com/script.js", options)
		require.NoError(t, err)
		require.NotNil(t, loader)

		// When no TLS options are set, the client uses the default transport without modifications
		client, ok := loader.client.(*http.Client)
		require.True(t, ok, "Expected http.Client")

		// The http.Client only initializes Transport when needed, so it might be nil at this point
		// We should check that it's nil when neither TLS option is set
		require.Nil(t, client.Transport, "Expected Transport to be nil when no TLS options are set")
	})
}

func TestFromHTTP_GetReader(t *testing.T) {
	t.Parallel()

	const testScript = `function test() { return "Hello, World!"; }`

	tests := []struct {
		name             string
		url              string
		optionsModifier  func(options *HTTPOptions) *HTTPOptions
		customResp       func() *http.Response
		mockError        error
		requestValidator func(t *testing.T, req *http.Request)
		expectError      bool
		errorContains    string
		validateBody     bool
	}{
		{
			name: "Success - Default",
			url:  "https://example.com/script.js",
			customResp: func() *http.Response {
				return newMockResponse(http.StatusOK, testScript)
			},
			requestValidator: func(t *testing.T, req *http.Request) {
				t.Helper()
				require.Equal(t, "https://example.com/script.js", req.URL.String())
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "go-polyscript/http-loader", req.Header.Get("User-Agent"))
			},
			validateBody: true,
		},
		{
			name: "Success - Basic Auth",
			url:  "https://example.com/auth",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				return options.WithBasicAuth("user", "pass").WithTimeout(5 * time.Second)
			},
			customResp: func() *http.Response {
				return newMockResponse(http.StatusOK, testScript)
			},
			requestValidator: func(t *testing.T, req *http.Request) {
				t.Helper()
				require.Equal(t, "https://example.com/auth", req.URL.String())
				username, password, ok := req.BasicAuth()
				require.True(t, ok, "Expected Basic Auth to be set")
				require.Equal(t, "user", username)
				require.Equal(t, "pass", password)
			},
			validateBody: true,
		},
		{
			name: "Success - Bearer Auth",
			url:  "https://example.com/header-auth",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				return options.WithBearerAuth("test-token").WithTimeout(5 * time.Second)
			},
			customResp: func() *http.Response {
				return newMockResponse(http.StatusOK, testScript)
			},
			requestValidator: func(t *testing.T, req *http.Request) {
				t.Helper()
				require.Equal(t, "https://example.com/header-auth", req.URL.String())
				require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			},
			validateBody: true,
		},
		{
			name: "Success - Custom Headers",
			url:  "https://example.com/header-auth",
			optionsModifier: func(options *HTTPOptions) *HTTPOptions {
				options.Headers["User-Agent"] = "Custom-Agent"
				options.Headers["X-Custom"] = "value"
				return options
			},
			customResp: func() *http.Response {
				return newMockResponse(http.StatusOK, testScript)
			},
			requestValidator: func(t *testing.T, req *http.Request) {
				t.Helper()
				require.Equal(t, "Custom-Agent", req.Header.Get("User-Agent"))
				require.Equal(t, "value", req.Header.Get("X-Custom"))
			},
			validateBody: true,
		},
		{
			name: "Failure - Unauthorized",
			url:  "https://example.com/auth",
			customResp: func() *http.Response {
				return newMockResponse(http.StatusUnauthorized, "Unauthorized")
			},
			expectError:   true,
			errorContains: "HTTP 401",
		},
		{
			name: "Failure - Not Found",
			url:  "https://example.com/error",
			customResp: func() *http.Response {
				return newMockResponse(http.StatusNotFound, "Not Found")
			},
			expectError:   true,
			errorContains: "HTTP 404",
		},
		{
			name:          "Failure - Network Error",
			url:           "https://invalid-domain.example",
			mockError:     errors.New("network error"),
			expectError:   true,
			errorContains: "failed to execute HTTP request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client for this test
			mockClient := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					if tt.requestValidator != nil {
						tt.requestValidator(t, req)
					}

					if tt.mockError != nil {
						return nil, tt.mockError
					}

					resp := tt.customResp()
					return resp, nil
				},
			}

			var loader *FromHTTP
			var err error

			// Start with default options and apply modifier if provided
			options := DefaultHTTPOptions()
			if tt.optionsModifier != nil {
				options = tt.optionsModifier(options)
			}

			loader, err = NewFromHTTPWithOptions(tt.url, options)
			require.NoError(t, err, "Failed to create HTTP loader")

			// Replace the client with our mock
			loader.client = mockClient

			reader, err := loader.GetReader()
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, reader)
			defer func() { require.NoError(t, reader.Close(), "Failed to close reader") }()

			if tt.validateBody {
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.Equal(t, testScript, string(content))
			}
		})
	}
}

func TestFromHTTP_GetReaderWithContext(t *testing.T) {
	t.Parallel()

	const testScript = `function test() { return "Hello, World!"; }`

	tests := []struct {
		name          string
		url           string
		ctx           context.Context
		cancelFunc    func()
		expectError   bool
		errorContains string
	}{
		{
			name: "Success - Background Context",
			url:  "https://example.com/script.js",
			ctx:  context.Background(),
		},
		{
			name:          "Failure - Cancelled Context",
			url:           "https://example.com/script.js",
			ctx:           func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			expectError:   true,
			errorContains: "context canceled",
		},
		{
			name: "Failure - Timeout Context",
			url:  "https://example.com/script.js",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(5 * time.Millisecond)
				return ctx
			}(),
			expectError:   true,
			errorContains: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewFromHTTP(tt.url)
			require.NoError(t, err)

			mockClient := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					// Check if context error happens before we'd even make the request
					if err := req.Context().Err(); err != nil {
						return nil, err
					}
					// Create a new response each time to ensure it can be properly closed
					resp := newMockResponse(http.StatusOK, testScript)
					return resp, nil
				},
			}
			loader.client = mockClient

			reader, err := loader.GetReaderWithContext(tt.ctx)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, reader)
			defer func() { require.NoError(t, reader.Close(), "Failed to close reader") }()
		})
	}
}

func TestFromHTTP_String(t *testing.T) {
	t.Parallel()

	t.Run("successful string representation", func(t *testing.T) {
		// Test successful String() result with mock client
		testURL := "https://example.com/script.js"
		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Mock client that returns content for SHA256 calculation
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return newMockResponse(http.StatusOK, "test script content"), nil
			},
		}
		loader.client = mockClient

		str := loader.String()
		require.Contains(t, str, "loader.FromHTTP{URL:")
		require.Contains(t, str, testURL)
		require.Contains(t, str, "SHA256:")
	})

	t.Run("string representation with network error", func(t *testing.T) {
		testURL := "https://example.com/script.js"
		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Mock client that simulates an error
		failingMockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}
		loader.client = failingMockClient

		str := loader.String()
		require.Contains(t, str, "loader.FromHTTP{URL:")
		require.Contains(t, str, testURL)
		require.NotContains(t, str, "SHA256")
	})

	t.Run("string representation with HTTP error", func(t *testing.T) {
		testURL := "https://example.com/script.js"
		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		// Mock client that returns an error status code
		errorMockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return newMockResponse(http.StatusNotFound, "Not Found"), nil
			},
		}
		loader.client = errorMockClient

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
		testURL := "https://example.com/script.js"
		loader, err := NewFromHTTP(testURL)
		require.NoError(t, err)

		sourceURL := loader.GetSourceURL()
		require.NotNil(t, sourceURL)
		require.Equal(t, testURL, sourceURL.String())

		// Test that the returned URL is a copy that can't modify the internal state
		parsedURL, err := url.Parse(testURL)
		require.NoError(t, err)
		require.Equal(t, parsedURL, sourceURL)
	})
}
