package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-todo/internal/config"
	"go-todo/internal/handlers"
	"go-todo/internal/logging"
	"go-todo/internal/services"
)

func main() {
	loggerConfig := config.NewLoggerConfigFromEnv()
	loggerBase := logging.NewLoggerBase(loggerConfig)
	logger := loggerBase.With("scope", "main")

	appConfig := config.NewAppConfigFromEnv(loggerBase)

	mux := http.NewServeMux()

	healthService := services.NewHealthCheckService(loggerBase)
	healthHandler := handlers.NewHealthHandler(loggerBase, healthService)
	mux.HandleFunc("/health", healthHandler.HealthCheck)

	server := &http.Server{
		Addr:         appConfig.Addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			loggerBase.Error("Failed to listen and serve", "err", err)
			os.Exit(1)
		}
	}()

	loggerBase.Info("Server started at", "addr", appConfig.Addr, "pretty_url", "http://"+appConfig.Addr)

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, os.Interrupt, syscall.SIGTERM)

	<-quitChan
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "err", err)
	} else {
		logger.Info("Server exited gracefully")
	}
}
