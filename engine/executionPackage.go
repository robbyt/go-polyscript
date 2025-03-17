package engine

import (
	"fmt"
	"time"

	"github.com/robbyt/go-polyscript/execution/script"
)

type ExecutionPackage interface {
	// GetEvaluator returns the evaluator for this script
	GetEvaluator() Evaluator

	// GetExecutableUnit returns the executable unit for this script
	GetExecutableUnit() *script.ExecutableUnit

	// GetEvalTimeout returns the timeout for this script
	GetEvalTimeout() time.Duration
}

// executionPackage is the concrete implementation of ExecutionPackage
type executionPackage struct {
	evaluator   Evaluator
	unit        *script.ExecutableUnit
	evalTimeout time.Duration
}

// NewScriptContext creates a new ScriptContext
func NewExecutionPackage(evaluator Evaluator, unit *script.ExecutableUnit, evalTimeout time.Duration) *executionPackage {
	return &executionPackage{
		evaluator:   evaluator,
		unit:        unit,
		evalTimeout: evalTimeout,
	}
}

func (sc *executionPackage) String() string {
	return fmt.Sprintf("engine.ExecutionPackage{Evaluator: %s, ExecutableUnit: %s}", sc.evaluator, sc.unit)
}

// GetEvaluator returns a evaluator that can run the associated executable unit
func (sc *executionPackage) GetEvaluator() Evaluator {
	return sc.evaluator
}

// GetExecutableUnit returns an executable unit (bytecode, source)
func (sc *executionPackage) GetExecutableUnit() *script.ExecutableUnit {
	return sc.unit
}

// GetEvalTimeout returns the timeout for this script
func (sc *executionPackage) GetEvalTimeout() time.Duration {
	return sc.evalTimeout
}
