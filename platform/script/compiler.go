package script

import (
	"context"
	"io"
)

// Compiler defines the interface for validating scripts before execution.
// It checks syntax and semantics, and may perform parsing, compilation,
// and optimization. A valid script is returned as ExecutableContent.
type Compiler interface {
	// Compile checks if a script is valid and returns it as executable content.
	// The returned ExecutableContent contains the validated and possibly compiled
	// script ready for execution. Cancellation of ctx halts the parser/compiler.
	Compile(ctx context.Context, scriptReader io.ReadCloser) (ExecutableContent, error)
}
