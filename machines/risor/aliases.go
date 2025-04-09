package risor

import (
	"github.com/robbyt/go-polyscript/machines/risor/compiler"
	"github.com/robbyt/go-polyscript/machines/risor/evaluator"
)

type BytecodeEvaluator = evaluator.BytecodeEvaluator

var NewBytecodeEvaluator = evaluator.NewBytecodeEvaluator

type Compiler = compiler.Compiler

var NewCompiler = compiler.NewCompiler
