package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test required variables
	requiredVars := map[string]string{
		"SLACK_BOT_TOKEN": "xoxb-test-token",
		"SLACK_CHANNELS":  "channel1,channel2",
		"DB_PATH":         "/tmp/db",
		"STORAGE_PATH":    "/tmp/storage",
		"LOG_PATH":        "/tmp/logs",
	}

	// Set environment variables
	for k, v := range requiredVars {
		os.Setenv(k, v)
	}
	defer func() {
		// Clean up environment variables
		for k := range requiredVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cfg.SlackAPIToken != requiredVars["SLACK_BOT_TOKEN"] {
		t.Errorf("Expected token %s, got %s", requiredVars["SLACK_BOT_TOKEN"], cfg.SlackAPIToken)
	}

	if len(cfg.SlackChannels) != 2 {
		t.Errorf("Expected 2 channels, got %d", len(cfg.SlackChannels))
	}

	// Test missing required variable
	os.Unsetenv("SLACK_BOT_TOKEN")
	_, err = Load()
	if err == nil {
		t.Error("Expected error for missing SLACK_BOT_TOKEN, got nil")
	}
}
