package config

import "github.com/kelseyhightower/envconfig"

// ServiceConfig holds service-level configuration
type ServiceConfig struct {
	BindAddress string `envconfig:"BIND_ADDRESS" default:"0.0.0.0:8080"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
}

// DBConfig holds database configuration
type DBConfig struct {
	Type     string `envconfig:"DB_TYPE" default:"sqlite"`
	Name     string `envconfig:"DB_NAME" default:"policy_manager.db"`
	Hostname string `envconfig:"DB_HOSTNAME" default:"localhost"`
	Port     string `envconfig:"DB_PORT" default:"5432"`
	User     string `envconfig:"DB_USER" default:"postgres"`
	Password string `envconfig:"DB_PASSWORD" default:""`
}

// Config is the root configuration structure
type Config struct {
	Service  ServiceConfig
	Database *DBConfig
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Database: &DBConfig{},
	}
	if err := envconfig.Process("", &cfg.Service); err != nil {
		return nil, err
	}
	if err := envconfig.Process("", cfg.Database); err != nil {
		return nil, err
	}
	return cfg, nil
}
