package service

import (
	"context"
	"errors"
	"fmt"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
)

type taskRepository interface {
	Create(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID, name domain.TaskName, description domain.TaskDescription) (domain.Task, error)
	ListByColumnID(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID) ([]domain.Task, error)
	Update(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID, taskID domain.TaskID, name *domain.TaskName, description *domain.TaskDescription) (domain.Task, error)
	Move(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, currentColumnID domain.ColumnID, taskID domain.TaskID, targetColumnID domain.ColumnID, targetPosition domain.TaskPosition) (domain.ColumnID, domain.TaskPosition, error)
	Delete(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID, taskID domain.TaskID) error
}

type task struct {
	taskRepo taskRepository
}

func NewTask(taskRepo taskRepository) *task {
	return &task{taskRepo: taskRepo}
}

func (s *task) Create(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	name domain.TaskName,
	description domain.TaskDescription,
) (domain.Task, error) {
	task, err := s.taskRepo.Create(ctx, callerID, boardID, columnID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Task{}, ErrColumnNotFound
		}
		return domain.Task{}, fmt.Errorf("task service: create: %v: %w", err, ErrInternal)
	}

	return task, nil
}

func (s *task) ListByColumnID(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
) ([]domain.Task, error) {
	tasks, err := s.taskRepo.ListByColumnID(ctx, callerID, boardID, columnID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return nil, ErrColumnNotFound
		}
		return nil, fmt.Errorf("task service: list: %v: %w", err, ErrInternal)
	}

	return tasks, nil
}

func (s *task) Update(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
	name *domain.TaskName,
	description *domain.TaskDescription,
) (domain.Task, error) {
	updated, err := s.taskRepo.Update(ctx, callerID, boardID, columnID, taskID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Task{}, ErrTaskNotFound
		}
		return domain.Task{}, fmt.Errorf("task service: update: %v: %w", err, ErrInternal)
	}

	return updated, nil
}

func (s *task) Delete(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
) error {
	err := s.taskRepo.Delete(ctx, callerID, boardID, columnID, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("task service: delete: %v: %w", err, ErrInternal)
	}

	return nil
}

func (s *task) Move(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
	targetColumnID domain.ColumnID,
	targetPosition domain.TaskPosition,
) (domain.ColumnID, domain.TaskPosition, error) {
	newColumnID, newPosition, err := s.taskRepo.Move(ctx, callerID, boardID, columnID, taskID, targetColumnID, targetPosition)
	if err != nil {
		if errors.Is(err, repository.ErrTargetRowNotFound) {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrColumnNotFound
		}
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrTaskNotFound
		}
		if errors.Is(err, repository.ErrIndexOutOfBounds) {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrIndexOutOfBounds
		}
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task service: move: %v: %w", err, ErrInternal)
	}

	return newColumnID, newPosition, nil
}
