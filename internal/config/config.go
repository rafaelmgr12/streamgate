// Package config provides configuration management for the application.
// It loads settings from environment variables and provides sensible defaults.
package config

import "os"

// Config holds all configuration for the application.
// Each field can be populated from environment variables.
type Config struct {
	// ListenAddr is the network address the server will listen on (e.g., ":8080").
	ListenAddr string
}

// New creates a new Config instance, populating it with values from environment
// variables. It provides default values for any settings that are not specified.
func New() *Config {
	return &Config{
		ListenAddr: getEnv("LISTEN_ADDR", ":8080"),
	}
}

// getEnv retrieves the value of an environment variable by its key.
// If the variable is not set, it returns the provided fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
