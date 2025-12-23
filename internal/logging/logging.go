package logging

import (
	"io"
	"log/slog"
	"os"
)

func NewLoggerBase() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func NewDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
