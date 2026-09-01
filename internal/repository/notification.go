package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGNotification struct {
	pgPool *pgxpool.Pool
}

func NewPGNotification(pgPool *pgxpool.Pool) *PGNotification {
	return &PGNotification{
		pgPool: pgPool,
	}
}

type OutboxEvent struct {
	ID              int64
	RecipientUserID uuid.UUID
	EventType       string
	Payload         []byte
	CreatedAt       time.Time
}

func (r *PGNotification) Claim(ctx context.Context, count int) ([]OutboxEvent, error) {
	const query = `
		SELECT id, recipient_user_id, event_type, payload, created_at
		FROM notification_outbox
		ORDER BY row_number() OVER (PARTITION BY recipient_user_id ORDER BY id), id
		LIMIT @count`

	rows, err := r.pgPool.Query(ctx, query, pgx.NamedArgs{"count": count})
	if err != nil {
		return nil, fmt.Errorf("notification repo: claim: %v: %w", err, ErrInternal)
	}
	defer rows.Close()

	var result []OutboxEvent
	for rows.Next() {
		record, scanErr := ScanOutboxRecord(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("notification repo: claim: scan: %v: %w", scanErr, ErrInternal)
		}
		result = append(result, record)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("notification repo: claim: rows final error: %v: %w", err, ErrInternal)
	}

	return result, nil
}

func ScanOutboxRecord(row interface{ Scan(...any) error }) (OutboxEvent, error) {
	var record OutboxEvent
	err := row.Scan(
		&record.ID,
		&record.RecipientUserID,
		&record.EventType,
		&record.Payload,
		&record.CreatedAt,
	)
	if err != nil {
		return OutboxEvent{}, fmt.Errorf("scan notification: %w", err)
	}

	return record, nil
}
