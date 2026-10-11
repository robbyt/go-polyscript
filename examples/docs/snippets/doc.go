// Package snippets holds the source of the Go code fragments shown in
// README.md and in the doc comments of the module's Go files, such as
// polyscript.go, engines/{risor,starlark,extism}/new.go, and the platform
// packages (godoc_platform.go). Complete programs shown in
// README.md live beside this package, in examples/docs/quickstart and
// examples/docs/extism.
//
// Each fragment sits between a "// snippet:begin NAME" line and a
// "// snippet:end" line, inside a function whose parameters are the
// variables the fragment uses but does not declare. A file can hold several
// regions with the same NAME (for example an import block and a function
// body); the documented fragment is those regions joined by a blank line.
// The code after a region evaluates the result and prints it, and the
// Example functions in snippets_test.go check that output.
//
// docs_test.go in the repository root fails when a documented fragment no
// longer matches its region here. To change a fragment, edit it here, run
// go test ./examples/docs/... to see that it still works, then copy the
// region (minus its indentation) into the doc. The README marks the source
// of each Go block with an HTML comment like
// <!-- snippet: examples/docs/snippets/providers.go#static-provider -->.
package snippets
