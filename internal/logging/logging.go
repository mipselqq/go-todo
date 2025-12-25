package logging

import (
	"io"
	"log/slog"
	"os"

	"go-todo/internal/config"
)

func NewLoggerBase(loggerConfig config.LoggerConfig) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggerConfig.Level})
	return slog.New(handler)
}

func NewDiscardLogger() *slog.Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	return slog.New(handler)
}
