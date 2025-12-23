package services

import "log/slog"

type HealthCheckService struct {
	logger *slog.Logger
}

func NewHealthCheckService(loggerBase *slog.Logger) *HealthCheckService {
	return &HealthCheckService{logger: loggerBase.With("component", "health_service")}
}

func (s *HealthCheckService) HealthCheck() error {
	s.logger.Debug("Health check passed")

	return nil
}
