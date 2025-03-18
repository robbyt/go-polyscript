package loader

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

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
}

func (m mockResponseBody) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

// newMockResponse creates a new mock HTTP response
func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       mockResponseBody{Reader: bytes.NewBufferString(body)},
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
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
		name           string
		url            string
		options        *HTTPOptions
		validateOption func(t *testing.T, loader *FromHTTP)
		expectError    bool
		errorContains  string
	}{
		{
			name: "Custom timeout",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				Timeout: 60 * time.Second,
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, 60*time.Second, loader.options.Timeout)
			},
		},
		{
			name: "Basic auth",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				AuthType: BasicAuth,
				Username: "user",
				Password: "pass",
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, BasicAuth, loader.options.AuthType)
				require.Equal(t, "user", loader.options.Username)
				require.Equal(t, "pass", loader.options.Password)
			},
		},
		{
			name: "Custom headers",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				AuthType: HeaderAuth,
				Headers: map[string]string{
					"Authorization": "Bearer token",
					"User-Agent":    "Test-Agent",
				},
			},
			validateOption: func(t *testing.T, loader *FromHTTP) {
				require.Equal(t, HeaderAuth, loader.options.AuthType)
				require.Equal(t, "Bearer token", loader.options.Headers["Authorization"])
				require.Equal(t, "Test-Agent", loader.options.Headers["User-Agent"])
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use either provided options or default options
			options := tt.options
			if options == nil {
				options = DefaultHTTPOptions()
			} else {
				// Fill in any unset fields with defaults
				defaultOptions := DefaultHTTPOptions()
				if options.Timeout == 0 {
					options.Timeout = defaultOptions.Timeout
				}
				if options.Headers == nil {
					options.Headers = defaultOptions.Headers
				}
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

func TestFromHTTPGetReader(t *testing.T) {
	t.Parallel()

	const testScript = `function test() { return "Hello, World!"; }`

	tests := []struct {
		name             string
		url              string
		options          *HTTPOptions
		mockResponse     *http.Response
		mockError        error
		requestValidator func(t *testing.T, req *http.Request)
		expectError      bool
		errorContains    string
		validateBody     bool
	}{
		{
			name:         "Success - Default",
			url:          "https://example.com/script.js",
			mockResponse: newMockResponse(http.StatusOK, testScript),
			requestValidator: func(t *testing.T, req *http.Request) {
				require.Equal(t, "https://example.com/script.js", req.URL.String())
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "go-polyscript/http-loader", req.Header.Get("User-Agent"))
			},
			validateBody: true,
		},
		{
			name: "Success - Basic Auth",
			url:  "https://example.com/auth",
			options: &HTTPOptions{
				AuthType: BasicAuth,
				Username: "user",
				Password: "pass",
				Timeout:  5 * time.Second,
			},
			mockResponse: newMockResponse(http.StatusOK, testScript),
			requestValidator: func(t *testing.T, req *http.Request) {
				require.Equal(t, "https://example.com/auth", req.URL.String())
				username, password, ok := req.BasicAuth()
				require.True(t, ok, "Expected Basic Auth to be set")
				require.Equal(t, "user", username)
				require.Equal(t, "pass", password)
			},
			validateBody: true,
		},
		{
			name: "Success - Header Auth",
			url:  "https://example.com/header-auth",
			options: &HTTPOptions{
				AuthType: HeaderAuth,
				Headers: map[string]string{
					"Authorization": "Bearer test-token",
					"User-Agent":    "Custom-Agent",
				},
				Timeout: 5 * time.Second,
			},
			mockResponse: newMockResponse(http.StatusOK, testScript),
			requestValidator: func(t *testing.T, req *http.Request) {
				require.Equal(t, "https://example.com/header-auth", req.URL.String())
				require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
				require.Equal(t, "Custom-Agent", req.Header.Get("User-Agent"))
			},
			validateBody: true,
		},
		{
			name:          "Failure - Unauthorized",
			url:           "https://example.com/auth",
			mockResponse:  newMockResponse(http.StatusUnauthorized, "Unauthorized"),
			expectError:   true,
			errorContains: "HTTP 401",
		},
		{
			name:          "Failure - Not Found",
			url:           "https://example.com/error",
			mockResponse:  newMockResponse(http.StatusNotFound, "Not Found"),
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
		{
			name: "Failure - Digest Auth Not Implemented",
			url:  "https://example.com/script.js",
			options: &HTTPOptions{
				AuthType: DigestAuth,
				Timeout:  5 * time.Second,
			},
			expectError:   true,
			errorContains: "digest authentication not implemented",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a mock client for this test
			mockClient := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					if tt.requestValidator != nil {
						tt.requestValidator(t, req)
					}

					if tt.mockError != nil {
						return nil, tt.mockError
					}

					return tt.mockResponse, nil
				},
			}

			var loader *FromHTTP
			var err error

			if tt.options != nil {
				loader, err = NewFromHTTPWithOptions(tt.url, tt.options)
			} else {
				loader, err = NewFromHTTP(tt.url)
			}
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
			defer reader.Close()

			if tt.validateBody {
				content, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.Equal(t, testScript, string(content))
			}
		})
	}
}

func TestFromHTTPString(t *testing.T) {
	t.Parallel()

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

	// Test error case for String method
	failingLoader, err := NewFromHTTP(testURL)
	require.NoError(t, err)

	// Mock client that simulates an error
	failingMockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}
	failingLoader.client = failingMockClient

	str = failingLoader.String()
	require.Contains(t, str, "loader.FromHTTP{URL:")
	require.Contains(t, str, testURL)
	require.NotContains(t, str, "SHA256")

	// Test when HTTP response is an error code
	errorLoader, err := NewFromHTTP(testURL)
	require.NoError(t, err)

	// Mock client that returns an error status code
	errorMockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return newMockResponse(http.StatusNotFound, "Not Found"), nil
		},
	}
	errorLoader.client = errorMockClient

	str = errorLoader.String()
	require.Contains(t, str, "loader.FromHTTP{URL:")
	require.Contains(t, str, testURL)
	require.NotContains(t, str, "SHA256")
}

func TestDefaultHTTPOptions(t *testing.T) {
	t.Parallel()

	options := DefaultHTTPOptions()
	require.NotNil(t, options)
	require.Equal(t, 30*time.Second, options.Timeout)
	require.False(t, options.InsecureSkipVerify)
	require.Equal(t, NoAuth, options.AuthType)
	require.NotNil(t, options.Headers)
	require.Empty(t, options.Headers)
}

// Test the GetSourceURL method
func TestFromHTTPGetSourceURL(t *testing.T) {
	t.Parallel()

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
}
