package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"sync"
)

var (
	cfg  *Config
	once sync.Once
)

// LoadConfig reads the config file, parses it, and initializes the global cfg variable.
func LoadConfig(configFile string) (*Config, error) {
	var err error
	once.Do(func() {
		viper.SetConfigFile(configFile)
		viper.SetConfigType("yaml")

		viper.SetDefault("models", []ModelConfigEntry{})
		viper.SetDefault("listen_address", "127.0.0.1:8080")

		err = viper.ReadInConfig()
		if err != nil {
			err = fmt.Errorf("error reading config file: %w", err)
			return
		}

		var configuration Config
		if err = viper.Unmarshal(&configuration); err != nil {
			err = fmt.Errorf("error unmarshaling config: %w", err)
			return
		}

		if configuration.BackendURL == "" {
			err = errors.New("backend_url is required")
			return
		}

		cfg = &configuration
	})

	if err != nil {
		return nil, err
	}

	if cfg == nil {
		return nil, errors.New("configuration was not set")
	}

	return cfg, nil
}

// GetConfig returns the loaded configuration.
func GetConfig() *Config {
	if cfg == nil {
		panic("Config has not been set! Call LoadConfig first.")
	}
	return cfg
}
