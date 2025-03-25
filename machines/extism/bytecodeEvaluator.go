package extism

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/execution/script"
	"github.com/robbyt/go-polyscript/internal/helpers"

	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// BytecodeEvaluator executes compiled WASM modules with provided runtime data
type BytecodeEvaluator struct {
	execUnit   *script.ExecutableUnit
	logHandler slog.Handler
	logger     *slog.Logger
}

func NewBytecodeEvaluator(handler slog.Handler, execUnit *script.ExecutableUnit) *BytecodeEvaluator {
	handler, logger := helpers.SetupLogger(handler, "extism", "BytecodeEvaluator")

	return &BytecodeEvaluator{
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
}

func (be *BytecodeEvaluator) String() string {
	return "extism.BytecodeEvaluator"
}

func (be *BytecodeEvaluator) getPluginInstanceConfig() extismSDK.PluginInstanceConfig {
	// Create base config if none provided
	moduleConfig := wazero.NewModuleConfig()

	// Configure with recommended options
	moduleConfig = moduleConfig.
		WithSysWalltime().          // For consistent time functions
		WithSysNanotime().          // For high-precision timing
		WithRandSource(rand.Reader) // For secure randomness

	return extismSDK.PluginInstanceConfig{
		ModuleConfig: moduleConfig,
	}
}

// execHelper is a utility function to handle common execution logic
// Extracted to make unit testing easier
func execHelper(
	ctx context.Context,
	logger *slog.Logger,
	instance sdkPluginInstanceConfig,
	entryPoint string,
	inputJSON []byte,
) (any, time.Duration, error) {
	// Call the function (context handles timeout)
	startTime := time.Now()
	exit, output, err := instance.CallWithContext(ctx, entryPoint, inputJSON)
	execTime := time.Since(startTime)
	if err != nil {
		if ctx.Err() != nil {
			return nil, execTime, fmt.Errorf("execution cancelled: %w", ctx.Err())
		}
		return nil, execTime, fmt.Errorf("execution failed: %w", err)
	}
	if exit != 0 {
		// TODO should we return the output in this case?
		return nil, execTime, fmt.Errorf("function returned non-zero exit code: %d", exit)
	}

	// Try to parse output as JSON with number handling
	var result any
	d := json.NewDecoder(bytes.NewReader(output))
	d.UseNumber() // Preserve number types
	if err := d.Decode(&result); err != nil {
		// If not JSON, use raw output as string
		result = string(output)
	}

	result = fixJSONNumberTypes(result)

	logger.Debug("execution complete",
		"result", result,
		"execTime", execTime,
	)

	return result, execTime, nil
}

// exec handles WASM-specific execution details
// Using the interface and helper function to improve testability
func (be *BytecodeEvaluator) exec(
	ctx context.Context,
	plugin compiledPlugin,
	entryPoint string,
	instanceConfig extismSDK.PluginInstanceConfig,
	inputJSON []byte,
) (*execResult, error) {
	logger := be.logger.WithGroup("exec")

	instance, err := plugin.Instance(ctx, instanceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin instance: %w", err)
	}
	defer instance.Close(ctx)

	// Use the helper function for execution
	result, execTime, err := execHelper(ctx, logger, instance, entryPoint, inputJSON)
	if err != nil {
		return nil, err
	}

	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the WASM module.
//
// Note: This method handles data retrieval using the GetData method of the provider.
// It's part of the standard pattern used across all VMs as defined in the
// context harmonization plan.
func (be *BytecodeEvaluator) loadInputData(ctx context.Context) (map[string]any, error) {
	logger := be.logger.WithGroup("loadInputData")

	// If no executable unit or data provider, return empty map
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		logger.WarnContext(ctx, "no data provider available, using empty data")
		return make(map[string]any), nil
	}

	// Get input data from provider
	inputData, err := be.execUnit.GetDataProvider().GetData(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get input data from provider", "error", err)
		return nil, err
	}

	logger.DebugContext(ctx, "input data loaded from provider", "inputData", inputData)
	return inputData, nil
}

// Eval implements engine.Evaluator
// TODO: Some error paths in this method are hard to test with the current design
// Consider adding more integration tests to cover these paths.
func (be *BytecodeEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	if be.execUnit.GetContent() == nil {
		return nil, fmt.Errorf("content is nil")
	}

	// Get and validate bytecode
	bytecode := be.execUnit.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	// Get execution ID
	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("execution ID is empty")
	}
	logger = logger.With("exeID", exeID)

	// Type assert to WASM module
	wasmExe, ok := be.execUnit.GetContent().(*Executable)
	if !ok {
		return nil, fmt.Errorf("invalid executable type: expected *Executable, got %T", be.execUnit.GetContent())
	}

	// Get compiled plugin
	plugin := wasmExe.GetExtismByteCode()
	if plugin == nil {
		return nil, fmt.Errorf("compiled plugin is nil")
	}

	// 1. Get the raw input data
	inputData, err := be.loadInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// 2. Convert input data to JSON for passing into the WASM VM
	inputJSON, err := marshalInputData(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	// 3. Execute in the VM
	result, err := be.exec(
		ctx, plugin,
		wasmExe.GetEntryPoint(),
		be.getPluginInstanceConfig(),
		inputJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}

	// Set execution ID and return
	result.scriptExeID = exeID
	logger.Debug("execution complete")
	return result, nil
}

// PrepareContext implements the EvalDataPreparer interface for Extism WebAssembly modules.
// It enriches the provided context with data for script evaluation, using the
// ExecutableUnit's DataProvider to store the data.
func (be *BytecodeEvaluator) PrepareContext(ctx context.Context, d ...any) (context.Context, error) {
	logger := be.logger.WithGroup("PrepareContext")

	// Use the shared helper function for context preparation
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		return ctx, fmt.Errorf("no data provider available")
	}

	return data.PrepareContextHelper(
		ctx,
		logger,
		be.execUnit.GetDataProvider(),
		d...,
	)
}
