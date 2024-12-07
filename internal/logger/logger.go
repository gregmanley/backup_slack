package logger

import (
	"log"
	"os"
	"path/filepath"
)

var (
	Info  *log.Logger
	Error *log.Logger
	Debug *log.Logger
)

func Init(logPath string) error {
	// Ensure log directory exists
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return err
	}

	// Open log file
	logFile, err := os.OpenFile(
		filepath.Join(logPath, "backup_slack.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	// Initialize loggers
	Info = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(logFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}
