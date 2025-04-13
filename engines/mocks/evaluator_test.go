package mocks

import (
	"testing"

	"github.com/robbyt/go-polyscript/abstract/evaluation"
)

// TestEvaluatorImplementsEvaluatorWithPrep verifies at compile time
// that our mock Evaluator implements the EvaluatorWithPrep interface.
func TestEvaluatorImplementsEvaluatorWithPrep(t *testing.T) {
	t.Parallel()
	// This is a compile-time check - if it doesn't compile, the test fails
	var _ evaluation.EvaluatorWithPrep = (*Evaluator)(nil)
}
