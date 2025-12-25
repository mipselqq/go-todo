package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestLoggerConfig(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected slog.Level
	}{
		{"DebugLevel", "DEBUG", slog.LevelDebug},
		{"InfoLevel", "INFO", slog.LevelInfo},
		{"WarnLevel", "WARN", slog.LevelWarn},
		{"ErrorLevel", "ERROR", slog.LevelError},
		{"UnknownLevel", "UNKNOWN", slog.LevelInfo},
		{"EmptyLevel", "", slog.LevelInfo},
		{"LowercaseLevel", "debug", slog.LevelInfo},
		{"NoEnv", "", slog.LevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "NoEnv" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tc.envValue)
				defer os.Unsetenv("LOG_LEVEL")
			}

			config := NewLoggerConfigFromEnv()
			if config.Level != tc.expected {
				t.Errorf("expected level %v, got %v", tc.expected, config.Level)
			}
		})
	}
}
