# Variables
PACKAGES := $(shell go list ./...)

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
test: go-generate
	go test -race -cover $(PACKAGES)

## bench: Run performance benchmarks and create reports
.PHONY: bench
bench: go-generate
	benchmarks/run.sh

## bench-quick: Run benchmarks without creating reports
.PHONY: bench-quick
bench-quick: go-generate
	go test -run=^$$ -bench=. -benchmem $(PACKAGES)

## lint: Run golangci-lint code quality checks
.PHONY: lint
lint: go-generate
	golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix for common issues
.PHONY: lint-fix
lint-fix: go-generate
	golangci-lint fmt
	golangci-lint run --fix ./...

## go-generate: Run code generation for type wrappers
.PHONY: go-generate
go-generate:
	cd machines/types && go generate
