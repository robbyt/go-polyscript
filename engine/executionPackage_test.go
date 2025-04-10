package engine_test

import (
	"testing"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/machines/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionPackage(t *testing.T) {
	// Setup test mocks
	mockEvaluator := new(mocks.Evaluator)
	mockUnit := &script.ExecutableUnit{}
	timeout := 5 * time.Second

	// Create execution package
	execPkg := engine.NewExecutionPackage(mockEvaluator, mockUnit, timeout)

	// Assert execution package was created correctly
	assert.NotNil(t, execPkg, "Execution package should not be nil")
	assert.Equal(t, mockEvaluator, execPkg.GetEvaluator(), "Should return the provided evaluator")
	assert.Equal(
		t,
		mockUnit,
		execPkg.GetExecutableUnit(),
		"Should return the provided executable unit",
	)
	assert.Equal(t, timeout, execPkg.GetEvalTimeout(), "Should return the provided timeout")
}

func TestExecutionPackage_String(t *testing.T) {
	// Setup test mocks
	mockEvaluator := new(mocks.Evaluator)
	mockUnit := &script.ExecutableUnit{}
	timeout := 5 * time.Second

	// Create execution package
	execPkg := engine.NewExecutionPackage(mockEvaluator, mockUnit, timeout)

	// Test String method
	stringRep := execPkg.String()
	assert.Contains(
		t,
		stringRep,
		"engine.ExecutionPackage",
		"String representation should contain type information",
	)
	assert.Contains(t, stringRep, "Evaluator", "String representation should mention evaluator")
	assert.Contains(
		t,
		stringRep,
		"ExecutableUnit",
		"String representation should mention executable unit",
	)
}

func TestExecutionPackage_Getters(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name        string
		evaluator   engine.Evaluator
		unit        *script.ExecutableUnit
		evalTimeout time.Duration
	}{
		{
			name:        "Standard values",
			evaluator:   new(mocks.Evaluator),
			unit:        &script.ExecutableUnit{},
			evalTimeout: 5 * time.Second,
		},
		{
			name:        "Zero timeout",
			evaluator:   new(mocks.Evaluator),
			unit:        &script.ExecutableUnit{},
			evalTimeout: 0,
		},
		{
			name:        "Negative timeout (should still work, though not recommended)",
			evaluator:   new(mocks.Evaluator),
			unit:        &script.ExecutableUnit{},
			evalTimeout: -1 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create execution package
			execPkg := engine.NewExecutionPackage(tc.evaluator, tc.unit, tc.evalTimeout)

			// Test getter methods
			assert.Equal(t, tc.evaluator, execPkg.GetEvaluator(),
				"GetEvaluator should return the provided evaluator")
			assert.Equal(t, tc.unit, execPkg.GetExecutableUnit(),
				"GetExecutableUnit should return the provided executable unit")
			assert.Equal(t, tc.evalTimeout, execPkg.GetEvalTimeout(),
				"GetEvalTimeout should return the provided timeout")
		})
	}
}

func TestExecutionPackage_WithNilValues(t *testing.T) {
	// Test with nil evaluator (not recommended but should still create the package)
	execPkgNilEval := engine.NewExecutionPackage(nil, &script.ExecutableUnit{}, 5*time.Second)
	assert.NotNil(t, execPkgNilEval, "Should create package even with nil evaluator")
	assert.Nil(t, execPkgNilEval.GetEvaluator(), "GetEvaluator should return nil when provided nil")

	// Test with nil unit (not recommended but should still create the package)
	execPkgNilUnit := engine.NewExecutionPackage(new(mocks.Evaluator), nil, 5*time.Second)
	assert.NotNil(t, execPkgNilUnit, "Should create package even with nil unit")
	assert.Nil(
		t,
		execPkgNilUnit.GetExecutableUnit(),
		"GetExecutableUnit should return nil when provided nil",
	)

	// Test String method with nil values
	stringRep := execPkgNilEval.String()
	require.NotEmpty(t, stringRep, "String method should handle nil values without panicking")
}
