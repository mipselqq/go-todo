package main

import (
	"go-todo/internal/handlers"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.HealthCheck)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	addr := ":8080"
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			logger.Error("Failed to listen and serve", "err", err)
			os.Exit(1)
		}
	}()

	logger.Info("Server started at", "addr", addr)

	select {}
}
