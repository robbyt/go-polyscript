package loader

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/robbyt/go-polyscript/internal/helpers"
)

// FromBytes implements the Loader interface for content from a byte slice.
type FromBytes struct {
	content   []byte
	sourceURL *url.URL
}

// NewFromBytes creates a new Loader from a byte slice.
func NewFromBytes(content []byte) (*FromBytes, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("%w: content is empty", ErrScriptNotAvailable)
	}

	// Check if content contains only whitespace
	contentStr := string(content)
	contentStr = strings.TrimSpace(contentStr)
	if len(contentStr) == 0 {
		return nil, fmt.Errorf(
			"%w: content is empty or contains only whitespace",
			ErrScriptNotAvailable,
		)
	}

	// Create source URL with a unique identifier based on content
	u, err := url.Parse("bytes://inline/" + helpers.SHA256(string(content))[:8])
	if err != nil {
		return nil, fmt.Errorf("failed to create source URL: %w", err)
	}

	return &FromBytes{
		content:   content,
		sourceURL: u,
	}, nil
}

func (l *FromBytes) String() string {
	return fmt.Sprintf("loader.FromBytes{Bytes: %d}", len(l.content))
}

// GetReader returns a new reader for the stored content.
func (l *FromBytes) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(l.content)), nil
}

// GetSourceURL returns the source URL of the script.
func (l *FromBytes) GetSourceURL() *url.URL {
	return l.sourceURL
}
