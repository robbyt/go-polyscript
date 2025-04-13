package loader

import (
	"testing"
)

// TestMockLoaderImplementsLoaderInterface ensures that MockLoader correctly implements the Loader interface
func TestMockLoaderImplementsLoaderInterface(t *testing.T) {
	// This is a compile-time check to ensure MockLoader implements Loader interface
	var _ Loader = (*MockLoader)(nil)

	// No need for further testing as the mock implementation is handled by testify/mock
	// and will be tested indirectly when used in other tests
}
