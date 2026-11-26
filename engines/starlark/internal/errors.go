package internal

import "errors"

// ErrDictKeyCollision is returned when two distinct Starlark dict keys
// stringify to the same Go map key, e.g. {1: "a", "1": "b"}.
var ErrDictKeyCollision = errors.New("starlark dict keys collide when converted to string map keys")
