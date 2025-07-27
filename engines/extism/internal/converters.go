package internal

import (
	"encoding/json"
)

// ConvertToExtismFormat converts a Go map into JSON format for the Extism engine.
func ConvertToExtismFormat(inputData map[string]any) ([]byte, error) {
	if len(inputData) == 0 {
		return nil, nil
	}
	return json.Marshal(inputData)
}
