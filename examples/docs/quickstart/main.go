package main

import (
	"context"
	"fmt"
	"log"

	"github.com/robbyt/go-polyscript"
)

func main() {
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

	ctx := context.Background()

	evaluator, err := polyscript.New[polyscript.Risor](
		ctx,
		polyscript.FromString(script),
		polyscript.WithStaticData[polyscript.Risor](map[string]any{"name": "World"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	result, err := evaluator.Eval(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result.Interface())
}
