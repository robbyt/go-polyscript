package data

// Types of an object as a string.
type Types string

// These valid types as constants, limited for our use. Every engine's
// EvaluatorResponse.Type() returns one of these values; ERROR doubles as the
// fallback for results that have no counterpart in this list.
const (
	BOOL     Types = "bool"
	BYTES    Types = "bytes"
	ERROR    Types = "error"
	FUNCTION Types = "function"
	INT      Types = "int"
	MAP      Types = "map"
	STRING   Types = "string"
	NONE     Types = "none"
	FLOAT    Types = "float"
	LIST     Types = "list"
	TUPLE    Types = "tuple"
	SET      Types = "set"
)
