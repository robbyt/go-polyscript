package engine

import (
	"context"

	"github.com/robbyt/go-polyscript/execution/script"
)

// Evaluator is the interface for the generic code evaluator.
type Evaluator interface {
	// Eval takes a context (for cancellation), an ExecutableUnit, and a map of runtime data (string is the top-level scope variable name, value is the object)
	Eval(ctx context.Context, exe *script.ExecutableUnit) (EvaluatorResponse, error)
}
