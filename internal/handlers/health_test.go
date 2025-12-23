package handlers

import (
	logging "go-todo/internal"
	"go-todo/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckResponds200(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	healthService := services.NewHealthCheckService(logging.NewDiscardLogger())
	healthHandler := NewHealthHandler(logging.NewDiscardLogger(), healthService)

	healthHandler.HealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected OK 200, got %d", rec.Code)
	}
}
