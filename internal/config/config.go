// Package config provides configuration management for the application.
// It loads settings from environment variables and provides sensible defaults.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TransportConfig defines the configuration for a single network transport.
type TransportConfig struct {
	Type string `yaml:"type"`
	Addr string `yaml:"addr"`
}

// ServiceConfig defines the configuration for a service.
type ServiceConfig struct {
	Name          string   `yaml:"name"`
	PathPrefix    string   `yaml:"path_prefix"`
	Backends      []string `yaml:"backends"`
	LoadBalancing string   `yaml:"load_balancing,omitempty"`
	HealthCheck   HealthCheckConfig `yaml:"health_check,omitempty"`
}

// HealthCheckConfig defines active health check settings.
type HealthCheckConfig struct {
	Path     string        `yaml:"path,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty"`
	Timeout  time.Duration `yaml:"timeout,omitempty"`
}

// Config holds all configuration for the application.
type Config struct {
	Transports []TransportConfig `yaml:"transports"`
	Services   []ServiceConfig   `yaml:"services,omitempty"`
	HealthCheck HealthCheckConfig `yaml:"health_check,omitempty"`
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

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

func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if len(cfg.Transports) == 0 {
		return fmt.Errorf("at least one transport is required")
	}

	for _, transport := range cfg.Transports {

		tType := strings.TrimSpace(transport.Type)

		if tType == "" {
			return fmt.Errorf("transport type is required")
		}
		if strings.TrimSpace(transport.Addr) == "" {
			return fmt.Errorf("transport %q has empty addr", tType)
		}

	}

	for _, svc := range cfg.Services {

		if strings.TrimSpace(svc.Name) == "" {
			return fmt.Errorf("service name is required")
		}

		p := strings.TrimSpace(svc.PathPrefix)
		if p == "" {
			return fmt.Errorf("service %q has empty path_prefix", svc.Name)
		}

		if !strings.HasPrefix(p, "/") {
			return fmt.Errorf("service %q path_prefix must start with '/'", svc.Name)
		}

		if len(svc.Backends) == 0 {
			return fmt.Errorf("service %q must have at least one backend", svc.Name)
		}

		for _, raw := range svc.Backends {
			u, err := parseBackendURLForValidation(raw)
			if err != nil || u.Host == "" {
				return fmt.Errorf("service %q has invalid backend %q: %w", svc.Name, raw, err)
			}
		}

		if err := validateHealthCheck(svc.HealthCheck); err != nil {
			return fmt.Errorf("service %q health_check invalid: %w", svc.Name, err)
		}
	}

	if err := validateHealthCheck(cfg.HealthCheck); err != nil {
		return fmt.Errorf("health_check invalid: %w", err)
	}

	return nil

}

func validateHealthCheck(h HealthCheckConfig) error {
	path := strings.TrimSpace(h.Path)
	if path != "" && !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with '/'")
	}
	if h.Interval < 0 {
		return fmt.Errorf("interval must be >= 0")
	}
	if h.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}
	return nil
}

func parseBackendURLForValidation(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("backend URL is empty")
	}

	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	return url.Parse(raw)
}
