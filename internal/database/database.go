package database

import (
	"backup_slack/internal/logger"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

type Channel struct {
	ID          string
	Name        string
	ChannelType string
	IsArchived  bool
	CreatedAt   time.Time
	Topic       string
	Purpose     string
}

// New creates a new database connection and ensures schema is up to date
func New(dbPath string) (*DB, error) {
	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Apply migrations
	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) InsertChannel(ch Channel) error {
	query := `
        INSERT INTO channels (
            id, name, channel_type, is_archived, created_at, topic, purpose
        ) VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name = excluded.name,
            channel_type = excluded.channel_type,
            is_archived = excluded.is_archived,
            topic = excluded.topic,
            purpose = excluded.purpose
    `
	result, err := db.DB.Exec(query,
		ch.ID, ch.Name, ch.ChannelType, ch.IsArchived, ch.CreatedAt, ch.Topic, ch.Purpose)
	if err != nil {
		logger.Error.Printf("Database error upserting channel %s: %v", ch.Name, err)
		return err
	}

	rows, _ := result.RowsAffected()
	logger.Debug.Printf("Upserted channel %s (rows affected: %d)", ch.Name, rows)
	return nil
}
