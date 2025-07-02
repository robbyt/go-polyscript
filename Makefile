.PHONY: all
all: help

## help: Display this help message
.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'
	@echo

## test: Run tests with race detection and coverage
.PHONY: test
test: go-generate engines/extism/wasmdata/main.wasm
	go test -race -cover ./...

## bench: Run performance benchmarks and create reports
.PHONY: bench
bench: go-generate
	engines/benchmarks/run.sh

## bench-quick: Run benchmarks without creating reports
.PHONY: bench-quick
bench-quick: go-generate
	go test -run=^$$ -bench=. -benchmem ./...

## lint: Run golangci-lint code quality checks
.PHONY: lint
lint: go-generate
	golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix for common issues
.PHONY: lint-fix
lint-fix: go-generate
	golangci-lint fmt
	golangci-lint run --fix ./...

# Build WASM module when needed
engines/extism/wasmdata/main.wasm: engines/extism/wasmdata/examples/main.go engines/extism/wasmdata/examples/go.mod engines/extism/wasmdata/examples/go.sum
	$(MAKE) -C engines/extism/wasmdata main.wasm

## go-generate: Run code generation for type wrappers
.PHONY: go-generate
go-generate:
	cd engines/types && go generate

## wasmdata-build: Build WASM test data
.PHONY: wasmdata-build
wasmdata-build: engines/extism/wasmdata/main.wasm

## clean: Clean up build artifacts
.PHONY: clean
clean:
	$(MAKE) -C engines/extism/wasmdata clean
