package config

// ModelConfigEntry represents the configuration for a single model.
type ModelConfigEntry struct {
	Size int `mapstructure:"size"`
}

// Config holds the application configuration.
type Config struct {
	APIRoot       string                      `mapstructure:"api_root"`
	ListenAddress string                      `mapstructure:"listen_address"`
	Models        map[string]ModelConfigEntry `mapstructure:"models"`
}
