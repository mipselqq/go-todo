package handlers

import (
	"go-todo/internal/services"
	"log/slog"
	"net/http"
)

type HealthHandler struct {
	logger  *slog.Logger
	service *services.HealthCheckService
}

func NewHealthHandler(logger_base *slog.Logger, service *services.HealthCheckService) *HealthHandler {
	return &HealthHandler{
		logger:  logger_base.With("component", "health_handler"),
		service: service,
	}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	result := h.service.HealthCheck()

	if result == "ok" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		h.logger.Info("Sent 200 OK response for health check")
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	h.logger.Error("Sent 500 Internal Server Error response for health check")
}
