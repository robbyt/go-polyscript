package loader

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

type FromString struct {
	content   string
	sourceURL *url.URL
}

func NewFromString(content string) (*FromString, error) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, fmt.Errorf("%w: content is empty", ErrScriptNotAvailable)
	}

	u, err := url.Parse("string://")
	if err != nil {
		return nil, fmt.Errorf("this should never happen: %w", err)
	}

	return &FromString{
		content:   content,
		sourceURL: u,
	}, nil
}

func (l *FromString) String() string {
	return fmt.Sprintf("loader.FromString{Chars: %d}", len(l.content))
}

func (l *FromString) GetReader() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(l.content)), nil
}

// GetSourceURL returns the source URL of the script.
func (l *FromString) GetSourceURL() *url.URL {
	return l.sourceURL
}
