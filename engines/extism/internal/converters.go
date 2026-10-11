package internal

import (
	"encoding/json"
)

// ConvertToExtismFormat converts a Go map into JSON format for the Extism engine.
// Empty or nil input is sent as an empty JSON object ("{}") rather than zero
// bytes, so a guest that parses its input as JSON always receives a valid object.
func ConvertToExtismFormat(inputData map[string]any) ([]byte, error) {
	if inputData == nil {
		inputData = map[string]any{}
	}
	return json.Marshal(inputData)
}
