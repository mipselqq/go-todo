package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	logging "go-todo/internal"
	"go-todo/internal/services"
)

func TestHealthCheckResponds200(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthService := services.NewHealthCheckService(logging.NewDiscardLogger())
	healthHandler := NewHealthHandler(logging.NewDiscardLogger(), healthService)

	healthHandler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected OK 200, got %d", rec.Code)
	}
}
