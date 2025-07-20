package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	// DefaultEndpoint is the default ClippingKK GraphQL endpoint
	DefaultEndpoint = "https://clippingkk-api.annatarhe.com/api/v2/graphql"
	// ConfigFileName is the default configuration file name
	ConfigFileName = ".ck-cli.toml"
)

// Config represents the configuration structure
type Config struct {
	HTTP HTTPConfig `toml:"http"`
}

// HTTPConfig represents HTTP configuration
type HTTPConfig struct {
	Endpoint string            `toml:"endpoint"`
	Headers  map[string]string `toml:"headers"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Endpoint: DefaultEndpoint,
			Headers:  make(map[string]string),
		},
	}
}

// UpdateToken adds or updates the authorization token
func (c *Config) UpdateToken(token string) {
	if c.HTTP.Headers == nil {
		c.HTTP.Headers = make(map[string]string)
	}
	c.HTTP.Headers["Authorization"] = fmt.Sprintf("X-CLI %s", token)
}

// HasToken checks if the configuration has an authorization token
func (c *Config) HasToken() bool {
	return c.HTTP.Headers != nil && c.HTTP.Headers["Authorization"] != ""
}

// Save writes the configuration to the specified file path
func (c *Config) Save(path string) error {
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load reads the configuration from the specified file path
func Load(path string) (*Config, error) {
	// If path is empty, use default location
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, ConfigFileName)
	}

	// Handle ~ prefix
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	// If file doesn't exist, create default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		config := NewConfig()
		if err := config.Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	// Read existing config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = toml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure defaults
	if config.HTTP.Endpoint == "" {
		config.HTTP.Endpoint = DefaultEndpoint
	}
	if config.HTTP.Headers == nil {
		config.HTTP.Headers = make(map[string]string)
	}

	return &config, nil
}

// GetConfigPath returns the configuration file path
func GetConfigPath(customPath string) (string, error) {
	if customPath != "" {
		return customPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ConfigFileName), nil
}