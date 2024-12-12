package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"backup_slack/internal/logger"
)

const maxRetries = 3
const retryDelay = 5 * time.Second

// Downloader handles file downloads with retry logic
type Downloader struct {
	storage *FileStorage
	client  *http.Client
}

// NewDownloader creates a new file downloader
func NewDownloader(storage *FileStorage) *Downloader {
	return &Downloader{
		storage: storage,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DownloadFile downloads a file from Slack and stores it locally
func (d *Downloader) DownloadFile(metadata FileMetadata, token string) (string, error) {
	// Check disk space before download
	if err := d.storage.CheckDiskSpace(metadata.SizeBytes); err != nil {
		return "", fmt.Errorf("disk space check failed: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("GET", metadata.OriginalURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Add("Authorization", "Bearer "+token)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Debug.Printf("Download attempt %d for file %s", attempt, metadata.FileName)

		if err := d.downloadWithRetry(req, metadata); err != nil {
			lastErr = err
			logger.Warn.Printf("Download attempt %d failed for %s: %v", attempt, metadata.FileName, err)
			time.Sleep(retryDelay * time.Duration(attempt))
			continue
		}

		// Calculate checksum after successful download
		checksum, err := CalculateChecksum(metadata.LocalPath)
		if err != nil {
			lastErr = fmt.Errorf("failed to calculate checksum: %w", err)
			continue
		}

		return checksum, nil
	}

	return "", fmt.Errorf("failed to download file after %d attempts: %w", maxRetries, lastErr)
}

// downloadWithRetry performs a single download attempt
func (d *Downloader) downloadWithRetry(req *http.Request, metadata FileMetadata) error {
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile := metadata.LocalPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer out.Close()

	// Copy the response body to the temporary file
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Move temporary file to final location
	if err := os.Rename(tmpFile, metadata.LocalPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	return nil
}

// verifyChecksum calculates and verifies the SHA-256 checksum of a file
func (d *Downloader) verifyChecksum(filePath, expectedChecksum string) (bool, error) {
	logger.Debug.Printf("Verifying checksum for %s (expected: %s)", filePath, expectedChecksum)

	actualChecksum, err := CalculateChecksum(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	logger.Debug.Printf("Checksum comparison - Expected: %s, Actual: %s", expectedChecksum, actualChecksum)

	return actualChecksum == expectedChecksum, nil
}

// CalculateChecksum generates a SHA-256 checksum for a file
func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (d *Downloader) DownloadFileWithRetry(url, destPath, expectedChecksum string, metadata FileMetadata) error {
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Debug.Printf("Download attempt %d/%d for %s", attempt, maxRetries, url)

		if _, err := d.DownloadFile(metadata, ""); err != nil {
			logger.Debug.Printf("Download attempt %d failed: %v", attempt, err)
			if attempt == maxRetries {
				return fmt.Errorf("failed to download file after %d attempts: %w", maxRetries, err)
			}
			// Exponential backoff
			time.Sleep(time.Second * time.Duration(1<<uint(attempt)))
			continue
		}

		// Verify checksum
		match, err := d.verifyChecksum(destPath, expectedChecksum)
		if err != nil {
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		if !match {
			if attempt == maxRetries {
				return fmt.Errorf("checksum mismatch after %d attempts", maxRetries)
			}
			logger.Debug.Printf("Checksum mismatch on attempt %d, retrying...", attempt)
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to download and verify file after %d attempts", maxRetries)
}

func cleanup(path string) {
	if err := os.Remove(path); err != nil {
		logger.Debug.Printf("Failed to cleanup file %s: %v", path, err)
	}
}
