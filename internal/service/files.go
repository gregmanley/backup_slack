package service

import (
	"fmt"

	"backup_slack/internal/database"
	"backup_slack/internal/files"
	"backup_slack/internal/logger"
)

type FileService struct {
	storage    *files.FileStorage
	downloader *files.Downloader
	db         *database.DB
	token      string
}

func NewFileService(basePath string, maxFileSize int64, db *database.DB, token string) (*FileService, error) {
	storage, err := files.NewFileStorage(basePath, maxFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	return &FileService{
		storage:    storage,
		downloader: files.NewDownloader(storage),
		db:         db,
		token:      token,
	}, nil
}

// ProcessFile handles the complete file processing pipeline
func (s *FileService) ProcessFile(slackFile database.File) error {
	// Check for duplicates first
	duplicates, err := s.db.GetDuplicateFiles(slackFile.Checksum)
	if err != nil {
		return fmt.Errorf("failed to check duplicates: %w", err)
	}

	if len(duplicates) > 0 && slackFile.Checksum != "" {
		// Use existing file via hard link
		logger.Info.Printf("Found duplicate for file %s, creating hard link", slackFile.FileName)
		err := s.storage.HandleDuplicate(duplicates[0].LocalPath, slackFile.LocalPath)
		if err != nil {
			return fmt.Errorf("failed to handle duplicate: %w", err)
		}
	} else {
		// Download new file
		metadata := files.FileMetadata{
			ID:              slackFile.ID,
			MessageID:       slackFile.MessageID,
			OriginalURL:     slackFile.OriginalURL,
			LocalPath:       slackFile.LocalPath,
			FileName:        slackFile.FileName,
			FileType:        slackFile.FileType,
			SizeBytes:       slackFile.SizeBytes,
			UploadTimestamp: slackFile.UploadTimestamp,
		}

		checksum, err := s.downloader.DownloadFile(metadata, s.token)
		if err != nil {
			return fmt.Errorf("failed to download file: %w", err)
		}

		// Update file metadata with calculated checksum
		slackFile.Checksum = checksum
	}

	// Store file metadata in database
	if err := s.db.InsertFile(slackFile); err != nil {
		return fmt.Errorf("failed to store file metadata: %w", err)
	}

	return nil
}

// CleanupOrphanedFiles removes files without associated messages
func (s *FileService) CleanupOrphanedFiles() error {
	orphans, err := s.db.GetOrphanedFiles()
	if err != nil {
		return fmt.Errorf("failed to get orphaned files: %w", err)
	}

	for _, file := range orphans {
		logger.Info.Printf("Cleaning up orphaned file: %s", file.FileName)
		if err := s.storage.CleanupOrphanedFile(file.LocalPath); err != nil {
			logger.Error.Printf("Failed to cleanup file %s: %v", file.FileName, err)
			continue
		}

		// Remove the file record from database after successful filesystem cleanup
		if err := s.db.DeleteFile(file.ID); err != nil {
			logger.Error.Printf("Failed to remove file record from database for %s: %v", file.FileName, err)
			continue
		}
		logger.Debug.Printf("Successfully cleaned up orphaned file %s", file.FileName)
	}

	return nil
}
