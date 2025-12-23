package services

import (
	logging "go-todo/internal"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	result := NewHealthCheckService(logging.NewDiscardLogger()).HealthCheck()

	if result != "ok" {
		t.Errorf("Expected 'ok', got '%s'", result)
	}
}
