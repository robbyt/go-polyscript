package loader

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// Standard test content strings used across loader tests
const (
	SimpleContent    = "test content"
	MultilineContent = "line1\nline2\nline3"
	FunctionContent  = "function test(x) { return x * 2; }"
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

// verifyLoader performs common verification steps for all loader implementations
func verifyLoader(t *testing.T, loader Loader, expectedURLString string) {
	t.Helper()

	// Verify loader is properly instantiated
	require.NotNil(t, loader)

	// Verify source URL
	sourceURL := loader.GetSourceURL()
	require.NotNil(t, sourceURL)

	if expectedURLString != "" {
		parsedURL, err := url.Parse(expectedURLString)
		require.NoError(t, err)
		require.Equal(t, parsedURL.Scheme, sourceURL.Scheme)
	}

	// Test getting a reader
	reader, err := loader.GetReader()
	if err == nil {
		// If no error, verify reader works and cleanup
		require.NotNil(t, reader)
		t.Cleanup(func() {
			require.NoError(t, reader.Close(), "Failed to close reader")
		})
	}
}

// verifyReaderContent verifies the content returned by a reader
func verifyReaderContent(t *testing.T, reader io.ReadCloser, expectedContent string) {
	t.Helper()

	// Add cleanup to ensure reader is closed
	t.Cleanup(func() {
		require.NoError(t, reader.Close(), "Failed to close reader")
	})

	// Read content
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content))
}

// verifyMultipleReads tests that a loader can provide multiple readers
// with the same content
func verifyMultipleReads(t *testing.T, loader Loader, expectedContent string) {
	t.Helper()

	// First read
	reader1, err := loader.GetReader()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, reader1.Close(), "Failed to close first reader")
	})

	content1, err := io.ReadAll(reader1)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content1))

	// Second read
	reader2, err := loader.GetReader()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, reader2.Close(), "Failed to close second reader")
	})

	content2, err := io.ReadAll(reader2)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content2))
}
