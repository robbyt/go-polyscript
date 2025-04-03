# Data Preparation Examples Cleanup Plan

## Issue Summary
- Core bug: The Extism example in the data-prep directory fails with "input string is empty" error
- Root cause: Data is nested under "script_data" by ContextProvider, but Extism expects a top-level "input" field
- This is a broader architectural issue in how static data (compile-time) and dynamic data (runtime) are combined

## Architectural Changes Implemented
1. ExecutableUnit improvements:
   - Removed redundant ScriptData field (data now handled through DataProvider)
   - Made GetScriptData() backward compatible by retrieving data from provider

2. Provider enhancements:
   - Added sentinel error ErrStaticProviderNoRuntimeUpdates in StaticProvider
   - Updated CompositeProvider to handle this sentinel error appropriately

3. Testing:
   - Tests added to bytecodeEvaluator_test.go confirm the changes resolve the issue
   - TestScriptDataAndDataProvider verifies correct data handling

## Current Status (April 2, 2025)
- Core architecture changes have been implemented and tested successfully
- The Extism example now compiles but still encounters data structure issues
- Risor and Starlark examples need similar attention

## Remaining Work
1. Extism Example Fix:
   - Problem: Main example needs to be updated to handle the nested data structure
   - Extism tests pass when data is directly at the top level or when script is updated to access data via script_data

2. Risor Example Fix:
   - Problem: Risor script expects data at top level (ctx["name"]) but data is nested (ctx["script_data"]["name"])
   - Options:
     a. Update script to handle both structures with null checks
     b. Modify data provider/converter to expose data at both levels

3. Starlark Example:
   - Same issue as Risor, will need similar approach

## Decision Points
1. How to maintain backward compatibility:
   - Should we modify converters to "flatten" nested data?
   - Should we update all example scripts to handle both data structures?
   - Should we add utility functions to help scripts handle both layouts?

2. Documentation needs:
   - Update documentation to reflect the latest best practices
   - Clarify data flow between providers and execution environments

## Plan Priorities
1. Fix the examples to work correctly with current architecture
2. Add better error handling for data access in example scripts
3. Document the data structure patterns for future script development