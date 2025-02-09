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
	LogLevel      string
	Environment   string
	LogDir        string // New field for explicit log directory
}

// Load returns a Config struct populated with current configuration
func Load() (*Config, error) {
	c := &Config{}

	var missingVars []string

	// Required variables
	c.SlackAPIToken = getEnvOrDefault("SLACK_BOT_TOKEN", "")
	if c.SlackAPIToken == "" {
		missingVars = append(missingVars, "SLACK_BOT_TOKEN")
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
	c.LogLevel = getEnvOrDefault("LOG_LEVEL", "INFO")

	// Environment with default
	c.Environment = getEnvOrDefault("ENVIRONMENT", "development")

	// Get explicit log directory if specified
	c.LogDir = getEnvOrDefault("LOG_DIR", "")

	// If LogDir not explicitly set, use default based on environment
	if c.LogDir == "" {
		if c.Environment == "production" {
			c.LogDir = "/var/log/backup_slack"
		} else {
			c.LogDir = "./logs"
		}
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missingVars, ", "))
	}

	return c, nil
}
