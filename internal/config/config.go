package config

import "os"

// Config holds the application configuration.
type Config struct {
	ListenAddr string
}

// New returns a new Config struct populated from environment variables.
func New() *Config {
	return &Config{
		ListenAddr: getEnv("LISTEN_ADDR", ":8080"),
	}
}

// getEnv retrieves an environment variable or returns a fallback value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
