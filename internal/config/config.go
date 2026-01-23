package config

import "github.com/kelseyhightower/envconfig"

// ServiceConfig holds service-level configuration
type ServiceConfig struct {
	BindAddress string `envconfig:"BIND_ADDRESS" default:"0.0.0.0:8080"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
}

// Config is the root configuration structure
type Config struct {
	Service ServiceConfig
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}
	if err := envconfig.Process("", &cfg.Service); err != nil {
		return nil, err
	}
	return cfg, nil
}
