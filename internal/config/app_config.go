package config

import (
	"log/slog"
	"os"
)

type AppConfig struct {
	Port string
	Host string
	Addr string
}

func NewAppConfigFromEnv(loggerBase *slog.Logger) *AppConfig {
	logger := loggerBase.With("scope", "app_config")

	envHost := os.Getenv("APP_HOST")

	var host string

	if envHost == "" {
		host = "localhost"
		logger.Warn("APP_HOST not set, using default", "env:APP_HOST", envHost, "host", host)
	} else {
		host = envHost
	}

	envPort := os.Getenv("APP_PORT")

	var port string

	if envPort == "" {
		port = "8080"
		logger.Warn("APP_PORT not set, using default", "env:APP_PORT", envPort, "port", port)
	} else {
		port = envPort
	}

	return &AppConfig{
		Port: port,
		Host: host,
		Addr: host + ":" + port,
	}
}
