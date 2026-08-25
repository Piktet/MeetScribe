// Package logger — обёртка над log/slog для структурированного логирования.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// Error logs an error-level message with the provided error.
func Error(err error, msg string, args ...any) {
	if err != nil {
		defaultLogger.Error(msg, "err", err)
	} else {
		defaultLogger.Error(msg, args...)
	}
}

// Warn logs a warning-level message.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Info logs an info-level message.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Debug logs a debug-level message.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// With returns a new logger that includes the given key-value pairs in every log entry.
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

// levelFromString преобразует строковый уровень в slog.Level.
func levelFromString(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// InitLogger устанавливает уровень логирования.
// level — текстовый уровень: "DEBUG", "INFO", "WARN", "ERROR".
func InitLogger(level string) error {
	lvl := levelFromString(level)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})
	defaultLogger = slog.New(handler)
	return nil
}
