package snippets

import (
	"context"
	"fmt"

	"github.com/robbyt/go-polyscript/platform"
)

// printResult returns err if it is set, and otherwise prints the value of
// result so an Example can check it.
func printResult(result platform.EvaluatorResponse, err error) error {
	if err != nil {
		return err
	}
	fmt.Println(result.Interface())
	return nil
}

// evalAndPrint returns err if it is set, and otherwise evaluates eval and
// prints the result.
func evalAndPrint(ctx context.Context, eval platform.EvalOnly, err error) error {
	if err != nil {
		return err
	}
	result, err := eval.Eval(ctx)
	return printResult(result, err)
}
