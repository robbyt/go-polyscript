package data

// Types of an object as a string.
type Types string

// These valid types as constants, limited for our use.
const (
	BOOL     Types = "bool"
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
