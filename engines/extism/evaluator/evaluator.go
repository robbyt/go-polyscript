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
	return data.LoadInputData(ctx, be.logHandler.WithGroup("Evaluator"), be.getDataProvider())
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
//     so callers can tell something was elided
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
	// Call the function. Unless the compiler disabled CloseOnContextDone, a
	// cancelled or expired ctx stops the guest mid-execution.
	startTime := time.Now()
	exit, output, err := instance.CallWithContext(ctx, entryPoint, inputJSON)
	execTime := time.Since(startTime)
	if err != nil {
		if ctx.Err() != nil {
			return nil, execTime, fmt.Errorf("execution cancelled: %w (call error: %w)", ctx.Err(), err)
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
		// Use a cancel-immune ctx for cleanup so a cancelled Eval ctx
		// doesn't abort the instance Close.
		if err := instance.Close(context.WithoutCancel(ctx)); err != nil {
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

	content := be.execUnit.GetContent()
	if content == nil {
		return nil, fmt.Errorf("content is nil")
	}

	// Get execution ID
	exeID := be.execUnit.GetID()
	if exeID == "" {
		return nil, fmt.Errorf("exeID is empty")
	}
	logger = logger.With("exeID", exeID)

	// 1. Type assert to WASM module
	wasmExe, ok := content.(*compiler.Executable)
	if !ok {
		return nil, fmt.Errorf(
			"invalid executable type: expected *Executable, got %T",
			content,
		)
	}

	// 2. Load the input data, convert it to JSON and execute the program
	// inside WithPlugin. It holds the executable's read lock until fn
	// returns, so a concurrent Close also waits for data loading. Once Close
	// has been called it returns compiler.ErrExecutableClosed without calling
	// fn, so the data provider is never called. Its error is returned
	// unwrapped; each step inside fn wraps its own.
	var result *execResult
	err := wasmExe.WithPlugin(func(plugin adapters.CompiledPlugin) error {
		rawInputData, loadErr := be.loadInputData(ctx)
		if loadErr != nil {
			return fmt.Errorf("failed to get input data: %w", loadErr)
		}

		runtimeData, convErr := internal.ConvertToExtismFormat(rawInputData)
		if convErr != nil {
			return fmt.Errorf("failed to marshal input data: %w", convErr)
		}

		var execErr error
		result, execErr = be.exec(
			ctx, plugin,
			wasmExe.GetEntryPoint(),
			adapters.NewPluginInstanceConfig(),
			runtimeData,
		)
		if execErr != nil {
			return fmt.Errorf("exec error: %w", execErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	logger.DebugContext(ctx, "exec completed", "result", result)

	// 3. Collect results
	result.scriptExeID = exeID
	return result, nil
}

// AddDataToContext implements the data.Setter interface which stores and prepares runtime data
// which can be eventually passed to the Eval method.
func (be *Evaluator) AddDataToContext(
	ctx context.Context,
	d map[string]any,
) (context.Context, error) {
	return data.AddDataToContextFromProvider(ctx, be.logHandler.WithGroup("Evaluator"), be.getDataProvider(), d)
}

// Close implements platform.Evaluator. It releases the compiled plugin and its
// wazero runtime, waiting first for in-flight Eval calls to finish, including
// their data loading. ctx cannot interrupt that wait; cancelling an Eval's ctx
// shortens it only if the guest stops and the data provider honours
// cancellation. Eval calls made once Close has been called, including while
// it waits, return an error wrapping compiler.ErrExecutableClosed without
// calling the data provider. A data provider must not call Close or Eval on
// the same Evaluator, since Close would wait for the Eval that is calling it.
// It is safe to call more than once and on a nil Evaluator.
func (be *Evaluator) Close(ctx context.Context) error {
	if be == nil {
		return nil
	}
	return be.execUnit.Close(ctx)
}
