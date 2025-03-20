package options

import (
	"log/slog"
	"os"

	"github.com/robbyt/go-polyscript/execution/data"
	"github.com/robbyt/go-polyscript/machines/types"
)

// DefaultConfig initializes a Config with sensible defaults
func DefaultConfig(machineType types.Type) *Config {
	cfg := &Config{}
	cfg.SetMachineType(machineType)
	cfg.SetHandler(DefaultHandler())
	cfg.SetDataProvider(DefaultDataProvider())
	return cfg
}

// DefaultHandler returns the default logging handler
func DefaultHandler() slog.Handler {
	return slog.NewTextHandler(os.Stdout, nil)
}

// DefaultDataProvider returns the default data provider
func DefaultDataProvider() data.InputDataProvider {
	return data.NewStaticProvider(map[string]any{})
}

// WithDefaults applies default values to any config properties that are nil
func WithDefaults() Option {
	return func(c *Config) error {
		if c.handler == nil {
			c.handler = DefaultHandler()
		}

		if c.dataProvider == nil {
			c.dataProvider = DefaultDataProvider()
		}

		return nil
	}
}
