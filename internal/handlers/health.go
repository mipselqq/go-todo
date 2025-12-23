package handlers

import (
	"go-todo/internal/service"
	"net/http"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	result := service.HealthCheck()

	if result == "ok" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}
