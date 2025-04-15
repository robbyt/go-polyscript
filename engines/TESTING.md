# Testing Guidelines for go-polyscript

This document outlines standardized testing principles for go-polyscript to ensure consistency, maintainability, and comprehensive test coverage.

## Core Testing Principles

1. **Consistency**: Use standardized patterns across all packages.
2. **Clarity**: Write clear, self-documenting tests with logical organization.
3. **Comprehensiveness**: Test both success paths and error conditions.
4. **Efficiency**: Avoid duplication through table-driven tests and helper functions.

## Test Quality Checklist

- Use **table-driven tests** for similar test cases.
- Apply `t.Parallel()` **only in parent tests**, not sub-tests.
- Capture **range variables** in loops to prevent race conditions.
- Separate **setup code** from **verification code**.
- Test both **success paths** and **error conditions**.
- Use `require.ErrorIs()` for error comparison instead of string matching.
- Provide **descriptive assertion messages** for debugging.
- Mark helper functions with `t.Helper()` for better error reporting.
- Verify all **mock expectations** after tests.
- Check **error returns** from all functions that return errors.
- Use **local test servers** instead of external dependencies.
- Organize imports consistently: **stdlib**, then external, then local.
- Group related tests under logical parent tests with `t.Run()`.

## Test Structure and Organization

### Naming Conventions

| Component          | Test Function Name          |
|---------------------|-----------------------------|
| Compiler creation   | `TestNewCompiler`          |
| Compilation         | `TestCompiler_Compile`     |
| Options             | `TestCompilerOptions`      |
| Executables         | `TestExecutable`           |
| Evaluator execution | `TestEvaluator_Evaluate`   |
| Context preparation | `TestEvaluator_AddDataToContext` |
| Response handling   | `TestResponseMethods`      |
| Type conversion     | `TestToGoType` / `TestToMachineType` |

### Sub-test Organization

- Use `t.Run()` to group related test cases:
  ```go
  t.Run("success cases", func(t *testing.T) {
      // Tests for normal operation
  })
  t.Run("error cases", func(t *testing.T) {
      // Tests for error handling
  })
  ```
- Use descriptive subtest names instead of comments.

### Parallelization

- Apply `t.Parallel()` only in parent test functions:
  ```go
  func TestResponseMethods(t *testing.T) {
      t.Parallel()
      for _, tc := range tests {
          t.Run(tc.name, func(t *testing.T) { ... })
      }
  }
  ```

## Best Practices

### Assertions

- Use `require` for critical conditions and `assert` for non-critical checks.
- Preferred assertion methods:
  - `assert.Equal(t, expected, actual)`
  - `require.NoError(t, err)`
  - `require.ErrorIs(t, err, expectedErr)`
  - `assert.ElementsMatch(t, expected, actual)` for unordered slices.

### Helper Functions

- Mark helpers with `t.Helper()`:
  ```go
  func assertMapContains(t *testing.T, expected, actual map[string]any) {
      t.Helper()
      // Verification logic
  }
  ```
- Keep helpers focused on a single task.

### Mock Usage

- Use mocks from the `machines/mocks` package.
- Verify expectations with `mockObj.AssertExpectations(t)`.

## Example Test Pattern

```go
func TestResponseMethods(t *testing.T) {
    t.Parallel()

    t.Run("type detection", func(t *testing.T) {
        tests := []struct {
            name     string
            input    any
            expected data.Types
        }{
            {"string value", "test", data.STRING},
            {"integer value", 42, data.INT},
            {"bool value", true, data.BOOL},
        }

        for _, tc := range tests {
            tc := tc // Capture range variable
            t.Run(tc.name, func(t *testing.T) {
                // Setup
                handler := slog.NewTextHandler(os.Stdout, nil)
                result := newEvalResult(handler, tc.input, 0, "test-id")

                // Verify
                assert.Equal(t, tc.expected, result.Type(), "Type detection should match expected type")
            })
        }
    })
}
```
