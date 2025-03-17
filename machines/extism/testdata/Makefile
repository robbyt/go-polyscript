.PHONY: all test build clean

all: build test

# Build the WASM module
build: clean
	tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o examples/main.wasm examples/main.go

# Run the Go tests
test: build
	go test .

# Clean up artifacts
clean:
	rm -f examples/*.wasm
