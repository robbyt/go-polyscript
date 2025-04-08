package starlark

import (
	"github.com/robbyt/go-polyscript/machines/starlark/compiler"
	"github.com/robbyt/go-polyscript/machines/starlark/evaluator"
)

type BytecodeEvaluator = evaluator.BytecodeEvaluator

var NewBytecodeEvaluator = evaluator.NewBytecodeEvaluator

type Compiler = compiler.Compiler

var NewCompiler = compiler.NewCompiler
