// Package config provides configuration management for the application.
// It loads settings from environment variables and provides sensible defaults.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// TransportConfig defines the configuration for a single network transport.
type TransportConfig struct {
	Type string `yaml:"type"`
	Addr string `yaml:"addr"`
}

// Config holds all configuration for the application.
type Config struct {
	Transports []TransportConfig `yaml:"transports"`
}

// Load reads a configuration file from the given path, unmarshals it into a
// Config struct, and returns it.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
