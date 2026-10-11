package polyscript_test

// Every Go code block in README.md, and every code block in the doc comments
// of the module's Go files, is a copy of Go code under examples/docs. go test compiles that code and runs it through the Example
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
// regions listed for their file in godocSnippets. Every Go file in the
// module outside examples/, testdata, vendor, dot-directories, nested
// modules, and _test.go files is searched, and godoc treats any indented
// text in a doc comment as a code block, so text there that is not Go must be
// written as prose instead. Comments inside function bodies are not doc
// comments and are not checked. See examples/docs/snippets/doc.go for how regions are
// written.
//
// The comparison is exact apart from this normalization: trailing whitespace
// is removed from each line, leading and trailing blank lines are removed,
// and each line of a region loses the indentation of the region's begin
// marker.

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
// TestDocSnippets finds those files itself (see godocFiles) and fails when
// one is missing here, or when an entry names a file with no code blocks.
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
	{"platform/data/compositeProvider.go", []string{
		"examples/docs/snippets/godoc_platform.go#composite-add-data",
	}},
	{"platform/data/provider.go", []string{
		"examples/docs/snippets/godoc_platform.go#setter-add-data",
	}},
	{"platform/script/loader/fromHTTP.go", []string{
		"examples/docs/snippets/godoc_platform.go#http-options",
		"examples/docs/snippets/godoc_platform.go#new-from-http",
		"examples/docs/snippets/godoc_platform.go#new-from-http-with-options",
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

// errUnterminatedFence is returned by markdownBlocks for a code fence that
// is never closed.
var errUnterminatedFence = errors.New("unterminated code fence")

// readmeBlocks returns every Go code block in a Markdown file; see
// markdownBlocks.
func readmeBlocks(t *testing.T, path string) []docBlock {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "Should read %s", path)
	blocks, err := markdownBlocks(path, string(raw))
	require.NoError(t, err, "Should parse the code fences in %s", path)
	return blocks
}

// markdownBlocks returns every Go code block in text, the Markdown content
// of path, each with the source named by the snippet marker on the last
// non-blank line above its fence. Fences follow CommonMark, as GitHub renders
// them: a run of at least three backticks or tildes opens a block, the first
// word of the info string after it names the language (go or golang, in any
// case), and only a run of the same character at least as long closes it.
func markdownBlocks(path, text string) ([]docBlock, error) {
	r := &fenceReader{path: path}
	for i, line := range strings.Split(text, "\n") {
		r.readLine(i+1, line)
	}
	if r.fence != "" {
		return nil, fmt.Errorf("%s: %w", path, errUnterminatedFence)
	}
	return r.blocks, nil
}

// fenceReader collects the Go code blocks of a Markdown file one line at a
// time.
type fenceReader struct {
	path     string
	blocks   []docBlock
	fence    string    // the opening fence run while inside a block
	current  *docBlock // the Go block being read; nil inside other blocks
	body     []string  // the lines of current read so far
	previous string    // the last non-blank line, for the snippet marker
}

// readLine handles line n (counting from 1) of the file.
func (r *fenceReader) readLine(n int, line string) {
	trimmed := strings.TrimSpace(line)
	switch {
	case r.fence == "":
		r.open(n, trimmed)
	case closesFence(trimmed, r.fence):
		r.close()
	case r.current != nil:
		r.body = append(r.body, line)
	}
	if trimmed != "" {
		r.previous = trimmed
	}
}

// open starts a block when line n opens a fence, and starts collecting it
// when the block is Go.
func (r *fenceReader) open(n int, line string) {
	fence, info, ok := openFence(line)
	if !ok {
		return
	}
	r.fence = fence
	if !isGoInfo(info) {
		return
	}
	r.current = &docBlock{where: r.path + ":" + strconv.Itoa(n)}
	if marker := readmeMarker.FindStringSubmatch(r.previous); marker != nil {
		r.current.source = marker[1]
	}
	r.body = nil
}

// close ends the current block, keeping it when it is Go.
func (r *fenceReader) close() {
	r.fence = ""
	if r.current == nil {
		return
	}
	r.current.code = strings.Join(r.body, "\n")
	r.blocks = append(r.blocks, *r.current)
	r.current = nil
}

// openFence returns the fence run and info string when line, with its
// surrounding space trimmed, opens a fenced code block. A backtick fence
// whose info string holds a backtick is inline code, not a fence.
func openFence(line string) (fence, info string, ok bool) {
	m := fenceOpen.FindStringSubmatch(line)
	if m == nil || (m[1][0] == '`' && strings.Contains(m[2], "`")) {
		return "", "", false
	}
	return m[1], m[2], true
}

// closesFence reports whether line, with its surrounding space trimmed,
// closes a block opened by fence: a run of the same character, at least as
// long, with nothing after it.
func closesFence(line, fence string) bool {
	return len(line) >= len(fence) && strings.Trim(line, fence[:1]) == ""
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

// commentCodes returns the text of every code block in the doc comments of a
// Go file, as godoc renders them, in file order. Any indented text in a doc
// comment is a code block to godoc, whether or not it is Go. Comments inside
// function bodies are not doc comments and are ignored.
func commentCodes(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var codes []string
	var p comment.Parser
	for _, group := range docComments(file) {
		for _, block := range p.Parse(group.Text()).Content {
			if code, ok := block.(*comment.Code); ok {
				codes = append(codes, code.Text)
			}
		}
	}
	return codes, nil
}

// docComments returns the doc comments of a parsed Go file in file order: the
// package doc, the docs of top-level declarations and their specs, and the
// docs of struct fields and interface methods. It does not look inside
// function bodies, where no comment is rendered by godoc.
func docComments(file *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	add := func(group *ast.CommentGroup) {
		if group != nil {
			groups = append(groups, group)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.BlockStmt:
			return false
		case *ast.File:
			add(n.Doc)
		case *ast.GenDecl:
			add(n.Doc)
		case *ast.FuncDecl:
			add(n.Doc)
		case *ast.TypeSpec:
			add(n.Doc)
		case *ast.ValueSpec:
			add(n.Doc)
		case *ast.Field:
			add(n.Doc)
		}
		return true
	})
	return groups
}

// godocFiles returns the path, relative to root and with forward slashes, of
// every Go file in the module at root whose doc comments hold a code block. It
// skips _test.go files and these directories: examples at the root (the
// snippet sources and example programs), testdata, vendor, directories whose
// names start with a dot, and nested modules (any directory below root with
// its own go.mod). A file that does not parse is an error.
func godocFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			return skipGodocDir(path, rel, d.Name())
		}
		if filepath.Ext(rel) != ".go" || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		codes, err := commentCodes(path)
		if err != nil {
			return err
		}
		if len(codes) > 0 {
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// skipGodocDir returns fs.SkipDir for a directory godocFiles does not search,
// given its path, its path relative to the root, and its name.
func skipGodocDir(path, rel, name string) error {
	if rel == "." {
		return nil
	}
	if rel == "examples" || name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") {
		return fs.SkipDir
	}
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return fs.SkipDir
	}
	return nil
}

// godocListProblems compares the files found to have doc code blocks with the
// files listed in godocSnippets, and describes each file that is found but
// not listed, listed but not found, or listed more than once.
func godocListProblems(found, listed []string) []string {
	var problems []string
	count := map[string]int{}
	for _, file := range listed {
		count[file]++
		if count[file] == 2 {
			problems = append(problems, file+" is listed in godocSnippets more than once")
		}
	}
	isFound := map[string]bool{}
	for _, file := range found {
		isFound[file] = true
		if count[file] == 0 {
			problems = append(problems, file+" has doc code blocks but no godocSnippets entry")
		}
	}
	for file := range count {
		if !isFound[file] {
			problems = append(problems, file+" is listed in godocSnippets but has no doc code blocks")
		}
	}
	slices.Sort(problems)
	return problems
}

// godocBlocks returns the code blocks in the doc comments of a Go file, as godoc
// renders them, each paired with the next source from sources.
func godocBlocks(t *testing.T, path string, sources []string) []docBlock {
	t.Helper()
	codes, err := commentCodes(path)
	require.NoError(t, err, "Should parse %s", path)
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

	t.Run("every godoc code block is checked", func(t *testing.T) {
		found, err := godocFiles(".")
		require.NoError(t, err, "Should find the Go files with doc code blocks")
		listed := make([]string, len(godocSnippets))
		for i, doc := range godocSnippets {
			listed[i] = doc.file
		}
		for _, problem := range godocListProblems(found, listed) {
			assert.Fail(t, problem, "Map each doc code block to a region under %s in godocSnippets", snippetsDir)
		}
	})

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

// TestMarkdownBlocks checks which fenced code blocks markdownBlocks collects
// from Markdown text, and that it rejects an unterminated fence.
func TestMarkdownBlocks(t *testing.T) {
	t.Parallel()

	const marker = "<!-- snippet: a.go#x -->"

	t.Run("collects Go blocks", func(t *testing.T) {
		tests := []struct {
			name     string
			markdown string
			expected []docBlock
		}{
			{
				name:     "backtick fence with a marker",
				markdown: marker + "\n```go\nx := 1\n\ny := 2\n```\n",
				expected: []docBlock{{where: "doc.md:2", source: "a.go#x", code: "x := 1\n\ny := 2"}},
			},
			{
				name:     "tilde fence",
				markdown: marker + "\n~~~go\nx := 1\n~~~\n",
				expected: []docBlock{{where: "doc.md:2", source: "a.go#x", code: "x := 1"}},
			},
			{
				name:     "golang, any case, with more info",
				markdown: "```GoLang title=x\nx := 1\n```\n",
				expected: []docBlock{{where: "doc.md:1", code: "x := 1"}},
			},
			{
				name:     "indented fence and a marker above blank lines",
				markdown: marker + "\n\n  ```go\n  x := 1\n  ```\n",
				expected: []docBlock{{where: "doc.md:3", source: "a.go#x", code: "  x := 1"}},
			},
			{
				name:     "marker not on the last non-blank line",
				markdown: marker + "\ntext\n```go\nx := 1\n```\n",
				expected: []docBlock{{where: "doc.md:3", code: "x := 1"}},
			},
			{
				name:     "longer fence holds a shorter run",
				markdown: "````go\n```\nx := 1\n````\n",
				expected: []docBlock{{where: "doc.md:1", code: "```\nx := 1"}},
			},
			{
				name:     "longer run closes a fence",
				markdown: "```go\nx := 1\n`````\n",
				expected: []docBlock{{where: "doc.md:1", code: "x := 1"}},
			},
			{
				name:     "backticks do not close a tilde fence",
				markdown: "~~~go\n```\nx := 1\n~~~\n",
				expected: []docBlock{{where: "doc.md:1", code: "```\nx := 1"}},
			},
			{
				name:     "a run with text after it does not close a fence",
				markdown: "```go\n``` go\nx := 1\n```\n",
				expected: []docBlock{{where: "doc.md:1", code: "``` go\nx := 1"}},
			},
			{
				name:     "skips blocks that are not Go",
				markdown: "```sh\ngo test\n```\n```\nplain\n```\n~~~json\n{}\n~~~\n" + marker + "\n```go\nx := 1\n```\n",
				expected: []docBlock{{where: "doc.md:11", source: "a.go#x", code: "x := 1"}},
			},
			{
				name:     "a Go fence inside another block is code",
				markdown: "````md\n```go\nx := 1\n```\n````\n",
				expected: nil,
			},
			{
				name:     "inline code with a backtick is not a fence",
				markdown: "```go` is inline\n```go\nx := 1\n```\n",
				expected: []docBlock{{where: "doc.md:2", code: "x := 1"}},
			},
			{
				name:     "two runs are not a fence",
				markdown: "``go\nx := 1\n``\n",
				expected: nil,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				blocks, err := markdownBlocks("doc.md", tc.markdown)
				require.NoError(t, err, "Should parse the fences")
				assert.Equal(t, tc.expected, blocks, "Should collect the expected Go blocks")
			})
		}
	})

	t.Run("rejects an unterminated fence", func(t *testing.T) {
		tests := []struct {
			name     string
			markdown string
		}{
			{"Go block", "```go\nx := 1\n"},
			{"block that is not Go", "~~~sh\ngo test\n"},
			{"closed by a shorter run", "````go\nx := 1\n```\n"},
			{"closed by the other character", "```go\nx := 1\n~~~\n"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				blocks, err := markdownBlocks("doc.md", tc.markdown)
				require.ErrorIs(t, err, errUnterminatedFence, "Should reject the unterminated fence")
				assert.Contains(t, err.Error(), "doc.md", "The error should name the file")
				assert.Nil(t, blocks, "Should return no blocks")
			})
		}
	})
}

// writeFixture writes each file in files, a map of slash-separated path to
// content, under root.
func writeFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750), "Should create the directory for %s", name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600), "Should write %s", name)
	}
}

// TestGodocFiles checks which files godocFiles finds in a fixture module, and
// how godocListProblems reports them against a list.
func TestGodocFiles(t *testing.T) {
	t.Parallel()

	// withCode is a Go file whose package doc holds an indented block that
	// is not Go, which godoc still renders as code.
	const withCode = "// Package p does things.\n//\n//\tnot go, still a code block\npackage p\n"
	const withoutCode = "// Package p does things.\npackage p\n"
	// inBody has indented comment text only inside a function body, which
	// godoc never renders, so it has no doc code blocks.
	const inBody = "package p\n\nfunc f() int {\n\t// Layout:\n\t//\n\t//\t+---+---+\n\t//\t| a | b |\n" +
		"\n\t// local is documented.\n\t//\n\t//\tnot a doc code block\n\tvar local = 1\n\treturn local\n}\n"
	// inDecl and inField hold a code block in the doc of a function and of a
	// struct field.
	const inDecl = "package p\n\n// F does things.\n//\n//\tF()\nfunc F() {}\n"
	const inField = "package p\n\n// T is a type.\ntype T struct {\n\t// A is a field.\n\t//\n\t//\tt.A = 1\n\tA int\n}\n"

	t.Run("finds files with code blocks and skips excluded ones", func(t *testing.T) {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{
			"go.mod":                 "module example.com/m\n",
			"listed.go":              withCode,
			"unlisted.go":            withCode,
			"stale.go":               withoutCode,
			"body.go":                inBody,
			"decl.go":                inDecl,
			"field.go":               inField,
			"sub/deep.go":            withCode,
			"sub/examples/inner.go":  withCode,
			"listed_test.go":         withCode,
			"notes.txt":              "\tnot a Go file\n",
			"examples/ex.go":         withCode,
			"sub/testdata/td.go":     withCode,
			"vendor/v/v.go":          withCode,
			".hidden/h.go":           withCode,
			"nested/go.mod":          "module example.com/nested\n",
			"nested/n.go":            withCode,
			"nested/deeper/inner.go": withCode,
		})

		found, err := godocFiles(root)
		require.NoError(t, err, "Should walk the fixture module")
		assert.ElementsMatch(t,
			[]string{"listed.go", "unlisted.go", "decl.go", "field.go", "sub/deep.go", "sub/examples/inner.go"}, found,
			"Should find only files with doc code blocks outside the excluded directories")

		problems := godocListProblems(found, []string{
			"listed.go", "decl.go", "field.go", "sub/deep.go", "sub/examples/inner.go", "stale.go", "sub/deep.go",
		})
		assert.Equal(t, []string{
			"stale.go is listed in godocSnippets but has no doc code blocks",
			"sub/deep.go is listed in godocSnippets more than once",
			"unlisted.go has doc code blocks but no godocSnippets entry",
		}, problems, "Should report the unlisted file, the stale entry, and the duplicate entry")
	})

	t.Run("a complete list has no problems", func(t *testing.T) {
		assert.Empty(t, godocListProblems([]string{"a.go", "b/c.go"}, []string{"b/c.go", "a.go"}),
			"Should accept a list naming each found file once")
	})

	t.Run("a file that does not parse is an error", func(t *testing.T) {
		root := t.TempDir()
		writeFixture(t, root, map[string]string{
			"go.mod": "module example.com/m\n",
			"ok.go":  withCode,
			"bad.go": "package p\nfunc {\n",
		})

		_, err := godocFiles(root)
		require.Error(t, err, "Should fail on a file that does not parse")
		assert.Contains(t, err.Error(), "bad.go", "The error should name the file")
	})
}
