package loader

import (
	"context"
	"io"
	"net/url"
)

// Loader is an interface used by the engines to load scripts or binaries.
// GetReader accepts a context so loaders performing cancellable I/O (HTTP)
// can honor caller cancellation. In-memory loaders ignore ctx; FromDisk
// accepts ctx for interface conformance but os.Open itself is synchronous
// and does not observe it.
type Loader interface {
	GetReader(ctx context.Context) (io.ReadCloser, error)
	GetSourceURL() *url.URL
}
