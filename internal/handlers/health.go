package handlers

import (
	"log/slog"
	"net/http"

	"go-todo/internal/services"
)

type HealthHandler struct {
	logger  *slog.Logger
	service *services.HealthCheckService
}

func NewHealthHandler(loggerBase *slog.Logger, service *services.HealthCheckService) *HealthHandler {
	return &HealthHandler{
		logger:  loggerBase.With("component", "health_handler"),
		service: service,
	}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	result := h.service.HealthCheck()

	if result == "ok" {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("ok")); err != nil {
			h.logger.Debug("Sent 200 OK response for health check")
		}

		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	h.logger.Error("Sent 500 Internal Server Error response for health check")
}
