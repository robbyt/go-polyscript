package polyscript_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadmeQuickStart(t *testing.T) {
	t.Parallel()

	script := `
		// The ctx object holds the input data map
		let name = ctx.get("name")

		let p = "."
		if (ctx.get("excited")) {
			p = "!"
		}

		let message = "Hello, " + name + p

		// Return a map with our result
		{
			"greeting": message,
			"length": len(message)
		}
	`

	evaluator, err := polyscript.New[polyscript.Risor](
		t.Context(),
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
	)
	require.NoError(t, err, "Should create evaluator successfully")

	result, err := evaluator.Eval(t.Context())
	require.NoError(t, err, "Should evaluate successfully")
	require.NotNil(t, result, "Result should not be nil")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Equal(t, "Hello, World.", resultMap["greeting"], "Greeting should match")
	assert.Equal(t, int64(13), resultMap["length"], "Length should be 13")
}

func TestReadmeStaticProvider(t *testing.T) {
	t.Parallel()

	script := `
		let name = ctx.get("name")
		let excited = ctx.get("excited")

		let p = "."
		if (excited) {
			p = "!"
		}

		let message = "Hello, " + name + p

		{
			"greeting": message
		}
	`

	evaluator, err := polyscript.New[polyscript.Risor](
		t.Context(),
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "cats", "excited": true}),
	)
	require.NoError(t, err, "Should create evaluator successfully")

	result, err := evaluator.Eval(t.Context())
	require.NoError(t, err, "Should evaluate successfully")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Equal(t, "Hello, cats!", resultMap["greeting"], "Greeting should match with excitement")
}

func TestReadmeContextProvider(t *testing.T) {
	t.Parallel()

	script := `
		let name = ctx.get("name")
		let relationship = ctx.get("relationship")

		{
			"name": name,
			"is_not_my_lover": relationship == false
		}
	`

	evaluator, err := polyscript.New[polyscript.Risor](t.Context(), polyscript.FromString(script))
	require.NoError(t, err, "Should create evaluator successfully")

	runtimeData := map[string]any{"name": "Billie Jean", "relationship": false}
	enrichedCtx, err := evaluator.AddDataToContext(t.Context(), runtimeData)
	require.NoError(t, err, "Should add data to context successfully")

	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err, "Should evaluate successfully")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Equal(t, "Billie Jean", resultMap["name"], "Name should match")
	assert.Equal(t, true, resultMap["is_not_my_lover"], "Relationship status should be correct")
}

func TestReadmeCombiningStaticAndDynamic(t *testing.T) {
	t.Parallel()

	script := `
		// Access both static and dynamic data
		let name = ctx.get("name")
		let excited = ctx.get("excited")

		let p = "."
		if (excited) {
			p = "!"
		}

		let message = "Hello, " + name + p

		{
			"greeting": message
		}
	`

	staticData := map[string]any{
		"name":    "User",
		"excited": true,
	}

	evaluator, err := polyscript.New[polyscript.Risor](
		t.Context(),
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](staticData),
	)
	require.NoError(t, err, "Should create evaluator with static data")

	requestData := map[string]any{"name": "Robert"}
	enrichedCtx, err := evaluator.AddDataToContext(t.Context(), requestData)
	require.NoError(t, err, "Should add runtime data to context")

	result, err := evaluator.Eval(enrichedCtx)
	require.NoError(t, err, "Should evaluate with combined data")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Equal(t, "Hello, Robert!", resultMap["greeting"], "Should use runtime name over static")
}

func TestReadmeStarlark(t *testing.T) {
	t.Parallel()

	scriptContent := `
# Starlark has access to ctx variable
name = ctx["name"]
message = "Hello, " + name + "!"

# Create the result dictionary
result = {"greeting": message, "length": len(message)}

# Assign to _ to return the value
_ = result
`

	evaluator, err := polyscript.New[polyscript.Starlark](
		t.Context(),
		polyscript.FromString(scriptContent),
		polyscript.WithStaticData[polyscript.Starlark](map[string]any{"name": "World"}),
	)
	require.NoError(t, err, "Should create Starlark evaluator")

	result, err := evaluator.Eval(t.Context())
	require.NoError(t, err, "Should evaluate Starlark script")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Equal(t, "Hello, World!", resultMap["greeting"], "Greeting should match")
	assert.Equal(t, int64(13), resultMap["length"], "Length should be 13")
}

func TestReadmeExtism(t *testing.T) {
	t.Parallel()

	evaluator, err := polyscript.New[polyscript.Extism](
		t.Context(),
		polyscript.FromBytes(wasmdata.TestModule),
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)
	require.NoError(t, err, "Should create Extism evaluator")

	result, err := evaluator.Eval(t.Context())
	require.NoError(t, err, "Should evaluate WASM module")
	require.NotNil(t, result, "Result should not be nil")

	resultMap, err := result.AsMap()
	require.NoError(t, err, "Result should be a map")
	assert.Contains(t, resultMap, "greeting", "Result should contain greeting field")
	assert.Equal(t, "Hello, World!", resultMap["greeting"], "Greeting should match")
}

// constructorCallPattern matches a call to polyscript.New[E] or one of the
// engine FromXxxLoader constructors, capturing the text of the first argument
// up to the first comma or closing paren.
var constructorCallPattern = regexp.MustCompile(
	`(polyscript\.New\[[^\]]+\]|\bFrom(?:Risor|Starlark|Extism)Loader)\(\s*([^,)]*)`,
)

// contextArgPattern matches the first-argument forms accepted as a context in
// documentation snippets.
var contextArgPattern = regexp.MustCompile(`^(ctx|context\.\w+\(\)|t\.Context\(\))$`)

// markdownGoBlocks returns the contents of every fenced ```go block in a
// Markdown file.
func markdownGoBlocks(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "Should read %s", path)

	var blocks, current []string
	inBlock := false
	for line := range strings.SplitSeq(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case !inBlock && trimmed == "```go":
			inBlock = true
			current = nil
		case inBlock && trimmed == "```":
			inBlock = false
			blocks = append(blocks, strings.Join(current, "\n"))
		case inBlock:
			current = append(current, line)
		}
	}
	require.False(t, inBlock, "%s has an unterminated ```go block", path)
	return blocks
}

// godocCodeBlocks returns the tab-indented code lines of a Go file's doc
// comments, joined into a single snippet.
func godocCodeBlocks(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "Should read %s", path)

	var lines []string
	for line := range strings.SplitSeq(string(raw), "\n") {
		if code, ok := strings.CutPrefix(line, "//\t"); ok {
			lines = append(lines, code)
		}
	}
	return []string{strings.Join(lines, "\n")}
}

// TestDocExamplesPassContext guards against documentation drift: every
// documented constructor call must pass a context as its first argument,
// matching the real signatures of polyscript.New and the engine
// FromXxxLoader constructors. The other README tests exercise the snippets'
// logic but not the exact source shown, so they cannot catch this.
func TestDocExamplesPassContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		extract func(*testing.T, string) []string
	}{
		{"README.md", markdownGoBlocks},
		{"polyscript.go", godocCodeBlocks},
		{"engines/risor/new.go", godocCodeBlocks},
		{"engines/starlark/new.go", godocCodeBlocks},
		{"engines/extism/new.go", godocCodeBlocks},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			calls := 0
			for _, snippet := range tc.extract(t, tc.path) {
				for _, m := range constructorCallPattern.FindAllStringSubmatch(snippet, -1) {
					calls++
					assert.Regexp(t, contextArgPattern, m[2],
						"%s: %s(...) must take a context as its first argument", tc.path, m[1])
				}
			}
			assert.Positive(t, calls, "%s should contain at least one constructor example", tc.path)
		})
	}
}
