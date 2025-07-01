.PHONY: all test build clean

all: build test

# Build the WASM module
build: clean
	cd examples && tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o main.wasm main.go
	cp examples/main.wasm main.wasm

# Run the Go tests
test: build
	go test .

# Clean up artifacts
clean:
	rm -f examples/*.wasm main.wasm
