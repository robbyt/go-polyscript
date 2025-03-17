package helpers

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewHttpRequestWrapper tests the newHttpRequestWrapper function.
func TestNewHttpRequestWrapper(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		body := "test body"
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/test?query=1", bytes.NewBufferString(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Header", "test-value")
		req.Host = "localhost:8080"
		req.RemoteAddr = "127.0.0.1:12345"

		reqStruct, err := newHttpRequestWrapper(req)
		require.NoError(t, err)
		require.NotNil(t, reqStruct)

		require.Equal(t, http.MethodPost, reqStruct.Method)
		require.Equal(t, "http://localhost:8080/test?query=1", reqStruct.URL.String())
		require.Equal(t, "HTTP/1.1", reqStruct.Proto)
		require.Equal(t, int64(len(body)), reqStruct.ContentLength)
		require.Equal(t, "localhost:8080", reqStruct.Host)
		require.Equal(t, "127.0.0.1:12345", reqStruct.RemoteAddr)
		require.Equal(t, map[string][]string{"query": {"1"}}, reqStruct.QueryParams)
		require.Equal(t, map[string][]string{
			"Content-Type":  {"application/json"},
			"X-Test-Header": {"test-value"},
		}, reqStruct.Headers)
		require.Equal(t, body, reqStruct.Body)
	})

	t.Run("no body", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/test", nil)
		require.NoError(t, err)

		reqStruct, err := newHttpRequestWrapper(req)
		require.NoError(t, err)
		require.NotNil(t, reqStruct)

		require.Equal(t, http.MethodGet, reqStruct.Method)
		require.Equal(t, "http://localhost:8080/test", reqStruct.URL.String())
		require.Equal(t, "HTTP/1.1", reqStruct.Proto)
		require.Equal(t, int64(0), reqStruct.ContentLength)
		require.Equal(t, "localhost:8080", reqStruct.Host)
		require.Equal(t, "", reqStruct.RemoteAddr)
		require.Equal(t, map[string][]string{}, reqStruct.QueryParams)
		require.Equal(t, map[string][]string{}, reqStruct.Headers)
		require.Equal(t, "", reqStruct.Body)
	})

	t.Run("with query parameters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/test?param1=value1&param2=value2", nil)
		require.NoError(t, err)

		reqStruct, err := newHttpRequestWrapper(req)
		require.NoError(t, err)
		require.NotNil(t, reqStruct)

		require.Equal(t, http.MethodGet, reqStruct.Method)
		require.Equal(t, "http://localhost:8080/test?param1=value1&param2=value2", reqStruct.URL.String())
		require.Equal(t, "HTTP/1.1", reqStruct.Proto)
		require.Equal(t, int64(0), reqStruct.ContentLength)
		require.Equal(t, "localhost:8080", reqStruct.Host)
		require.Equal(t, "", reqStruct.RemoteAddr)
		require.Equal(t, map[string][]string{
			"param1": {"value1"},
			"param2": {"value2"},
		}, reqStruct.QueryParams)
		require.Equal(t, map[string][]string{}, reqStruct.Headers)
		require.Equal(t, "", reqStruct.Body)
	})

	t.Run("nil request", func(t *testing.T) {
		reqStruct, err := newHttpRequestWrapper(nil)
		require.Error(t, err)
		require.Nil(t, reqStruct)
		require.Equal(t, "request is nil", err.Error())
	})
}

// TestRequestToMap tests the RequestToMap function.
func TestRequestToMap(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		body := "test body"
		req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/test?query=1", bytes.NewBufferString(body))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Header", "test-value")
		req.Host = "localhost:8080"
		req.RemoteAddr = "127.0.0.1:12345"

		result, err := RequestToMap(req)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Equal(t, http.MethodPost, result["Method"])
		require.Equal(t, "http://localhost:8080/test?query=1", result["URL_String"])
		require.Equal(t, "HTTP/1.1", result["Proto"])
		require.Equal(t, int64(len(body)), result["ContentLength"])
		require.Equal(t, "localhost:8080", result["Host"])
		require.Equal(t, "127.0.0.1:12345", result["RemoteAddr"])
		require.Equal(t, map[string][]string{"query": {"1"}}, result["QueryParams"])
		require.Equal(t, map[string][]string{
			"Content-Type":  {"application/json"},
			"X-Test-Header": {"test-value"},
		}, result["Headers"])
		require.Equal(t, body, result["Body"])
	})

	t.Run("no body", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/test", nil)
		require.NoError(t, err)

		result, err := RequestToMap(req)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Equal(t, http.MethodGet, result["Method"])
		require.Equal(t, "http://localhost:8080/test", result["URL_String"])
		require.Equal(t, "HTTP/1.1", result["Proto"])
		require.Equal(t, int64(0), result["ContentLength"])
		require.Equal(t, "localhost:8080", result["Host"])
		require.Equal(t, "", result["RemoteAddr"])
		require.Equal(t, map[string][]string{}, result["QueryParams"])
		require.Equal(t, map[string][]string{}, result["Headers"])
		require.Equal(t, "", result["Body"])
	})

	t.Run("with query parameters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/test?param1=value1&param2=value2", nil)
		require.NoError(t, err)

		result, err := RequestToMap(req)
		require.NoError(t, err)
		require.NotNil(t, result)

		require.Equal(t, http.MethodGet, result["Method"])
		require.Equal(t, "http://localhost:8080/test?param1=value1&param2=value2", result["URL_String"])
		require.Equal(t, "HTTP/1.1", result["Proto"])
		require.Equal(t, int64(0), result["ContentLength"])
		require.Equal(t, "localhost:8080", result["Host"])
		require.Equal(t, "", result["RemoteAddr"])
		require.Equal(t, map[string][]string{
			"param1": {"value1"},
			"param2": {"value2"},
		}, result["QueryParams"])
		require.Equal(t, map[string][]string{}, result["Headers"])
		require.Equal(t, "", result["Body"])
	})
}
