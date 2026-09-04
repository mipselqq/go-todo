package main

import (
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"goroutine/internal/app"
)

func main() {
	logger := slog.Default()

	pool, err := app.SetupPostgresFromEnv(logger)
	if err != nil {
		logger.Error("failed to setup postgres", slog.String("err", err.Error()))
		os.Exit(1)
	}
	err = migratePostgres(pool, "migrations")
	if err != nil {
		logger.Error("failed to migrate postgres", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

func migratePostgres(pool *pgxpool.Pool, migrationsDir string) error {
	err := goose.SetDialect("postgres")
	if err != nil {
		return err
	}

	return goose.Up(stdlib.OpenDBFromPool(pool), migrationsDir)
}
