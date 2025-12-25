package config

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestAppConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("DefaultValues", func(t *testing.T) {
		os.Unsetenv("APP_HOST")
		os.Unsetenv("APP_PORT")

		config := NewAppConfigFromEnv(logger)

		if config.Host != "0.0.0.0" {
			t.Errorf("expected host '0.0.0.0', got '%s'", config.Host)
		}
		if config.Port != "8080" {
			t.Errorf("expected port '8080', got '%s'", config.Port)
		}
		if config.Addr != "0.0.0.0:8080" {
			t.Errorf("expected addr '0.0.0.0:8080', got '%s'", config.Addr)
		}
	})

	t.Run("CustomValues", func(t *testing.T) {
		os.Setenv("APP_HOST", "127.0.0.1")
		os.Setenv("APP_PORT", "3000")
		defer func() {
			os.Unsetenv("APP_HOST")
			os.Unsetenv("APP_PORT")
		}()

		config := NewAppConfigFromEnv(logger)

		if config.Host != "127.0.0.1" {
			t.Errorf("expected host '127.0.0.1', got '%s'", config.Host)
		}
		if config.Port != "3000" {
			t.Errorf("expected port '3000', got '%s'", config.Port)
		}
		if config.Addr != "127.0.0.1:3000" {
			t.Errorf("expected addr '127.0.0.1:3000', got '%s'", config.Addr)
		}
	})

	t.Run("EmptyEnvVars", func(t *testing.T) {
		os.Setenv("APP_HOST", "")
		os.Setenv("APP_PORT", "")
		defer func() {
			os.Unsetenv("APP_HOST")
			os.Unsetenv("APP_PORT")
		}()

		config := NewAppConfigFromEnv(logger)

		if config.Host != "0.0.0.0" {
			t.Errorf("expected host '0.0.0.0' for empty env, got '%s'", config.Host)
		}
		if config.Port != "8080" {
			t.Errorf("expected port '8080' for empty env, got '%s'", config.Port)
		}
		if config.Addr != "0.0.0.0:8080" {
			t.Errorf("expected addr '0.0.0.0:8080' for empty env, got '%s'", config.Addr)
		}
	})
}
