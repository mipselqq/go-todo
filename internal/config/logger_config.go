package config

import (
	"log/slog"
	"os"
)

type LoggerConfig struct {
	Level slog.Level
}

func NewLoggerConfigFromEnv() LoggerConfig {
	envLevel := os.Getenv("LOG_LEVEL")

	var level slog.Level

	switch envLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
		slog.Warn("Unknown log level, using default", "env:LOG_LEVEL", envLevel, "level", level)
	}

	return LoggerConfig{
		Level: level,
	}
}
