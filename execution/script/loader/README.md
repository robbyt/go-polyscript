# Script Loaders

This package defines the `Loader` interface, which provides a common abstraction for accessing script content from various sources.

## Overview

The `Loader` interface standardizes how script content is retrieved and identified within the go-polyscript framework. It serves as a building block in the execution pipeline where script content is loaded, compiled, and evaluated.

## Architecture

Loaders fit into the execution flow as follows:

1. A `Loader` provides script content to the compiler
2. The compiler processes this content into executable form
3. The `ExecutableUnit` maintains a reference to the original loader
4. During evaluation, the source information may be used for error reporting

## Implementation

When creating a new loader, follow the error patterns defined in `errors.go` and refer to existing implementations for consistency.

See the source files in this directory for specific loader implementations.