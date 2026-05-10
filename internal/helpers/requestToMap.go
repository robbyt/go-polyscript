package helpers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// httpRequestWrapper is a struct that mirrors the http.Request fields we are interested in.
type httpRequestWrapper struct {
	Method        string
	URL           *url.URL
	Proto         string
	Headers       map[string][]string
	Body          string
	ContentLength int64
	Host          string
	RemoteAddr    string
	QueryParams   map[string][]string
}

// resolveURL returns the request URL or a "/" sentinel when nil, so the
// caller's r.URL is never mutated.
func resolveURL(u *url.URL) *url.URL {
	if u == nil {
		return &url.URL{Path: "/"}
	}
	return u
}

// newHTTPRequestWrapper converts an http.Request to an httpRequest struct.
func newHTTPRequestWrapper(r *http.Request) (*httpRequestWrapper, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}

	urlToUse := resolveURL(r.URL)

	reqStruct := &httpRequestWrapper{
		Method:        r.Method,
		URL:           urlToUse,
		Proto:         r.Proto,
		ContentLength: r.ContentLength,
		Host:          r.Host,
		RemoteAddr:    r.RemoteAddr,
		Headers:       make(map[string][]string),
		QueryParams:   make(map[string][]string),
	}

	// Copy headers if present
	if r.Header != nil {
		for k, v := range r.Header {
			reqStruct.Headers[k] = v
		}
	}

	// Read the body. The reader is consumed once — we don't restore it,
	// since rewriting r.Body would mutate the caller's request.
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		reqStruct.Body = string(bodyBytes)
	}

	for k, v := range urlToUse.Query() {
		reqStruct.QueryParams[k] = v
	}

	return reqStruct, nil
}

// toMap converts an httpRequest struct to a map[string]any.
func (h *httpRequestWrapper) toMap() map[string]any {
	return map[string]any{
		"Method":        h.Method,
		"URL":           h.URL,
		"URL_String":    h.URL.String(),
		"URL_Host":      h.URL.Host,
		"URL_Scheme":    h.URL.Scheme,
		"URL_Path":      h.URL.Path,
		"Proto":         h.Proto,
		"Headers":       h.Headers,
		"Body":          h.Body,
		"ContentLength": h.ContentLength,
		"Host":          h.Host,
		"RemoteAddr":    h.RemoteAddr,
		"QueryParams":   h.QueryParams,
	}
}

// RequestToMap converts an http.Request to a map[string]any using the
// httpRequest struct as an intermediary.
//
// RequestToMap reads r without mutating it: r.URL and r.Body are
// observed but never reassigned. As a consequence, r.Body is consumed
// like any io.Reader — after the call it will be at EOF. Callers that
// need a re-readable body should clone it (e.g. via r.GetBody()) or
// buffer it before passing the request in.
func RequestToMap(r *http.Request) (map[string]any, error) {
	// Transform http.Request to httpRequest struct
	reqStruct, err := newHTTPRequestWrapper(r)
	if err != nil {
		return nil, fmt.Errorf("failed to transform http.Request to httpRequest struct: %w", err)
	}

	if reqStruct == nil {
		return nil, errors.New(
			"failed to transform http.Request to httpRequest struct: result is nil",
		)
	}

	// Convert httpRequest struct to map[string]any
	return reqStruct.toMap(), nil
}
