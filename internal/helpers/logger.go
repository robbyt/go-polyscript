package helpers

import (
	"log/slog"
	"os"
)

// SetupLogger creates a properly configured logger for virtual machine implementations.
// If the provided handler is nil, it creates a default handler with appropriate grouping.
//
// Parameters:
//   - handler: The slog.Handler to use, or nil for defaults
//   - vmName: The name of the virtual machine (e.g., "starlark", "risor")
//   - groupName: Optional additional group name within the VM
//
// Returns:
//   - The configured handler
//   - A logger created from the handler
func SetupLogger(handler slog.Handler, vmName string, groupName string) (slog.Handler, *slog.Logger) {
	if handler == nil {
		defaultHandler := slog.NewTextHandler(os.Stdout, nil)
		handler = defaultHandler.WithGroup(vmName)
		// Create a logger from the handler
		defaultLogger := slog.New(handler)
		defaultLogger.Warn("Handler is nil, using the default logger configuration.")
	}

	var logger *slog.Logger
	if groupName != "" {
		logger = slog.New(handler.WithGroup(groupName))
	} else {
		logger = slog.New(handler)
	}

	return handler, logger
}
