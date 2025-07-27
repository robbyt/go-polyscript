package helpers

import (
	"log/slog"
	"os"
)

// SetupLogger creates a properly configured logger for script engine implementations.
// If the provided handler is nil, it creates a default handler with appropriate grouping.
//
// Parameters:
//   - handler: The slog.Handler to use, or nil for defaults
//   - vmName: The name of the script engine (e.g., "starlark", "risor")
//   - groupName: Optional additional group name within the engine
//
// Returns:
//   - The configured handler
//   - A logger created from the handler
func SetupLogger(
	handler slog.Handler,
	engineName string,
	groupName string,
) (slog.Handler, *slog.Logger) {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup(engineName)
		// Create a logger from the handler
		defaultLogger := slog.New(handler)
		defaultLogger.Debug("Handler is nil, using the default logger configuration.")
	}

	var logger *slog.Logger
	if groupName != "" {
		logger = slog.New(handler.WithGroup(groupName))
	} else {
		logger = slog.New(handler)
	}

	return handler, logger
}
