package loader

import (
	"context"
	"io"
	"net/url"
)

// Loader is an interface used by the engines to load scripts or binaries.
// GetReader accepts a context so loaders performing real I/O (HTTP, disk)
// can honor caller cancellation; in-memory loaders ignore it.
type Loader interface {
	GetReader(ctx context.Context) (io.ReadCloser, error)
	GetSourceURL() *url.URL
}
