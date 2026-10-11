package main

import (
	"context"
	"fmt"
	"log"

	"github.com/robbyt/go-polyscript"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
)

func main() {
	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Extism](
		ctx,

		// pre-compiled WASM example module
		polyscript.FromBytes(wasmdata.TestModule),

		// main entrypoint function in the WASM module
		polyscript.WithEntryPoint(wasmdata.EntrypointGreet),

		// the go-polyscript Extism engine will encode the static data into
		// JSON and send it to the WASM application
		polyscript.WithStaticData[polyscript.Extism](map[string]any{"input": "World"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Execute, and print the result
	result, err := evaluator.Eval(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result.Interface())
}
