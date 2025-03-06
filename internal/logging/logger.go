package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fadedpez/tucoramirez/internal/types"
)

// Level represents a logging level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger represents our custom logger
type Logger struct {
	*log.Logger
	level Level
}

// NewLogger creates a new logger instance
func NewLogger(level Level) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", 0),
		level:  level,
	}
}

// formatMessage formats a log message with timestamp, level, and caller info
func (l *Logger) formatMessage(level Level, msg string) string {
	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	caller := "unknown"
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	return fmt.Sprintf("[%s] %-5s %s: %s",
		timestamp,
		levelNames[level],
		caller,
		msg,
	)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.Output(2, l.formatMessage(DEBUG, fmt.Sprintf(format, v...)))
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.Output(2, l.formatMessage(INFO, fmt.Sprintf(format, v...)))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		l.Output(2, l.formatMessage(WARN, fmt.Sprintf(format, v...)))
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.Output(2, l.formatMessage(ERROR, fmt.Sprintf(format, v...)))
	}
}

// LogError logs a GameError with appropriate context
func (l *Logger) LogError(err error) {
	var gameErr *types.GameError
	if types.As(err, &gameErr) {
		// Format error context
		context := []string{
			fmt.Sprintf("Code: %s", gameErr.Code),
			fmt.Sprintf("Message: %s", gameErr.Message),
		}
		if gameErr.Err != nil {
			context = append(context, fmt.Sprintf("Cause: %v", gameErr.Err))
		}

		l.Error("Game error occurred:\n\t%s", strings.Join(context, "\n\t"))
	} else {
		l.Error("Unexpected error: %v", err)
	}
}

// Default logger instance
var Default = NewLogger(INFO)
