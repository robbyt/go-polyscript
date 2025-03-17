# go-polyscript Examples

This directory contains examples of using the go-polyscript library with different script engines.

## Available Examples

### Risor

[Risor](https://github.com/risor-io/risor) is a scripting language embedded in Go that provides a simple, fast, and safe way to execute scripts.

To run the Risor example:

```bash
cd examples
go run risor/main.go
```

### Starlark

[Starlark](https://github.com/google/starlark-go) is a dialect of Python that is designed to be simple, small, and hermetic, and suitable for use as a configuration language.

To run the Starlark example:

```bash
cd examples
go run starlark/main.go
```

### Extism (WebAssembly)

[Extism](https://extism.org/) is a framework for executing WebAssembly modules that provides a simple way to call WebAssembly functions from Go.

To run the Extism example:

```bash
cd examples
go run extism/main.go
```

Note: The Extism example requires a WebAssembly module. A sample module is provided in the examples/extism directory.

## Example Structure

Each example demonstrates:

1. Creating a logger for the script engine
2. Configuring and creating a compiler for the specific script engine
3. Loading script content (from string or file)
4. Creating an executable unit
5. Creating an evaluator
6. Setting up a context with input data
7. Evaluating the script and handling the result

These examples show the common pattern for using any supported script engine with go-polyscript.