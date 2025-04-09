package compile

import (
	"context"
	"errors"
	"fmt"

	risorLib "github.com/risor-io/risor"
	risorCompiler "github.com/risor-io/risor/compiler"
	risorErrors "github.com/risor-io/risor/errz"
	risorParser "github.com/risor-io/risor/parser"
)

// Compile parses and compiles the script content into bytecode
func Compile(scriptContent *string, options ...risorCompiler.Option) (*risorCompiler.Code, error) {
	if scriptContent == nil {
		return nil, ErrContentNil
	}

	ast, err := risorParser.Parse(context.Background(), *scriptContent)
	if err != nil {
		// Create a better-looking error output when there's a syntax error
		errMsg := err.Error()
		var friendlyErr risorErrors.FriendlyError
		if errors.As(err, &friendlyErr) {
			errMsg = friendlyErr.FriendlyErrorMessage()
		}
		return nil, fmt.Errorf("%w: %s", ErrCompileFailed, errMsg)
	}

	// Compile the AST to bytecode
	bc, err := risorCompiler.Compile(ast, options...)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// CompileWithGlobals parses and compiles the script content into bytecode, with custom global names
// which are needed when parsing a script that will eventually have globals injected at eval time.
// For example, if a script uses a request or response object, it needs to be compiled with those
// global names, even though they won't be available until eval time.
func CompileWithGlobals(scriptContent *string, globals []string) (*risorCompiler.Code, error) {
	// Retrieve default global names, and append the custom globals
	cfg := risorLib.NewConfig()
	globalNames := append(cfg.GlobalNames(), globals...)

	options := []risorCompiler.Option{
		risorCompiler.WithGlobalNames(globalNames),
	}

	return Compile(scriptContent, options...)
}
