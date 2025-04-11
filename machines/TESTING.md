# Testing Guidelines for go-polyscript

This document outlines the standardized testing patterns for go-polyscript. Following these guidelines ensures consistency, maintainability, and comprehensive test coverage across the codebase.

## Core Testing Principles

1. **Consistency**: Use standardized test patterns across all packages and VM implementations
2. **Clarity**: Write clear, self-documenting tests with logical organization
3. **Comprehensiveness**: Test both success paths and error conditions thoroughly
4. **Efficiency**: Avoid duplication through table-driven tests and helper functions

## Test Structure and Organization

### Test Function Naming

Use these consistent naming patterns:

| Component | Test Function Name |
|-----------|-------------------|
| Compiler creation | `TestNewCompiler` |
| Compilation | `TestCompiler_Compile` |
| Options | `TestCompilerOptions` or `TestCompilerOptionsDetailed` |
| Executables | `TestExecutable` |
| Evaluator execution | `TestEvaluator_Evaluate` |
| Context preparation | `TestEvaluator_PrepareContext` |
| Response handling | `TestResponseMethods` |
| Type conversion | `TestToGoType`/`TestToMachineType` |

### Subtest Organization

- Use the Go subtests pattern with `t.Run()` to group related test cases
- Organize subtests into logical categories:
  ```go
  t.Run("success cases", func(t *testing.T) {
      // Tests for normal operation
  })
  t.Run("error cases", func(t *testing.T) {
      // Tests for error handling
  })
  ```
- Keep verification code separate from setup code
- Use descriptive subtest names instead of relying on comments

### Test Parallelization

- Use `t.Parallel()` ONLY in parent test functions, not in subtests
  ```go
  func TestResponseMethods(t *testing.T) {
    t.Parallel() // Only in parent test function
    for _, tc := range tests {
      t.Run(tc.name, func(t *testing.T) { ... })
    }
  }
  ```

## Testify Usage Standards

Always use the testify library consistently:

### Assertion Types

- **require**: For conditions that must pass for the test to continue
- **assert**: For conditions that shouldn't halt the test if they fail
- **mock**: For creating and verifying mock behaviors

### Preferred Assertion Methods

| Purpose | Preferred Method |
|---------|------------------|
| Value equality | `assert.Equal(t, expected, actual)` |
| Negative comparison | `assert.NotEqual(t, unexpected, actual)` |
| Boolean checks | `assert.True(t, condition)` / `assert.False(t, condition)` |
| Nil/NotNil checks | `assert.Nil(t, value)` / `assert.NotNil(t, value)` |
| Collections | `assert.Contains(t, container, element)` |
| Order-independent slice equality | `assert.ElementsMatch(t, expected, actual)` |
| No error | `require.NoError(t, err)` |
| Error occurred | `require.Error(t, err)` |
| Specific error | `require.ErrorIs(t, err, expectedErr)` (preferred over string comparison) |
| Error message check | `require.Contains(t, err.Error(), "expected message")` |
| Mock setup | `mock.On("MethodName", mock.Anything).Return(returnValue)` |

Include meaningful messages with assertions to aid debugging:
```go
assert.Equal(t, expected, actual, "User ID should match after conversion")
```

## Component-Specific Guidelines

### Compiler Tests

| Component | Key Testing Focus |
|-----------|------------------|
| Compiler Creation | • Creation with default settings<br>• Creation with various options<br>• Error handling for invalid options |
| Compilation | • Successful compilation of valid scripts<br>• Error handling for nil content, empty content, invalid syntax<br>• VM-specific compiler features |
| Options | • Group related options under logical sections<br>• Test both valid and invalid option values<br>• Test default values and option combinations |
| Evaluator | • Success paths with various input data types<br>• Context cancellation handling<br>• Nil executable/bytecode testing<br>• Metadata verification (execution time, script ID) |
| Response | • All methods: Type, Interface, Inspect, String, GetScriptExeID, GetExecTime<br>• All data types: primitives, collections, complex nested structures<br>• Error handling for invalid types |
| Type Conversion | • Bidirectional conversions: Go → VM and VM → Go<br>• Organized by type (primitives, collections, complex, errors)<br>• VM-specific type handling |

## Best Practices

### Test Helper Functions

- Always mark test helpers with `t.Helper()` to improve error reporting:
  ```go
  func assertMapContainsExpectedHelper(t *testing.T, expected, actual map[string]any) {
      t.Helper() // Marks this as a helper function
      // Verification logic
  }
  ```
- Keep helper functions focused on a single verification task
- Extract common verification logic for consistency and readability

### Mock Usage

- Use the standard mocks from `machines/mocks` package
- Set specific expectations for each test case:
  ```go
  mockObj.On("MethodName", mock.MatchedBy(func(arg string) bool {
      return strings.Contains(arg, "expected")
  })).Return("result", nil)
  ```
- Verify all expectations with `mockObj.AssertExpectations(t)` at the end of tests
- Use typed nil values when needed for interface parameters

## Example Test Pattern

Here's the recommended pattern for consistent table-driven tests:

```go
func TestResponseMethods(t *testing.T) {
    t.Parallel() // Only in parent test function
    
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
                
                // Verify - separate from setup
                assert.Equal(t, tc.expected, result.Type(), "Type detection should match expected type")
            })
        }
    })
}
```

## Test Quality Checklist

✅ Use table-driven tests for similar test cases  
✅ Apply proper parallelization with `t.Parallel()` only in parent tests  
✅ Capture range variables in loops to prevent race conditions  
✅ Separate setup code from verification code  
✅ Test both success paths and error conditions  
✅ Use `require.ErrorIs()` instead of string comparison for errors  
✅ Provide descriptive assertion messages  
✅ Mark helper functions with `t.Helper()`  
✅ Verify all mock expectations after tests  
✅ Check error returns from all functions that return errors  
✅ Use local test servers instead of external dependencies  
✅ Organize imports consistently (stdlib first, then external, then local)  
✅ Group related tests under logical parent tests

## Test Coverage Improvements

The codebase underwent significant test improvements with metrics tracked:

| Package | Original Coverage | Final Coverage | Notes |
|---------|------------------|----------------|-------|
| engine | 0.0% | 100% | Added comprehensive tests |
| machines/extism/evaluator | 62.5% | 90.4% | Added tests for edge cases |
| machines/risor/evaluator | 82.6% | 88.4% | Improved structure and coverage |
| machines/starlark/evaluator | 64.3% | 76.5% | Harmonized test structure |

Key improvements included:
- Reduction of test file size while maintaining or improving coverage
- Extracting common test patterns into helper functions
- Standardizing test structure across different VM implementations
- Improving test reliability by removing external dependencies
- Enhancing error case testing and edge case coverage
