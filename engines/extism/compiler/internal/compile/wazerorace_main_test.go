package compile

import (
	"os"
	"testing"

	"github.com/robbyt/go-polyscript/internal/wazerorace"
)

// TestMain works around a wazero v1.12.0 data race; see package wazerorace.
func TestMain(m *testing.M) {
	os.Exit(wazerorace.Run(m))
}
