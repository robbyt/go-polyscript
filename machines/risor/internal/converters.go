package internal

import (
	risorLib "github.com/risor-io/risor"
)

// ConvertToRisorOptions converts a Go map into Risor VM options object.
// The input data will be wrapped in a single "ctx" object passed to the VM.
//
// For example, if the inputData is {"foo": "bar", "baz": 123}, the output will be:
//
//	[]risorLib.Option{
//	  risorLib.WithGlobal("ctx", map[string]any{
//	    "foo": "bar",
//	    "baz": 123,
//	  }),
//	}
func ConvertToRisorOptions(ctxKey string, inputData map[string]any) []risorLib.Option {
	return []risorLib.Option{
		risorLib.WithGlobal(ctxKey, inputData),
	}
}
