package config

// ConfigServer defines the interface for reading configuration from different formats (YAML, TOML, JSON).
type ConfigServer interface {
	// ReadConfig reads and parses the configuration from a given file path.
	// It should detect or be provided with the correct format to parse (YAML, TOML, JSON).
	ReadConfig(filePath string) (map[string]interface{}, error)
}
