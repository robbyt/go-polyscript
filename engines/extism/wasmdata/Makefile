.PHONY: all help main.wasm test clean clean-all

all: help

## help: Display this help message
.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'
	@echo

## main.wasm: Build the WASM module from Go source
main.wasm: examples/main.go examples/go.mod examples/go.sum
	cd examples && tinygo build -scheduler=none -target=wasip1 -buildmode=c-shared -o main.wasm main.go
	cp examples/main.wasm main.wasm

# Copy committed WASM to examples for any processes that need it there
examples/main.wasm: main.wasm
	cp main.wasm examples/main.wasm

## test: Run Go tests (builds WASM if needed)
.PHONY: test
test: main.wasm
	go test .

## clean: Clean up build artifacts (keep committed main.wasm)
.PHONY: clean
clean:
	rm -f examples/*.wasm

## clean-all: Clean up all artifacts including committed main.wasm
.PHONY: clean-all
clean-all:
	rm -f examples/*.wasm main.wasm
