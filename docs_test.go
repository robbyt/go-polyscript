package polyscript_test

// Every Go code block in README.md, and every code block in the doc comments
// of the files listed in godocSnippets, is a copy of Go code under
// examples/docs. go test compiles that code and runs it through the Example
// functions beside it, which check its output. TestDocSnippets fails when a
// copy differs from its source, so the docs cannot drift away from code that
// is known to compile and run.
//
// To change a snippet, edit its source under examples/docs, run
// go test ./examples/docs/..., then copy the code into the doc. In README.md
// each Go block names its source in an HTML comment on the line above the
// fence: "<!-- snippet: PATH -->" for a whole file (a complete program), or
// "<!-- snippet: PATH#NAME -->" for the "// snippet:begin NAME" regions of a
// file (a fragment). Godoc code blocks are matched, in order, with the
// regions listed in godocSnippets. See examples/docs/snippets/doc.go for how
// regions are written.
//
// The comparison is exact apart from this normalization: trailing whitespace
// is removed from each line, leading and trailing blank lines are removed,
// and each line of a region loses the indentation of the region's begin
// marker.

import (
	"go/doc/comment"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snippetsDir holds the source of every documented snippet.
const snippetsDir = "examples/docs"

// godocSnippets lists, for each Go file with code in its doc comments, the
// source of each code block in the order the blocks appear in the file.
var godocSnippets = []struct {
	file    string
	sources []string
}{
	{"polyscript.go", []string{
		"examples/docs/snippets/godoc_polyscript.go#new",
		"examples/docs/snippets/godoc_polyscript.go#static-data",
		"examples/docs/snippets/godoc_polyscript.go#extism",
		"examples/docs/snippets/godoc_polyscript.go#type-argument",
		"examples/docs/snippets/godoc_polyscript.go#constructor-risor",
		"examples/docs/snippets/godoc_polyscript.go#constructor-extism",
	}},
	{"engines/risor/new.go", []string{
		"examples/docs/snippets/godoc_engines.go#risor-quick-start",
		"examples/docs/snippets/godoc_engines.go#risor-options",
	}},
	{"engines/starlark/new.go", []string{
		"examples/docs/snippets/godoc_engines.go#starlark-quick-start",
		"examples/docs/snippets/godoc_engines.go#starlark-options",
	}},
	{"engines/extism/new.go", []string{
		"examples/docs/snippets/godoc_engines.go#extism-quick-start",
		"examples/docs/snippets/godoc_engines.go#extism-options",
	}},
}

var (
	readmeMarker = regexp.MustCompile(`^<!-- snippet: (\S+) -->$`)
	regionBegin  = regexp.MustCompile(`^(\s*)// snippet:begin (\S+)$`)
	regionEnd    = regexp.MustCompile(`^\s*// snippet:end$`)
	// fenceOpen matches a Markdown fence: a run of three or more backticks
	// or tildes, then the info string.
	fenceOpen = regexp.MustCompile("^(`{3,}|~{3,})(.*)$")
)

// docBlock is a Go code block found in a doc, with the snippet source it
// should match.
type docBlock struct {
	where  string // file and line, or file and block number, for messages
	source string // PATH or PATH#NAME under snippetsDir
	code   string
}

// normalize removes trailing whitespace from each line and leading and
// trailing blank lines from s.
func normalize(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// readmeBlocks returns every Go code block in a Markdown file, each with the
// source named by the snippet marker on the line above its fence. Fences
// follow CommonMark, as GitHub renders them: a run of at least three
// backticks or tildes opens a block, the first word of the info string after
// it names the language (go or golang, in any case), and only a run of the
// same character at least as long closes it.
func readmeBlocks(t *testing.T, path string) []docBlock {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "Should read %s", path)

	var blocks []docBlock
	var current *docBlock
	var body []string
	fence := "" // the opening fence run while inside a block
	previous := ""
	for i, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case fence == "":
			m := fenceOpen.FindStringSubmatch(trimmed)
			if m == nil || (m[1][0] == '`' && strings.Contains(m[2], "`")) {
				break
			}
			fence = m[1]
			if isGoInfo(m[2]) {
				current = &docBlock{where: path + ":" + strconv.Itoa(i+1)}
				if marker := readmeMarker.FindStringSubmatch(previous); marker != nil {
					current.source = marker[1]
				}
				body = nil
			}
		case len(trimmed) >= len(fence) && strings.Trim(trimmed, fence[:1]) == "":
			fence = ""
			if current != nil {
				current.code = strings.Join(body, "\n")
				blocks = append(blocks, *current)
				current = nil
			}
		default:
			body = append(body, line)
		}
		if trimmed != "" {
			previous = trimmed
		}
	}
	require.Empty(t, fence, "%s has an unterminated code fence", path)
	return blocks
}

// isGoInfo reports whether a fence info string marks a Go code block.
func isGoInfo(info string) bool {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return false
	}
	lang := strings.ToLower(fields[0])
	return lang == "go" || lang == "golang"
}

// godocBlocks returns the code blocks in the comments of a Go file, as godoc
// renders them, each paired with the next source from sources.
func godocBlocks(t *testing.T, path string, sources []string) []docBlock {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	require.NoError(t, err, "Should parse %s", path)

	var codes []string
	var p comment.Parser
	for _, group := range file.Comments {
		for _, block := range p.Parse(group.Text()).Content {
			if code, ok := block.(*comment.Code); ok {
				codes = append(codes, code.Text)
			}
		}
	}
	// Compare counts rather than using require.Len, which would print every
	// block on failure.
	found, listed := len(codes), len(sources)
	require.Equal(t, listed, found,
		"%s has %d doc code blocks but godocSnippets lists %d sources for it", path, found, listed)

	blocks := make([]docBlock, len(codes))
	for i, code := range codes {
		blocks[i] = docBlock{where: path + " code block " + strconv.Itoa(i+1), source: sources[i], code: code}
	}
	return blocks
}

// loadSnippet returns the code a doc block should match: the whole file for
// PATH, or the regions named NAME for PATH#NAME, joined by blank lines.
func loadSnippet(t *testing.T, source string) string {
	t.Helper()
	path, name, hasName := strings.Cut(source, "#")
	require.True(t, strings.HasPrefix(path, snippetsDir+"/"),
		"Snippet source %q should be under %s", source, snippetsDir)
	raw, err := os.ReadFile(filepath.FromSlash(path))
	require.NoError(t, err, "Should read snippet source %s", path)

	regions := snippetRegions(t, path, string(raw))
	if !hasName {
		require.Empty(t, regions, "%s has snippet regions; name one as %s#NAME", path, path)
		return normalize(string(raw))
	}
	require.NotEmpty(t, regions[name], "%s has no \"// snippet:begin %s\" region", path, name)
	return strings.Join(regions[name], "\n\n")
}

// snippetRegions returns the normalized regions of a snippet source file,
// grouped by name in file order.
func snippetRegions(t *testing.T, path, src string) map[string][]string {
	t.Helper()
	regions := map[string][]string{}
	name, indent := "", ""
	var body []string
	for i, line := range strings.Split(src, "\n") {
		where := path + ":" + strconv.Itoa(i+1)
		if m := regionBegin.FindStringSubmatch(line); m != nil {
			require.Empty(t, name, "%s: snippet region %q starts inside region %q", where, m[2], name)
			name, indent, body = m[2], m[1], nil
			continue
		}
		if regionEnd.MatchString(line) {
			require.NotEmpty(t, name, "%s: snippet:end without a snippet:begin", where)
			regions[name] = append(regions[name], normalize(strings.Join(body, "\n")))
			name = ""
			continue
		}
		if name == "" {
			continue
		}
		// Lines indented less than the marker, such as the lines of a raw
		// string literal, are kept as they are.
		body = append(body, strings.TrimPrefix(line, indent))
	}
	require.Empty(t, name, "%s: snippet region %q has no snippet:end", path, name)
	return regions
}

// snippetSources returns every source under snippetsDir that should be
// shown in a doc: each region of a file with regions, and each main.go.
func snippetSources(t *testing.T) []string {
	t.Helper()
	var sources []string
	err := filepath.WalkDir(snippetsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		path = filepath.ToSlash(path)
		regions := snippetRegions(t, path, string(raw))
		for name := range regions {
			sources = append(sources, path+"#"+name)
		}
		if len(regions) == 0 && filepath.Base(path) == "main.go" {
			sources = append(sources, path)
		}
		return nil
	})
	require.NoError(t, err, "Should walk %s", snippetsDir)
	return sources
}

// TestDocSnippets checks that every Go code block in README.md and in the
// godoc of godocSnippets matches its source under examples/docs, and that
// every source there is shown somewhere.
func TestDocSnippets(t *testing.T) {
	t.Parallel()

	blocks := readmeBlocks(t, "README.md")
	require.NotEmpty(t, blocks, "README.md should contain Go code blocks")
	for _, doc := range godocSnippets {
		blocks = append(blocks, godocBlocks(t, doc.file, doc.sources)...)
	}

	shown := map[string]bool{}
	for _, block := range blocks {
		t.Run(block.where, func(t *testing.T) {
			require.NotEmpty(t, block.source,
				"%s: add a <!-- snippet: PATH#NAME --> line above this Go code block naming its source under %s",
				block.where, snippetsDir)
			shown[block.source] = true
			assert.Equal(t, loadSnippet(t, block.source), normalize(block.code),
				"%s should match %s; update whichever one is out of date", block.where, block.source)
		})
	}

	t.Run("every snippet is shown", func(t *testing.T) {
		for _, source := range snippetSources(t) {
			assert.True(t, shown[source], "%s is not shown in any doc; remove it or add it to a doc", source)
		}
	})
}
