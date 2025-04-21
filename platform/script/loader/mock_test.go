package loader

import (
	"testing"
)

// TestMockLoaderImplementsLoaderInterface ensures that MockLoader correctly implements the Loader interface
func TestMockLoaderImplementsLoaderInterface(t *testing.T) {
	var _ Loader = (*MockLoader)(nil)
}
