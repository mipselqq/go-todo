package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"goroutine/internal/app"
	"goroutine/internal/config"
	"goroutine/internal/logging"
	"goroutine/internal/repository"
	"goroutine/internal/service"
)

var version = "no version bundled by linker"

func main() {
	if os.Getenv("NOTIFY_ENV") != "prod" {
		_ = godotenv.Load(".env.dev")
	}

	notifyCfg := config.NewNotifyFromEnv(slog.Default())
	logger := logging.NewLogger(notifyCfg.Env, notifyCfg.LogLevel)

	logger.Info("Running", slog.String("version", version))
	logger.Info("Notify config", slog.Any("config", notifyCfg))

	pool, err := app.SetupPostgresFromEnv(logger)
	if err != nil {
		logger.Error("Failed to setup postgres", slog.String("err", err.Error()))
		os.Exit(1)
	}

	err = app.MigratePostgres(pool, logger, "migrations")
	if err != nil {
		logger.Error("Failed to migrate postgres", slog.String("err", err.Error()))
		os.Exit(1)
	}

	worker := service.NewNotificationWorker(logger, repository.NewPGNotification(pool), time.Second, 30)

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err = worker.Run(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logger.Info("Notify worker exited")
			return
		}

		logger.Error("Notification worker stopped", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
