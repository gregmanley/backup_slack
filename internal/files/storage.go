package files

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"backup_slack/internal/logger"
)

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(basePath string, maxFileSize int64) (*FileStorage, error) {
	storage := &FileStorage{
		BasePath:    basePath,
		MaxFileSize: maxFileSize,
	}

	// Create base directory structure
	paths := storage.GetStoragePaths()
	for _, path := range []string{paths.Images, paths.Files} {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return storage, nil
}

// GetStoragePaths returns the configured paths for different file types
func (fs *FileStorage) GetStoragePaths() StoragePaths {
	return StoragePaths{
		Images: filepath.Join(fs.BasePath, "images"),
		Files:  filepath.Join(fs.BasePath, "files"),
	}
}

// GenerateFilePath creates the appropriate path for a file
func (fs *FileStorage) GenerateFilePath(channelID, fileID, fileType string, timestamp time.Time) string {
	// Determine base directory based on file type
	var baseDir string
	if isImageType(fileType) {
		baseDir = filepath.Join(fs.GetStoragePaths().Images, channelID)
	} else {
		baseDir = filepath.Join(fs.GetStoragePaths().Files, channelID)
	}

	// Create year-month subdirectory
	yearMonth := timestamp.Format("2006-01")
	dirPath := filepath.Join(baseDir, yearMonth)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		logger.Error.Printf("Failed to create directory %s: %v", dirPath, err)
		return ""
	}

	return filepath.Join(dirPath, fmt.Sprintf("%s.%s", fileID, fileType))
}

// CheckDiskSpace verifies if there's enough space for a file
func (fs *FileStorage) CheckDiskSpace(requiredBytes int64) error {
	var stat syscall.Statfs_t
	err := syscall.Statfs(fs.BasePath, &stat)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Available bytes = blocks * size
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	if uint64(requiredBytes) > availableBytes {
		return fmt.Errorf("insufficient disk space. Required: %d bytes, Available: %d bytes",
			requiredBytes, availableBytes)
	}

	return nil
}

// FileExists checks if a file exists at the given path
func (fs *FileStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isImageType checks if the file type is an image
func isImageType(fileType string) bool {
	imageTypes := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"gif":  true,
		"webp": true,
		"tiff": true,
		"bmp":  true,
	}
	return imageTypes[fileType]
}
