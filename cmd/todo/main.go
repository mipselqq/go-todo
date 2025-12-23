package main

import (
	logging "go-todo/internal"
	"go-todo/internal/handlers"
	"go-todo/internal/services"
	"net/http"
	"os"
	"time"
)

func main() {
	logger_base := logging.NewLoggerBase()
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

	select {}
}
