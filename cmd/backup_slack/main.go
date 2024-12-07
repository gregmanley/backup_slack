package main

import (
	"log"
	"os"
	"path/filepath"

	"backup_slack/internal/config"
	"backup_slack/internal/database"
	"backup_slack/internal/logger"
	"backup_slack/internal/service"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}
}

func main() {
	// Create data directories
	dirs := []string{"./data", "./data/logs", "./data/storage"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger with log level
	if err := logger.Init(cfg.LogPath, logger.ParseLogLevel(cfg.LogLevel)); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		logger.Error.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := database.New(cfg.DBPath)
	if err != nil {
		logger.Error.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	logger.Info.Println("Database initialized successfully")

	// Initialize Slack service
	slackService := service.NewSlackService(cfg.SlackAPIToken, db)
	if err := slackService.Initialize(cfg.SlackChannels); err != nil {
		logger.Error.Fatalf("Failed to initialize Slack service: %v", err)
	}

	logger.Info.Println("Slack service initialized successfully")
	logger.Info.Printf("Configured to backup %d channels: %v", len(cfg.SlackChannels), cfg.SlackChannels)
}
