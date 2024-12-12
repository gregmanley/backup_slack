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

type User struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	FirstSeen   time.Time
}

type File struct {
	ID              string
	MessageID       string
	OriginalURL     string
	LocalPath       string
	FileName        string
	FileType        string
	SizeBytes       int64
	UploadTimestamp time.Time
	Checksum        string
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
	// First verify user exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", msg.UserID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		logger.Error.Printf("Attempted to insert message with non-existent user ID: %s", msg.UserID)
		return fmt.Errorf("user %s does not exist", msg.UserID)
	}

	query := `
        INSERT INTO messages (
            id, channel_id, user_id, content, timestamp, 
            thread_ts, message_type, is_deleted, last_edited
        ) VALUES (?, ?, ?, ?, datetime(?), ?, ?, ?, datetime(?))
        ON CONFLICT(id) DO UPDATE SET
            content = excluded.content,
            is_deleted = excluded.is_deleted,
            last_edited = excluded.last_edited
    `

	lastEdited := sql.NullString{Valid: false}
	if msg.LastEdited.Valid {
		lastEdited.String = msg.LastEdited.Time.Format("2006-01-02 15:04:05")
		lastEdited.Valid = true
	}

	_, err = db.Exec(query,
		msg.ID, msg.ChannelID, msg.UserID, msg.Content,
		msg.Timestamp.Format("2006-01-02 15:04:05"),
		msg.ThreadTS, msg.MessageType,
		msg.IsDeleted, lastEdited,
	)

	return err
}

func (db *DB) GetLastMessageTimestamp(channelID string) (time.Time, error) {
	var unixTime int64
	query := `SELECT COALESCE(MAX(CAST(strftime('%s', timestamp) AS INTEGER)), 0)
              FROM messages 
              WHERE channel_id = ?`

	err := db.DB.QueryRow(query, channelID).Scan(&unixTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last message timestamp: %w", err)
	}

	timestamp := time.Unix(unixTime, 0)
	logger.Debug.Printf("Retrieved timestamp from database: %v (Unix: %d)",
		timestamp, unixTime)

	return timestamp, nil
}

func (db *DB) InsertUser(user User) error {
	query := `
        INSERT INTO users (
            id, username, display_name, avatar_url, first_seen
        ) VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(id) DO NOTHING
    `

	_, err := db.Exec(query,
		user.ID, user.Username, user.DisplayName,
		user.AvatarURL, user.FirstSeen)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (db *DB) MessageExists(messageID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = ?)`

	err := db.QueryRow(query, messageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check message existence: %w", err)
	}

	return exists, nil
}

// InsertFile stores file metadata in the database
func (db *DB) InsertFile(file File) error {
	query := `
		INSERT INTO files (
			id, message_id, original_url, local_path, file_name,
			file_type, size_bytes, upload_timestamp, checksum
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			local_path = excluded.local_path,
			checksum = excluded.checksum
	`

	_, err := db.Exec(query,
		file.ID, file.MessageID, file.OriginalURL, file.LocalPath,
		file.FileName, file.FileType, file.SizeBytes,
		file.UploadTimestamp, file.Checksum)

	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}
	return nil
}

// GetDuplicateFiles returns files with the same checksum
func (db *DB) GetDuplicateFiles(checksum string) ([]File, error) {
	query := `
		SELECT id, message_id, original_url, local_path, file_name,
			   file_type, size_bytes, upload_timestamp, checksum
		FROM files
		WHERE checksum = ?
	`

	rows, err := db.Query(query, checksum)
	if err != nil {
		return nil, fmt.Errorf("failed to query duplicate files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		err := rows.Scan(&f.ID, &f.MessageID, &f.OriginalURL, &f.LocalPath,
			&f.FileName, &f.FileType, &f.SizeBytes, &f.UploadTimestamp, &f.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file row: %w", err)
		}
		files = append(files, f)
	}

	return files, nil
}

// GetOrphanedFiles returns files that don't have an associated message
func (db *DB) GetOrphanedFiles() ([]File, error) {
	query := `
		SELECT f.id, f.message_id, f.original_url, f.local_path, f.file_name,
			   f.file_type, f.size_bytes, f.upload_timestamp, f.checksum
		FROM files f
		LEFT JOIN messages m ON f.message_id = m.id
		WHERE m.id IS NULL
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query orphaned files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		err := rows.Scan(&f.ID, &f.MessageID, &f.OriginalURL, &f.LocalPath,
			&f.FileName, &f.FileType, &f.SizeBytes, &f.UploadTimestamp, &f.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file row: %w", err)
		}
		files = append(files, f)
	}

	return files, nil
}

// DeleteFile removes a file record from the database
func (db *DB) DeleteFile(fileID string) error {
	query := `DELETE FROM files WHERE id = ?`

	result, err := db.Exec(query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no file found with ID %s", fileID)
	}

	return nil
}
