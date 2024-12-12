package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileStorage(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "file_storage_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		basePath    string
		maxFileSize int64
		wantErr     bool
	}{
		{
			name:        "Valid storage creation",
			basePath:    tmpDir,
			maxFileSize: 1024 * 1024 * 10, // 10MB
			wantErr:     false,
		},
		{
			name:        "Invalid path",
			basePath:    "/nonexistent/path/that/should/fail",
			maxFileSize: 1024 * 1024,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewFileStorage(tt.basePath, tt.maxFileSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && storage == nil {
				t.Error("NewFileStorage() returned nil storage without error")
			}
			if !tt.wantErr {
				// Verify directories were created
				paths := storage.GetStoragePaths()
				for _, path := range []string{paths.Images, paths.Files} {
					if _, err := os.Stat(path); os.IsNotExist(err) {
						t.Errorf("Expected directory not created: %s", path)
					}
				}
			}
		})
	}
}

func TestGenerateFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_path_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir, 1024*1024)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	tests := []struct {
		name      string
		channelID string
		fileID    string
		fileType  string
		timestamp time.Time
		wantDir   string
	}{
		{
			name:      "Image file path",
			channelID: "C123456",
			fileID:    "F789012",
			fileType:  "jpg",
			timestamp: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC),
			wantDir:   "images",
		},
		{
			name:      "Document file path",
			channelID: "C123456",
			fileID:    "F789012",
			fileType:  "pdf",
			timestamp: time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC),
			wantDir:   "files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := storage.GenerateFilePath(tt.channelID, tt.fileID, tt.fileType, tt.timestamp)

			// Check if path contains expected components
			if !filepath.IsAbs(path) {
				path = filepath.Join(tmpDir, path)
			}

			// Verify path structure
			expected := filepath.Join(
				tmpDir,
				tt.wantDir,
				tt.channelID,
				tt.timestamp.Format("2006-01"),
				tt.fileID+"."+tt.fileType,
			)

			if path != expected {
				t.Errorf("GenerateFilePath() got = %v, want %v", path, expected)
			}

			// Verify directory was created
			dir := filepath.Dir(path)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("Expected directory not created: %s", dir)
			}
		})
	}
}

func TestIsImageType(t *testing.T) {
	tests := []struct {
		fileType string
		want     bool
	}{
		{"jpg", true},
		{"jpeg", true},
		{"png", true},
		{"gif", true},
		{"pdf", false},
		{"doc", false},
		{"txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			if got := isImageType(tt.fileType); got != tt.want {
				t.Errorf("isImageType(%q) = %v, want %v", tt.fileType, got, tt.want)
			}
		})
	}
}
