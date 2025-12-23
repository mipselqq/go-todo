package services

import (
	"testing"

	logging "go-todo/internal"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	result := NewHealthCheckService(logging.NewDiscardLogger()).HealthCheck()

	if result != "ok" {
		t.Errorf("Expected 'ok', got '%s'", result)
	}
}
