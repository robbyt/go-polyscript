package risor

import (
	"context"
	"errors"
	"fmt"

	risorLib "github.com/risor-io/risor"
	risorCompiler "github.com/risor-io/risor/compiler"
	risorErrors "github.com/risor-io/risor/errz"
	risorParser "github.com/risor-io/risor/parser"
)

// compile parses and compiles the script content into bytecode
func compile(scriptContent *string, options ...risorCompiler.Option) (*risorCompiler.Code, error) {
	if scriptContent == nil {
		return nil, fmt.Errorf("script content is nil")
	}

	ast, err := risorParser.Parse(context.Background(), *scriptContent)
	if err != nil {
		// Create a better-looking error output when there's a syntax error
		errMsg := err.Error()
		var friendlyErr risorErrors.FriendlyError
		if errors.As(err, &friendlyErr) {
			errMsg = friendlyErr.FriendlyErrorMessage()
		}
		return nil, fmt.Errorf("compilation: %s", errMsg)
	}

	// Compile the AST to bytecode
	bc, err := risorCompiler.Compile(ast, options...)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// compileWithGlobals parses and compiles the script content into bytecode, with custom global names
// which are needed when parsing a script that will eventually have globals injected at eval time.
// For example, if a script uses a request or response object, it needs to be compiled with those
// global names, even though they won't be available until eval time.
func compileWithGlobals(scriptContent *string, globals []string) (*risorCompiler.Code, error) {
	// Retrieve default global names, and append the custom globals
	cfg := risorLib.NewConfig()
	globalNames := append(cfg.GlobalNames(), globals...)

	options := []risorCompiler.Option{
		risorCompiler.WithGlobalNames(globalNames),
	}

	return compile(scriptContent, options...)
}
