package main

import (
	"context"
	logging "go-todo/internal"
	"go-todo/internal/handlers"
	"go-todo/internal/services"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger_base := logging.NewLoggerBase()
	logger := logger_base.With("component", "main")

	mux := http.NewServeMux()

	healthService := services.NewHealthCheckService(logger_base)
	healthHandler := handlers.NewHealthHandler(logger_base, healthService)
	mux.HandleFunc("/health", healthHandler.HealthCheck)

	addr := ":8080"
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger_base.Error("Failed to listen and serve", "err", err)
			os.Exit(1)
		}
	}()
	logger_base.Info("Server started at", "addr", addr)

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
