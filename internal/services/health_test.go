package services

import (
	"testing"

	logging "go-todo/internal"
)

func TestHealthCheck(t *testing.T) {
	result := NewHealthCheckService(logging.NewDiscardLogger()).HealthCheck()

	if result != "ok" {
		t.Errorf("Expected 'ok', got '%s'", result)
	}
}
