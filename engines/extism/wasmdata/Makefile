.PHONY: all test clean

all: main.wasm test

# Build the WASM module with proper dependency tracking
main.wasm: examples/main.go examples/go.mod examples/go.sum
	cd examples && tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o main.wasm main.go
	cp examples/main.wasm main.wasm

# Run the Go tests
test: main.wasm
	go test .

# Clean up artifacts
clean:
	rm -f examples/*.wasm main.wasm
