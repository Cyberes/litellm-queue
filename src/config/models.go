package config

// ModelConfigEntry represents the configuration for a single model.
type ModelConfigEntry struct {
	Name string `mapstructure:"name"`
	Size int    `mapstructure:"size"`
}

// Config holds the application configuration.
type Config struct {
	BackendURL    string             `mapstructure:"backend_url"`
	ListenAddress string             `mapstructure:"listen_address"`
	Models        []ModelConfigEntry `mapstructure:"models"`
}
