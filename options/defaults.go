package options

import (
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript/execution/constants"
	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/types"
)

// DefaultConfig initializes a Config with sensible defaults
func DefaultConfig(machineType types.Type) *Config {
	cfg := &Config{}
	cfg.SetMachineType(machineType)
	cfg.SetHandler(DefaultLoggingHandler())
	cfg.SetDataProvider(DefaultDataProvider())
	return cfg
}

// DefaultLoggingHandler returns the default logging handler
func DefaultLoggingHandler() slog.Handler {
	return slog.NewTextHandler(os.Stdout, nil)
}

// DefaultDataProvider returns the default data provider
func DefaultDataProvider() data.Provider {
	return data.NewContextProvider(constants.EvalData)
}

// WithDefaults applies default values to any config properties that are nil
func WithDefaults() Option {
	return func(c *Config) error {
		if c.handler == nil {
			c.handler = DefaultLoggingHandler()
		}

		if c.dataProvider == nil {
			c.dataProvider = DefaultDataProvider()
		}

		return nil
	}
}
