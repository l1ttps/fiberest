package configs

import (
	"github.com/spf13/viper"
)

// Config holds all application configurations
type Config struct {
	Port string
}

// NewConfig creates a new Config instance using Viper
func NewConfig() (*Config, error) {
	viper.SetDefault("PORT", "3278")
	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	// Optional: Load from .env file if exists
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig() // Ignore error if .env doesn't exist

	config := &Config{
		Port: viper.GetString("PORT"),
	}

	return config, nil
}

// Get retrieves a configuration value by key
func (c *Config) Get(key string) interface{} {
	return viper.Get(key)
}

// GetString retrieves a string configuration value by key
func (c *Config) GetString(key string) string {
	return viper.GetString(key)
}

// GetInt retrieves an integer configuration value by key
func (c *Config) GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool retrieves a boolean configuration value by key
func (c *Config) GetBool(key string) bool {
	return viper.GetBool(key)
}
