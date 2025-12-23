package services

import (
	"go-todo/internal/logging"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	err := NewHealthCheckService(logging.NewDiscardLogger()).HealthCheck()

	if err != nil {
		t.Errorf("Expected no errors, got '%s'", err)
	}
}
