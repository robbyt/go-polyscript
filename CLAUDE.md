# go-polyscript Commands and Guidelines

## Build & Test Commands
- Run all tests: `make test`
- Run single test: `go test -v -run=TestName ./path/to/package`
- Run tests with coverage: `go test -race -cover ./...`
- Run benchmarks: `make bench`
- Lint code: `make lint`
- Auto-fix linting issues: `make lint-fix`
- Generate code: `make go-generate`

## Code Style Guidelines
- Use `gofmt` for consistent formatting (handled by lint-fix)
- Import order: standard library, then third-party, then local packages
- Error handling: check and return errors, don't use panic
- Use meaningful error messages with context
- Variable naming: camelCase for variables, PascalCase for exported functions
- Tests: table-driven tests with descriptive names, use testify/require
- Interface naming: add 'er' suffix for behavior interfaces
- Documentation: all exported functions must have comments
- Minimize dependencies, prefer standard library when possible
- Use context.Context for cancellation and timeouts