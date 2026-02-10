package compile

import (
	"context"
	"fmt"
	"maps"
	"slices"

	risor "github.com/deepnoodle-ai/risor/v2"
	"github.com/deepnoodle-ai/risor/v2/pkg/bytecode"
	risorCompiler "github.com/deepnoodle-ai/risor/v2/pkg/compiler"
	risorParser "github.com/deepnoodle-ai/risor/v2/pkg/parser"
)

// Compile parses and compiles the script content into bytecode
func Compile(scriptContent *string, cfg *risorCompiler.Config) (*bytecode.Code, error) {
	if scriptContent == nil {
		return nil, ErrContentNil
	}

	ast, err := risorParser.Parse(context.Background(), *scriptContent, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCompileFailed, err.Error())
	}

	bc, err := risorCompiler.Compile(ast, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCompileFailed, err.Error())
	}

	return bc, nil
}

// CompileWithGlobals parses and compiles the script content into bytecode, with custom global names
// which are needed when parsing a script that will eventually have globals injected at eval time.
// For example, if a script uses a request or response object, it needs to be compiled with those
// global names, even though they won't be available until eval time.
func CompileWithGlobals(scriptContent *string, globals []string) (*bytecode.Code, error) {
	// Start with the standard builtins env and add custom globals
	env := risor.Builtins()
	for _, g := range globals {
		if _, exists := env[g]; !exists {
			env[g] = nil
		}
	}

	globalNames := slices.Sorted(maps.Keys(env))

	cfg := &risorCompiler.Config{
		GlobalNames: globalNames,
	}

	return Compile(scriptContent, cfg)
}
