package service

import "testing"

func TestHealthCheck(t *testing.T) {
	result := HealthCheck()

	if result != "ok" {
		t.Errorf("Expected 'ok', got '%s'", result)
	}
}
