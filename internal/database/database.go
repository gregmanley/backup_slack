package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"backup_slack/internal/logger"
	"database/sql"

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

type Message struct {
	ID          string
	ChannelID   string
	UserID      string
	Content     string
	Timestamp   time.Time
	ThreadTS    sql.NullString
	MessageType string
	IsDeleted   bool
	LastEdited  sql.NullTime
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

func (db *DB) InsertMessage(msg Message) error {
	query := `
		INSERT INTO messages (
			id, channel_id, user_id, content, timestamp, 
			thread_ts, message_type, is_deleted, last_edited
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			is_deleted = excluded.is_deleted,
			last_edited = excluded.last_edited
	`

	_, err := db.Exec(query,
		msg.ID, msg.ChannelID, msg.UserID, msg.Content,
		msg.Timestamp, msg.ThreadTS, msg.MessageType,
		msg.IsDeleted, msg.LastEdited,
	)

	return err
}

func (db *DB) GetLastMessageTimestamp(channelID string) (time.Time, error) {
	var timestamp time.Time
	query := `SELECT COALESCE(MAX(timestamp), datetime('1970-01-01T00:00:00Z')) 
              FROM messages 
              WHERE channel_id = ?`

	err := db.DB.QueryRow(query, channelID).Scan(&timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last message timestamp: %w", err)
	}

	return timestamp, nil
}
