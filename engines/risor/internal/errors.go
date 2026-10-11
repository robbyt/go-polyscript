package internal

import "errors"

// ErrUintOverflow is returned when input data holds an unsigned integer above
// math.MaxInt64. Risor stores integers as int64 and would silently wrap it.
var ErrUintOverflow = errors.New("unsigned integer exceeds Risor's int64 range")
