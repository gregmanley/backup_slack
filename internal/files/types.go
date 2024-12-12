package files

import (
	"time"
)

// FileMetadata represents information about a downloaded Slack file
type FileMetadata struct {
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

// FileStorage configuration and paths
type FileStorage struct {
	BasePath    string
	MaxFileSize int64 // maximum file size in bytes
}

// StoragePaths holds the configured paths for different file types
type StoragePaths struct {
	Images string
	Files  string
}

// ValidationResult represents the outcome of a file validation check
type ValidationResult struct {
	IsValid  bool
	Checksum string
	Error    error
}

// FileTypeInfo contains metadata about supported file types
type FileTypeInfo struct {
	MaxSize    int64
	AllowedExt []string
}
