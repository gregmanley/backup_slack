package main

import (
	"log"

	"backup_slack/internal/config"
	"backup_slack/internal/logger"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	if err := logger.Init(cfg.LogPath); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info.Println("Slack backup system starting...")
	logger.Info.Printf("Configured to backup %d channels: %v", len(cfg.SlackChannels), cfg.SlackChannels)
}
