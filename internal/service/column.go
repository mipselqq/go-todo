package service

import (
	"context"
	"errors"
	"fmt"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
)

type columnRepository interface {
	Create(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, name domain.ColumnName, description domain.ColumnDescription) (domain.Column, error)
	ListByBoardID(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) ([]domain.Column, error)
	Update(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID, name *domain.ColumnName, description *domain.ColumnDescription) (domain.Column, error)
	Move(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID, targetPosition domain.ColumnPosition) (domain.ColumnPosition, error)
	Delete(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID) error
}

type column struct {
	columnRepo columnRepository
}

func NewColumn(columnRepo columnRepository) *column {
	return &column{columnRepo: columnRepo}
}

func (s *column) Create(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, name domain.ColumnName, description domain.ColumnDescription) (domain.Column, error) {
	column, err := s.columnRepo.Create(ctx, callerID, boardID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Column{}, ErrBoardNotFound
		}
		return domain.Column{}, fmt.Errorf("column service: create: %v: %w", err, ErrInternal)
	}

	return column, nil
}

func (s *column) ListByBoardID(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) ([]domain.Column, error) {
	columns, err := s.columnRepo.ListByBoardID(ctx, callerID, boardID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return nil, ErrBoardNotFound
		}
		return nil, fmt.Errorf("column service: list by board id: %v: %w", err, ErrInternal)
	}

	return columns, nil
}

func (s *column) Update(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	name *domain.ColumnName,
	description *domain.ColumnDescription,
) (domain.Column, error) {
	updated, err := s.columnRepo.Update(ctx, callerID, boardID, columnID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Column{}, ErrColumnNotFound
		}
		return domain.Column{}, fmt.Errorf("column service: update: %v: %w", err, ErrInternal)
	}

	return updated, nil
}

func (s *column) Delete(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
) error {
	err := s.columnRepo.Delete(ctx, callerID, boardID, columnID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return ErrColumnNotFound
		}
		return fmt.Errorf("column service: delete: %v: %w", err, ErrInternal)
	}

	return nil
}

func (s *column) Move(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	targetPosition domain.ColumnPosition,
) (domain.ColumnPosition, error) {
	position, err := s.columnRepo.Move(ctx, callerID, boardID, columnID, targetPosition)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.ColumnPosition{}, ErrColumnNotFound
		}
		if errors.Is(err, repository.ErrIndexOutOfBounds) {
			return domain.ColumnPosition{}, ErrIndexOutOfBounds
		}
		return domain.ColumnPosition{}, fmt.Errorf("column service: move: %v: %w", err, ErrInternal)
	}

	return position, nil
}
