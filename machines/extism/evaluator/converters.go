package evaluator

import (
	"crypto/rand"
	"encoding/json"

	extismSDK "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

// convertToExtismFormat converts a Go map into JSON format for the Extism VM.
func convertToExtismFormat(inputData map[string]any) ([]byte, error) {
	if len(inputData) == 0 {
		return nil, nil
	}
	return json.Marshal(inputData)
}

func (be *BytecodeEvaluator) getPluginInstanceConfig() extismSDK.PluginInstanceConfig {
	// Create base config if none provided
	moduleConfig := wazero.NewModuleConfig()

	// Configure with recommended options
	moduleConfig = moduleConfig.
		WithSysWalltime().          // For consistent time functions
		WithSysNanotime().          // For high-precision timing
		WithRandSource(rand.Reader) // For secure randomness

	return extismSDK.PluginInstanceConfig{
		ModuleConfig: moduleConfig,
	}
}
