package config

import "os"

// Config holds all configuration for the application
type Config struct {
	AppName string
	// Add more configuration fields as needed
}

// Load returns a Config struct populated with current configuration
func Load() *Config {
	return &Config{
		AppName: getEnvOrDefault("APP_NAME", "myapp"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
