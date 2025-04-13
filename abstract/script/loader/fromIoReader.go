package loader

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// FromIoReader implements the Loader interface for content from an io.Reader.
type FromIoReader struct {
	content   []byte
	sourceURL *url.URL
}

// NewFromIoReader creates a new Loader from an io.Reader source.
// The entire reader content is read and stored to allow multiple GetReader calls.
func NewFromIoReader(reader io.Reader, sourceName string) (*FromIoReader, error) {
	if reader == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrScriptNotAvailable)
	}

	// Read all content from reader
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	// Check if content is empty or contains only whitespace
	contentStr := string(content)
	contentStr = strings.TrimSpace(contentStr)
	if len(contentStr) == 0 {
		return nil, fmt.Errorf(
			"%w: content is empty or contains only whitespace",
			ErrScriptNotAvailable,
		)
	}

	// Create source URL with identifier based on content
	urlStr := "reader://"
	if sourceName != "" {
		urlStr += sourceName + "/"
	} else {
		urlStr += "unnamed/"
	}
	urlStr += helpers.SHA256(string(content))[:8]

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create source URL: %w", err)
	}

	return &FromIoReader{
		content:   content,
		sourceURL: u,
	}, nil
}

func (l *FromIoReader) String() string {
	return fmt.Sprintf(
		"loader.FromIoReader{Bytes: %d, Source: %s}",
		len(l.content),
		l.sourceURL.String(),
	)
}

// GetReader returns a new reader for the stored content.
func (l *FromIoReader) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

// GetSourceURL returns the source URL of the script.
func (l *FromIoReader) GetSourceURL() *url.URL {
	return l.sourceURL
}
