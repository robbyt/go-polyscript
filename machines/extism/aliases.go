package extism

import (
	"github.com/robbyt/go-polyscript/machines/extism/compiler"
	"github.com/robbyt/go-polyscript/machines/extism/evaluator"
)

type BytecodeEvaluator = evaluator.BytecodeEvaluator

var NewBytecodeEvaluator = evaluator.NewBytecodeEvaluator

type Compiler = compiler.Compiler

var NewCompiler = compiler.NewCompiler
