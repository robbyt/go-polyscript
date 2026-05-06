package helpers

import (
	"log/slog"
)

// SetupLogger creates a configured logger and handler for an engine implementation.
//
// If handler is nil, the slog default handler ([slog.Default]) is used. This lets
// hosts wire one configuration (level, format, sink) globally and have the engine
// inherit it. Hosts that want a different handler should pass one explicitly.
//
// Parameters:
//   - handler: handler to use, or nil to inherit from slog.Default()
//   - engineName: short engine identifier added as a slog group (e.g. "risor")
//   - groupName: optional sub-group within the engine
//
// Returns the resolved handler and a logger built from it.
func SetupLogger(
	handler slog.Handler,
	engineName string,
	groupName string,
) (slog.Handler, *slog.Logger) {
	if handler == nil {
		handler = slog.Default().Handler().WithGroup(engineName)
	}

	if groupName != "" {
		return handler, slog.New(handler.WithGroup(groupName))
	}
	return handler, slog.New(handler)
}
