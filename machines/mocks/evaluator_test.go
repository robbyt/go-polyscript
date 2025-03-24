package mocks

import (
	"testing"

	"github.com/robbyt/go-polyscript/engine"
)

// TestEvaluatorImplementsEvaluatorWithPrep verifies at compile time
// that our mock Evaluator implements the EvaluatorWithPrep interface.
func TestEvaluatorImplementsEvaluatorWithPrep(t *testing.T) {
	// This is a compile-time check - if it doesn't compile, the test fails
	var _ engine.EvaluatorWithPrep = (*Evaluator)(nil)
}
