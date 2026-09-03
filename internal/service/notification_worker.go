package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"goroutine/internal/domain"
	"goroutine/internal/logging"
	"goroutine/internal/repository"
)

type notificationRepository interface {
	Claim(ctx context.Context, count int) ([]repository.OutboxEvent, error)
	Ack(ctx context.Context, ids []int64) error
}

type notificationWorker struct {
	notificationRepo notificationRepository
	logger           *slog.Logger
	pollInterval     time.Duration
	claimBatchSize   int
}

func NewNotificationWorker(logger *slog.Logger, notificationRepo notificationRepository, pollInterval time.Duration, claimBatchSize int) *notificationWorker {
	return &notificationWorker{
		notificationRepo: notificationRepo,
		logger:           logging.WithModule(logger, "service.notification_worker"),
		pollInterval:     pollInterval,
		claimBatchSize:   claimBatchSize,
	}
}

func (w *notificationWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := w.processBatch(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (w *notificationWorker) processBatch(ctx context.Context) error {
	events, err := w.notificationRepo.Claim(ctx, w.claimBatchSize)
	if err != nil {
		return fmt.Errorf("notification worker: claim: %v: %w", err, ErrInternal)
	}
	if len(events) == 0 {
		return nil
	}

	ids := make([]int64, len(events))
	for i, event := range events {
		ids[i] = event.ID
		w.logNotification(ctx, &event)
	}

	err = w.notificationRepo.Ack(ctx, ids)
	if err != nil {
		return fmt.Errorf("notification worker: ack: %v: %w", err, ErrInternal)
	}

	return nil
}

func (w *notificationWorker) logNotification(ctx context.Context, event *repository.OutboxEvent) {
	notificationType, err := domain.NewNotificationType(event.EventType)
	if err != nil {
		w.logger.DebugContext(ctx, "Invalid notification type", slog.Int64("id", event.ID), slog.String("err", err.Error()))
		return
	}

	payload, issues := ParseNotificationPayload(notificationType, event.Payload)
	if len(issues) > 0 {
		w.logger.DebugContext(ctx, "Invalid notification payload", slog.Int64("id", event.ID), slog.Any("issues", issues))
		return
	}

	message := FormatNotificationMessage(domain.Notification{
		Type:    notificationType,
		Payload: payload,
	})
	w.logger.DebugContext(ctx, "Notification", slog.Int64("id", event.ID), slog.String("message", message))
}
