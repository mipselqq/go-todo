package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"goroutine/internal/config"
	"goroutine/internal/testutil"

	"github.com/google/go-cmp/cmp"
)

var defaultNotifyConfig = config.Notify{
	LogLevel: "info",
	Env:      "dev",
}

var notifyEnvVars = []string{"NOTIFY_LOG_LEVEL", "NOTIFY_ENV"}

func setCustomNotifyEnvVars(t *testing.T) {
	t.Setenv("NOTIFY_LOG_LEVEL", "debug")
	t.Setenv("NOTIFY_ENV", "prod")
}

func TestNewNotifyFromEnv(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		UnsetEnv(t, notifyEnvVars...)

		cfg := config.NewNotifyFromEnv(testutil.NewDiscardLogger())

		diff := cmp.Diff(defaultNotifyConfig, cfg)
		if diff != "" {
			t.Errorf("NewNotifyFromEnv() diff (-want +got):\n%s", diff)
		}
	})

	t.Run("uses env vars", func(t *testing.T) {
		setCustomNotifyEnvVars(t)

		cfg := config.NewNotifyFromEnv(testutil.NewDiscardLogger())
		wantCfg := config.Notify{
			LogLevel: "debug",
			Env:      "prod",
		}

		diff := cmp.Diff(wantCfg, cfg)
		if diff != "" {
			t.Errorf("NewNotifyFromEnv() diff (-want +got):\n%s", diff)
		}
	})

	t.Run("warnings on unset variables", func(t *testing.T) {
		UnsetEnv(t, notifyEnvVars...)

		logger, buf := testutil.NewBufJSONLogger(t, slog.LevelWarn)
		_ = config.NewNotifyFromEnv(logger)

		for _, envVar := range notifyEnvVars {
			if !strings.Contains(buf.String(), envVar) {
				t.Errorf("got log output %q, want mention of %q", buf.String(), envVar)
			}
		}
	})

	t.Run("no warnings if all variables are set", func(t *testing.T) {
		setCustomNotifyEnvVars(t)

		logger, buf := testutil.NewBufJSONLogger(t, slog.LevelWarn)
		_ = config.NewNotifyFromEnv(logger)

		if buf.String() != "" {
			t.Errorf("got warnings %q, want none", buf.String())
		}
	})
}

func TestNotify_LogValue(t *testing.T) {
	v := defaultNotifyConfig.LogValue()
	if v.Kind() != slog.KindGroup {
		t.Fatalf("got kind %v, want Group", v.Kind())
	}

	wantAttrs := map[string]string{
		"log_level": "info",
		"env":       "dev",
	}

	testutil.FailOnInvalidLogValue(t, v.Group(), wantAttrs)
}
