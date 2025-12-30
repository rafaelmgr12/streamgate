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

// ServiceConfig defines the configuration for a service.
type ServiceConfig struct {
	Name       string   `yaml:"name"`
	PathPrefix string   `yaml:"path_prefix"`
	Backends   []string `yaml:"backends"`
}

// Config holds all configuration for the application.
type Config struct {
	Transports []TransportConfig `yaml:"transports"`
	Services   []ServiceConfig   `yaml:"services,omitempty"`
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

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

// applyEnvOverrides overrides configuration settings with environment variables if they are set.
func applyEnvOverrides(cfg *Config) {
	overrideAddr(cfg, "http", os.Getenv("STREAMGATE_HTTP_ADDR"))
	overrideAddr(cfg, "grpc", os.Getenv("STREAMGATE_GRPC_ADDR"))
}

// overrideAddr sets the address for a given transport type if the provided addr is not empty.
func overrideAddr(cfg *Config, transportType, addr string) {
	if addr == "" {
		return
	}

	for i := range cfg.Transports {
		if cfg.Transports[i].Type == transportType {
			cfg.Transports[i].Addr = addr
			return
		}
	}

	cfg.Transports = append(cfg.Transports, TransportConfig{
		Type: transportType,
		Addr: addr,
	})
}
