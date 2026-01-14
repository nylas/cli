// Package common provides shared utilities for CLI commands.
package common

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	logger     *slog.Logger
	loggerOnce sync.Once
	debugMode  bool
	quietMode  bool
)

// LogLevel represents logging levels.
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// InitLogger initializes the global logger with the specified options.
func InitLogger(debug, quiet bool) {
	loggerOnce.Do(func() {
		debugMode = debug
		quietMode = quiet

		var level slog.Level
		if debug {
			level = slog.LevelDebug
		} else {
			level = slog.LevelInfo
		}

		var output io.Writer = os.Stderr
		if quiet {
			output = io.Discard
		}

		opts := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Remove time from output for cleaner CLI display
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}

		handler := slog.NewTextHandler(output, opts)
		logger = slog.New(handler)
	})
}

// ResetLogger resets the logger (for testing).
func ResetLogger() {
	loggerOnce = sync.Once{}
	logger = nil
	debugMode = false
	quietMode = false
}

// GetLogger returns the global logger, initializing with defaults if needed.
func GetLogger() *slog.Logger {
	if logger == nil {
		InitLogger(false, false)
	}
	return logger
}

// IsDebug returns true if debug mode is enabled.
func IsDebug() bool {
	return debugMode
}

// IsQuiet returns true if quiet mode is enabled.
func IsQuiet() bool {
	return quietMode
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// DebugHTTP logs HTTP request/response details in debug mode.
func DebugHTTP(method, url string, statusCode int, duration string) {
	if debugMode {
		Debug("HTTP request",
			"method", method,
			"url", url,
			"status", statusCode,
			"duration", duration,
		)
	}
}

// DebugAPI logs API operation details in debug mode.
func DebugAPI(operation string, args ...any) {
	if debugMode {
		allArgs := append([]any{"operation", operation}, args...)
		Debug("API call", allArgs...)
	}
}
