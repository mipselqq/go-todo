package services

import "log/slog"

type HealthCheckService struct {
	logger *slog.Logger
}

func NewHealthCheckService(logger_base *slog.Logger) *HealthCheckService {
	return &HealthCheckService{logger: logger_base.With("component", "health_service")}
}

func (s *HealthCheckService) HealthCheck() string {
	s.logger.Info("Health check passed")
	return "ok"
}
