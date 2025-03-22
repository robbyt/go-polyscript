package extism

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/robbyt/go-polyscript/engine"
	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/script"

	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// BytecodeEvaluator executes compiled WASM modules with provided runtime data
type BytecodeEvaluator struct {
	ctxKey     string // Variable name used to access data in WASM
	execUnit   *script.ExecutableUnit
	logHandler slog.Handler
	logger     *slog.Logger
}

func NewBytecodeEvaluator(handler slog.Handler, execUnit *script.ExecutableUnit) *BytecodeEvaluator {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup("extism")
		// Create a logger from the handler rather than using slog directly
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	return &BytecodeEvaluator{
		ctxKey:     constants.Ctx,
		execUnit:   execUnit,
		logHandler: handler,
		logger:     slog.New(handler.WithGroup("BytecodeEvaluator")),
	}
}

func (be *BytecodeEvaluator) getLogger() *slog.Logger {
	return be.logger
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

// exec handles WASM-specific execution details
// TODO: Consider refactoring to use interfaces instead of concrete types for better testability
// Currently, the error paths are difficult to test because Extism SDK uses concrete types.
func (be *BytecodeEvaluator) exec(
	ctx context.Context,
	plugin compiledPlugin,
	entryPoint string,
	instanceConfig extismSDK.PluginInstanceConfig,
	inputJSON []byte,
) (*execResult, error) {
	logger := be.getLogger()

	instance, err := plugin.Instance(ctx, instanceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin instance: %w", err)
	}
	defer instance.Close(ctx)

	// Call the function (context handles timeout)
	startTime := time.Now()
	exit, output, err := instance.CallWithContext(ctx, entryPoint, inputJSON)
	execTime := time.Since(startTime)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("execution cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("execution failed: %w", err)
	}
	if exit != 0 {
		// TODO should we return the output in this case?
		return nil, fmt.Errorf("function returned non-zero exit code: %d", exit)
	}

	// Try to parse output as JSON with number handling
	var result any
	d := json.NewDecoder(bytes.NewReader(output))
	d.UseNumber() // Preserve number types
	if err := d.Decode(&result); err != nil {
		// If not JSON, use raw output as string
		result = string(output)
	}

	// Convert json.Number to int for specific fields
	if m, ok := result.(map[string]any); ok {
		for k, v := range m {
			if num, ok := v.(json.Number); ok {
				if strings.HasSuffix(k, "_count") || k == "count" {
					if n, err := num.Int64(); err == nil {
						m[k] = int(n)
					}
				}
			}
		}
	}

	logger.Debug("execution complete",
		"result", result,
		"execTime", execTime,
	)

	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the WASM module.
func (be *BytecodeEvaluator) loadInputData(ctx context.Context) (map[string]any, error) {
	logger := be.getLogger()

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

func marshalInputData(inputData map[string]any) ([]byte, error) {
	if len(inputData) == 0 {
		return nil, nil
	}
	return json.Marshal(inputData)
}

// Eval implements engine.Evaluator
// TODO: Some error paths in this method are hard to test with the current design
// Consider adding more integration tests to cover these paths.
func (be *BytecodeEvaluator) Eval(ctx context.Context) (engine.EvaluatorResponse, error) {
	logger := be.getLogger()

	// Validate executable unit
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

	// Get execution ID
	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("execution ID is empty")
	}
	logger = logger.With("exeID", exeID)

	// Get input data from the provider
	inputData, err := be.loadInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// Marshal input data to JSON
	inputJSON, err := marshalInputData(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	// Execute WASM module
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
//
// The method accepts HTTP requests, maps, and other data types, converting them
// appropriately for WebAssembly execution. This enables separation of data preparation
// from evaluation, supporting distributed processing architectures.
//
// Example:
//  scriptData := map[string]any{"greeting": "Hello, World!"}
//  enrichedCtx, err := evaluator.PrepareContext(ctx, request, scriptData)
//  if err != nil {
//      return err
//  }
//  result, err := evaluator.Eval(enrichedCtx)
func (be *BytecodeEvaluator) PrepareContext(ctx context.Context, data ...any) (context.Context, error) {
	logger := be.getLogger()

	// Check if we have a data provider
	if be.execUnit == nil || be.execUnit.GetDataProvider() == nil {
		logger.WarnContext(ctx, "no data provider available for context preparation")
		return ctx, fmt.Errorf("no data provider available")
	}

	// Use the data provider to store the raw data
	enrichedCtx, err := be.execUnit.GetDataProvider().AddDataToContext(ctx, data...)
	if err != nil {
		logger.ErrorContext(ctx, "failed to prepare context", "error", err)
		// Return the partial context even with errors, as it may have some usable data
	}

	return enrichedCtx, err
}
