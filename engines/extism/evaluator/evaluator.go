package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	extismSDK "github.com/extism/go-sdk"
	"github.com/robbyt/go-polyscript/engines/extism/adapters"
	"github.com/robbyt/go-polyscript/engines/extism/compiler"
	"github.com/robbyt/go-polyscript/engines/extism/internal"
	"github.com/robbyt/go-polyscript/internal/helpers"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/data"
	"github.com/robbyt/go-polyscript/platform/script"
)

// Evaluator executes compiled WASM modules with provided runtime data
type Evaluator struct {
	execUnit           *script.ExecutableUnit
	logHandler         slog.Handler
	logger             *slog.Logger
	exitOutputMaxBytes int
}

// Option configures an [Evaluator]. Pass to [New].
type Option func(*Evaluator)

// WithExitOutputMaxBytes caps the WASM-output snippet included in non-zero-
// exit-code error messages. A zero value (or unset) uses
// [defaultExitOutputMaxBytes] (1024); a negative value disables the cap so
// the full output is included unchanged.
//
// This mirrors the cap semantics of HTTPOptions.MaxBodySize from the
// platform script loader.
func WithExitOutputMaxBytes(n int) Option {
	return func(e *Evaluator) { e.exitOutputMaxBytes = n }
}

// New creates a new Evaluator object.
func New(
	handler slog.Handler,
	execUnit *script.ExecutableUnit,
	opts ...Option,
) *Evaluator {
	handler, logger := helpers.SetupLogger(handler, "extism", "Evaluator")

	e := &Evaluator{
		execUnit:   execUnit,
		logHandler: handler,
		logger:     logger,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

func (be *Evaluator) String() string {
	return "extism.Evaluator"
}

// getDataProvider returns the data provider from the executable unit, or nil if unavailable.
func (be *Evaluator) getDataProvider() data.Provider {
	if be.execUnit == nil {
		return nil
	}
	return be.execUnit.GetDataProvider()
}

// loadInputData retrieves input data using the data provider in the executable unit.
// Returns a map that will be used as input for the WASM module.
func (be *Evaluator) loadInputData(ctx context.Context) (map[string]any, error) {
	return data.LoadInputData(ctx, be.logger.WithGroup("loadInputData"), be.getDataProvider())
}

// defaultExitOutputMaxBytes is the cap applied when an Evaluator is built
// without [WithExitOutputMaxBytes] (or with the zero value). It keeps a
// multi-megabyte plugin payload from blowing up logs while still surfacing
// enough bytes for the host to diagnose a misbehaving plugin.
const defaultExitOutputMaxBytes = 1024

// formatExitOutput returns a parenthesized " (output: %q)" suffix for
// inclusion in non-zero-exit-code errors. Returns "" when output is empty,
// so callers don't get a noisy '(output: "")' tail.
//
// maxBytes governs truncation:
//
//   - zero  → fall back to [defaultExitOutputMaxBytes]
//   - >0    → truncate at that value; the original byte length is surfaced
//             so callers can tell something was elided
//   - <0    → no cap; the full output is quoted unchanged
func formatExitOutput(output []byte, maxBytes int) string {
	if len(output) == 0 {
		return ""
	}
	if maxBytes == 0 {
		maxBytes = defaultExitOutputMaxBytes
	}
	if maxBytes < 0 || len(output) <= maxBytes {
		return fmt.Sprintf(" (output: %q)", output)
	}
	return fmt.Sprintf(
		" (output: %q, truncated from %d bytes)",
		output[:maxBytes], len(output),
	)
}

// execHelper is a utility function to handle common execution logic
// Extracted to make unit testing easier
func execHelper(
	ctx context.Context,
	logger *slog.Logger,
	instance adapters.SdkPluginInstanceConfig,
	entryPoint string,
	inputJSON []byte,
	exitOutputMaxBytes int,
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
		return nil, execTime, fmt.Errorf(
			"function returned non-zero exit code: %d%s",
			exit, formatExitOutput(output, exitOutputMaxBytes),
		)
	}

	// Try to parse output as JSON with number handling
	var result any
	d := json.NewDecoder(bytes.NewReader(output))
	d.UseNumber() // Preserve number types
	if err := d.Decode(&result); err != nil {
		// If not JSON, use raw output as string
		result = string(output)
	}

	result = internal.FixJSONNumberTypes(result)

	logger.Debug("execution complete",
		"result", result,
		"execTime", execTime,
	)

	return result, execTime, nil
}

// exec handles WASM-specific execution details
// Using the interface and helper function to improve testability
func (be *Evaluator) exec(
	ctx context.Context,
	plugin adapters.CompiledPlugin,
	entryPoint string,
	instanceConfig extismSDK.PluginInstanceConfig,
	inputJSON []byte,
) (*execResult, error) {
	logger := be.logger.WithGroup("exec")

	instance, err := plugin.Instance(ctx, instanceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin instance: %w", err)
	}
	defer func() {
		if err := instance.Close(ctx); err != nil {
			logger.Warn("Failed to close Extism plugin instance", "error", err)
		}
	}()

	// Use the helper function for execution
	result, execTime, err := execHelper(
		ctx, logger, instance, entryPoint, inputJSON, be.exitOutputMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("extism execution error: %w", err)
	}
	return newEvalResult(be.logHandler, result, execTime, ""), nil
}

// Eval implements evaluation.Evaluator
func (be *Evaluator) Eval(ctx context.Context) (platform.EvaluatorResponse, error) {
	logger := be.logger.WithGroup("Eval")
	if be.execUnit == nil {
		return nil, fmt.Errorf("executable unit is nil")
	}

	if be.execUnit.GetContent() == nil {
		return nil, fmt.Errorf("content is nil")
	}

	// Get bytecode from executable unit
	bytecode := be.execUnit.GetContent().GetByteCode()
	if bytecode == nil {
		return nil, fmt.Errorf("bytecode is nil")
	}

	// Get execution ID
	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("exeID is empty")
	}
	logger = logger.With("exeID", exeID)

	// 1. Type assert to WASM module, and get the compiled plugin object
	wasmExe, ok := be.execUnit.GetContent().(*compiler.Executable)
	if !ok {
		return nil, fmt.Errorf(
			"invalid executable type: expected *Executable, got %T",
			be.execUnit.GetContent(),
		)
	}
	plugin := wasmExe.GetExtismByteCode()
	if plugin == nil {
		return nil, fmt.Errorf("compiled plugin is nil")
	}

	// 2. Get the raw input data
	rawInputData, err := be.loadInputData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get input data: %w", err)
	}

	// 3. Convert input data to JSON for passing into the WASM engine
	runtimeData, err := internal.ConvertToExtismFormat(rawInputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	// 4. Execute the program
	result, err := be.exec(
		ctx, plugin,
		wasmExe.GetEntryPoint(),
		adapters.NewPluginInstanceConfig(),
		runtimeData,
	)
	if err != nil {
		return nil, fmt.Errorf("exec error: %w", err)
	}
	logger.DebugContext(ctx, "exec completed", "result", result)

	// 5. Collect results
	result.scriptExeID = exeID
	return result, nil
}

// AddDataToContext implements the data.Setter interface which stores and prepares runtime data
// which can be eventually passed to the Eval method.
func (be *Evaluator) AddDataToContext(
	ctx context.Context,
	d ...map[string]any,
) (context.Context, error) {
	return data.AddDataToContextFromProvider(ctx, be.logger.WithGroup("AddDataToContext"), be.getDataProvider(), d...)
}
