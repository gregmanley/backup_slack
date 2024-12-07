package config

import (
	"fmt"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	SlackAPIToken string
	SlackChannels []string
	DBPath        string
	StoragePath   string
	LogPath       string
	MaxRetries    int
	BatchSize     int
}

// Load returns a Config struct populated with current configuration
func Load() (*Config, error) {
	c := &Config{}

	var missingVars []string

	// Required variables
	c.SlackAPIToken = getEnvOrDefault("SLACK_APP_TOKEN", "")
	if c.SlackAPIToken == "" {
		missingVars = append(missingVars, "SLACK_APP_TOKEN")
	}

	channels := getEnvOrDefault("SLACK_CHANNELS", "")
	if channels == "" {
		missingVars = append(missingVars, "SLACK_CHANNELS")
	}
	c.SlackChannels = strings.Split(channels, ",")

	c.DBPath = getEnvOrDefault("DB_PATH", "")
	if c.DBPath == "" {
		missingVars = append(missingVars, "DB_PATH")
	}

	c.StoragePath = getEnvOrDefault("STORAGE_PATH", "")
	if c.StoragePath == "" {
		missingVars = append(missingVars, "STORAGE_PATH")
	}

	c.LogPath = getEnvOrDefault("LOG_PATH", "")
	if c.LogPath == "" {
		missingVars = append(missingVars, "LOG_PATH")
	}

	// Optional variables with defaults
	c.MaxRetries = getEnvAsIntOrDefault("MAX_RETRIES", 3)
	c.BatchSize = getEnvAsIntOrDefault("BATCH_SIZE", 100)

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	return c, nil
}
