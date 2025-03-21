# Risor Scripting Examples

This directory contains examples of using the go-polyscript library with the [Risor](https://github.com/risor-io/risor) scripting engine.

## Available Examples

### Simple Execution

The [Simple Example](./simple) demonstrates the basic "compile and run once" pattern:

- Script is compiled and immediately executed
- Data is provided at compile time through a `StaticProvider`
- Good for one-off script executions

To run:

```bash
cd examples/risor/simple
go run main.go
```

### Multiple Executions

The [Multiple Example](./multiple) demonstrates the efficient "compile once, run many times" pattern:

- Script is compiled only once
- The same compiled script is executed multiple times with different data
- Data is provided at runtime through a `ContextProvider`
- Significantly improves performance for multiple executions

To run:

```bash
cd examples/risor/multiple
go run main.go
```

## Risor Scripting Language

Risor is a modern embedded scripting language for Go with a focus on simplicity and performance. It offers:

- Clean syntax inspired by Go and Python
- Native access to Go data types
- Strong typing with type inference
- Easy integration with Go applications

For more information about Risor, visit [https://github.com/risor-io/risor](https://github.com/risor-io/risor).