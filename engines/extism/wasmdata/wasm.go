package wasmdata

import _ "embed"

// TestModule contains the compiled WASM bytecode for testing.
// This module is compiled from examples/main.go and provides several
// exported functions for testing purposes.
//
//go:embed main.wasm
var TestModule []byte

// Entrypoint constants for the embedded WASM module.
// These correspond to the exported functions from the WASM module.
const (
	// EntrypointGreet processes JSON input with "input" field and returns a greeting.
	// Input: {"input": "world"} -> Output: {"greeting": "Hello, world!"}
	EntrypointGreet = "greet"

	// EntrypointRun is an alias for the greet function.
	EntrypointRun = "run"

	// EntrypointProcessComplex processes complex Request objects and returns Response objects.
	// Handles structured data with ID, timestamp, data maps, tags, metadata, count, and active status.
	EntrypointProcessComplex = "process_complex"

	// EntrypointCountVowels counts vowels in the input string.
	// Input: {"input": "hello"} -> Output: {"count": 2, "vowels": "aeiouAEIOU", "input": "hello"}
	EntrypointCountVowels = "count_vowels"

	// EntrypointReverseString reverses the input string, handling UTF-8 correctly.
	// Input: {"input": "hello"} -> Output: {"reversed": "olleh"}
	EntrypointReverseString = "reverse_string"
)
