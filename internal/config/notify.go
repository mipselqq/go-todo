package config

import "log/slog"

type Notify struct {
	LogLevel string
	Env      string
}

func NewNotifyFromEnv(logger *slog.Logger) Notify {
	return Notify{
		LogLevel: getEnvStringOrDefault("NOTIFY_LOG_LEVEL", "info", logger),
		Env:      getEnvStringOrDefault("NOTIFY_ENV", "dev", logger),
	}
}

//nolint:gocritic // Pointer receiver disables formatting
func (c Notify) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("log_level", c.LogLevel),
		slog.String("env", c.Env),
	)
}
