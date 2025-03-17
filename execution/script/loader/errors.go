package loader

import "errors"

// ErrNoScript is returned when there is no script to return.
var ErrSchemeUnsupported = errors.New("unsupported scheme")
var ErrScriptNotAvailable = errors.New("script not available")
var ErrInputEmpty = errors.New("input is empty")
