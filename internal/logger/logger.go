package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelError LogLevel = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

var (
	Info  *log.Logger
	Error *log.Logger
	Debug *log.Logger
	Warn  *log.Logger
	level LogLevel
)

func ParseLogLevel(lvl string) LogLevel {
	switch strings.ToUpper(lvl) {
	case "DEBUG":
		return LevelDebug
	case "ERROR":
		return LevelError
	case "WARN":
		return LevelWarn
	default:
		return LevelInfo
	}
}

type nullWriter struct{}

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func Init(logPath string, logLevel LogLevel) error {
	level = logLevel

	if err := os.MkdirAll(logPath, 0755); err != nil {
		return err
	}

	// Open log file with 0644 permissions
	logFile, err := os.OpenFile(
		filepath.Join(logPath, "backup_slack.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return err
	}

	// Always enable Error logging
	Error = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Configure Warn logger based on level
	warnWriter := io.Writer(&nullWriter{})
	if level >= LevelWarn {
		warnWriter = logFile
	}
	Warn = log.New(warnWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Configure Info logger based on level
	infoWriter := io.Writer(&nullWriter{})
	if level >= LevelInfo {
		infoWriter = logFile
	}
	Info = log.New(infoWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Configure Debug logger based on level
	debugWriter := io.Writer(&nullWriter{})
	if level >= LevelDebug {
		debugWriter = logFile
	}
	Debug = log.New(debugWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// Add rate limit specific logging
func LogRateLimit(retryAfter time.Duration, operation string) {
	Warn.Printf("Rate limit hit during %s, waiting %v seconds",
		operation, retryAfter.Seconds())
}

func LogRetryAttempt(attempt, maxRetries int, operation string) {
	Debug.Printf("Retry attempt %d/%d for operation: %s",
		attempt+1, maxRetries, operation)
}
