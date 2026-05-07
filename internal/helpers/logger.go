package helpers

import (
	"log/slog"
)

// SetupLogger creates a configured logger and handler for an engine implementation.
//
// If handler is nil, the handler returned by slog.Default().Handler() is used,
// wrapped with engineName as a slog group so log records carry an
// "<engineName>.<…>" prefix (e.g. engineName "risor" produces "risor.<…>").
// This lets hosts wire one configuration (level, format, sink) globally and
// have the engine inherit it. Hosts that pass an explicit handler get it back
// unchanged — no engineName grouping is applied in that case (the caller is
// presumed to have configured the handler the way they want).
//
// Parameters:
//   - handler: handler to use, or nil to inherit from slog.Default().Handler()
//   - engineName: slog group name applied only when handler is nil (e.g. "risor")
//   - groupName: optional sub-group, applied to the returned logger only
//     (always honored regardless of whether handler was nil)
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
