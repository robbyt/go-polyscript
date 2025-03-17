package starlark

import (
	"testing"

	"github.com/stretchr/testify/require"
	starlarkLib "go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestCompileSuccess(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		globals []string
		opts    *syntax.FileOptions
	}{
		{
			name:   "simple true",
			script: `True`,
			opts:   &syntax.FileOptions{},
		},
		{
			name:   "simple print",
			script: `print("Hello, World!")`,
			opts:   &syntax.FileOptions{},
		},
		{
			name:    "with predeclared globals",
			script:  `print(request)`,
			globals: []string{"request"},
			opts:    &syntax.FileOptions{GlobalReassign: true},
		},
		{
			name:    "with multiple globals",
			script:  `print(request, response)`,
			globals: []string{"request", "response"},
			opts:    &syntax.FileOptions{GlobalReassign: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			scriptBytes := []byte(tt.script)

			if tt.globals != nil {
				// Create predeclared globals
				predeclared := make(starlarkLib.StringDict, len(tt.globals))
				for _, name := range tt.globals {
					predeclared[name] = starlarkLib.None
				}
				_, err = compile(scriptBytes, tt.opts, predeclared)
			} else {
				_, err = compile(scriptBytes, tt.opts, nil)
			}
			require.NoError(t, err)
		})
	}
}

func TestCompileWithEmptyGlobals(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		globals []string
		wantErr bool
	}{
		{
			name:    "valid globals",
			script:  `print(request)`,
			globals: []string{"request"},
		},
		{
			name:    "predeclared conflict",
			script:  `print(true)`,
			globals: []string{},
			wantErr: true,
		},
		{
			name:    "nil script",
			globals: []string{"request"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scriptBytes []byte
			if tt.script != "" {
				scriptBytes = []byte(tt.script)
			}
			_, err := compileWithEmptyGlobals(scriptBytes, tt.globals)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCompileSyntaxError(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		globals []string
		opts    *syntax.FileOptions
		wantErr string
	}{
		{
			name:    "unclosed string",
			script:  `print("Hello, World!`,
			opts:    &syntax.FileOptions{},
			wantErr: "compilation error:",
		},
		{
			name:    "invalid syntax",
			script:  `if true print("test")`,
			opts:    &syntax.FileOptions{},
			wantErr: "compilation error:",
		},
		{
			name:    "predeclared global",
			script:  `print(request)`,
			globals: []string{"print"},
			wantErr: "undefined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			scriptBytes := []byte(tt.script)

			if tt.globals != nil {
				// Create predeclared globals
				predeclared := make(starlarkLib.StringDict, len(tt.globals))
				for _, name := range tt.globals {
					predeclared[name] = starlarkLib.None
				}
				_, err = compile(scriptBytes, tt.opts, predeclared)
			} else {
				// Create empty globals dict for non-global tests
				emptyGlobals := make(starlarkLib.StringDict)
				_, err = compile(scriptBytes, tt.opts, emptyGlobals)
			}
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCompileNil(t *testing.T) {
	tests := []struct {
		name    string
		globals []string
		wantErr string
	}{
		{
			name:    "nil script",
			wantErr: "script content is nil",
		},
		{
			name:    "nil script with globals",
			globals: []string{"request"},
			wantErr: "script content is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.globals != nil {
				// Create predeclared globals
				predeclared := make(starlarkLib.StringDict, len(tt.globals))
				for _, name := range tt.globals {
					predeclared[name] = starlarkLib.None
				}
				_, err = compile(nil, &syntax.FileOptions{}, predeclared)
			} else {
				// Create empty globals dict for non-global tests
				emptyGlobals := make(starlarkLib.StringDict)
				_, err = compile(nil, &syntax.FileOptions{}, emptyGlobals)
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
