package database

import (
	"database/sql"
	"fmt"
)

type Migration struct {
	Version int
	SQL     string
}

var migrations = []Migration{
	{
		Version: 1,
		SQL: `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			display_name TEXT,
			avatar_url TEXT,
			first_seen DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			channel_type TEXT NOT NULL,
			is_archived BOOLEAN DEFAULT FALSE,
			created_at DATETIME NOT NULL,
			topic TEXT,
			purpose TEXT
		);

		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			channel_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			content TEXT,
			timestamp DATETIME NOT NULL,
			thread_ts TEXT,
			message_type TEXT NOT NULL,
			is_deleted BOOLEAN DEFAULT FALSE,
			last_edited DATETIME,
			FOREIGN KEY (channel_id) REFERENCES channels(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS reactions (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			emoji TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			PRIMARY KEY (message_id, user_id, emoji),
			FOREIGN KEY (message_id) REFERENCES messages(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS files (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			original_url TEXT NOT NULL,
			local_path TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_type TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			upload_timestamp DATETIME NOT NULL,
			checksum TEXT NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id)
		);`,
	},
}

func applyMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Apply each migration in transaction
	for _, migration := range migrations {
		var version int
		err := db.QueryRow("SELECT version FROM schema_migrations WHERE version = ?", migration.Version).Scan(&version)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to check migration version: %w", err)
		}
		if err == nil {
			continue // Migration already applied
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, datetime('now'))", migration.Version); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}
	}

	return nil
}
