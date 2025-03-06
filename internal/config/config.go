package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Discord configuration
	Token     string
	AppID     string
	GuildID   string

	// Resource paths
	DataDir    string
	ImagesPath string
	QuotesPath string

	// Environment
	Environment string // "development" or "production"
}

// Load reads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Only return error if file exists but couldn't be loaded
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	// Get working directory for resource paths
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	cfg := &Config{
		Token:       os.Getenv("DISCORD_TOKEN"),
		AppID:       os.Getenv("APP_ID"),
		GuildID:     os.Getenv("GUILD_ID"),
		Environment: getEnvWithDefault("ENVIRONMENT", "development"),
		DataDir:     getEnvWithDefault("DATA_DIR", filepath.Join(wd, "data")),
		ImagesPath:  filepath.Join(wd, "images.txt"),
		QuotesPath:  filepath.Join(wd, "quotes.txt"),
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return cfg, nil
}

// validate checks if all required configuration is present
func (c *Config) validate() error {
	if c.Token == "" {
		return fmt.Errorf("DISCORD_TOKEN is required")
	}
	if c.AppID == "" {
		return fmt.Errorf("APP_ID is required")
	}
	if c.GuildID == "" {
		return fmt.Errorf("GUILD_ID is required")
	}
	return nil
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
