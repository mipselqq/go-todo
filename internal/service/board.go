package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
)

type boardRepository interface {
	Create(ctx context.Context, ownerID domain.UserID, name domain.BoardName, description domain.BoardDescription) (domain.Board, error)
	Get(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) (domain.Board, error)
	GetAggregate(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) (domain.Board, []domain.Column, []domain.Task, error)
	ListByOwnerID(ctx context.Context, ownerID domain.UserID) ([]domain.Board, error)
	Update(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, name *domain.BoardName, description *domain.BoardDescription) (domain.Board, error)
	Delete(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) error
}

type board struct {
	boardRepo boardRepository
}

func NewBoard(boardRepo boardRepository) *board {
	return &board{boardRepo: boardRepo}
}

type AggregateBoard struct {
	Board   domain.Board
	Columns []AggregateColumn
}

type AggregateColumn struct {
	Column domain.Column
	Tasks  []domain.Task
}

func (s *board) Create(ctx context.Context, callerID domain.UserID, name domain.BoardName, description domain.BoardDescription) (domain.Board, error) {
	board, err := s.boardRepo.Create(ctx, callerID, name, description)
	if err != nil {
		return domain.Board{}, fmt.Errorf("board service: create: %v: %w", err, ErrInternal)
	}

	return board, nil
}

func (s *board) ListByOwnerID(ctx context.Context, callerID domain.UserID) ([]domain.Board, error) {
	boards, err := s.boardRepo.ListByOwnerID(ctx, callerID)
	if err != nil {
		return nil, fmt.Errorf("board service: list by owner id: %v: %w", err, ErrInternal)
	}

	return boards, nil
}

func (s *board) Get(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) (domain.Board, error) {
	board, err := s.boardRepo.Get(ctx, callerID, boardID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Board{}, ErrBoardNotFound
		}
		return domain.Board{}, fmt.Errorf("board service: get: %v: %w", err, ErrInternal)
	}

	return board, nil
}

func (s *board) GetAggregate(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) (AggregateBoard, error) {
	board, columns, tasks, err := s.boardRepo.GetAggregate(ctx, callerID, boardID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return AggregateBoard{}, ErrBoardNotFound
		}
		return AggregateBoard{}, fmt.Errorf("board service: get aggregate: %v: %w", err, ErrInternal)
	}

	aggregate := AggregateBoard{
		Board:   board,
		Columns: make([]AggregateColumn, len(columns)),
	}

	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position.Int64() < columns[j].Position.Int64()
	})

	columnIDToTaskMap := make(map[domain.ColumnID][]domain.Task, len(columns))
	for _, t := range tasks {
		columnIDToTaskMap[t.ColumnID] = append(columnIDToTaskMap[t.ColumnID], t)
	}
	for columnID, colTasks := range columnIDToTaskMap {
		sort.Slice(colTasks, func(i, j int) bool {
			return colTasks[i].Position.Int64() < colTasks[j].Position.Int64()
		})
		columnIDToTaskMap[columnID] = colTasks
	}

	for i, column := range columns {
		colTasks := columnIDToTaskMap[column.ID]
		if colTasks == nil {
			colTasks = []domain.Task{}
		}
		aggregate.Columns[i] = AggregateColumn{
			Column: column,
			Tasks:  colTasks,
		}
	}

	return aggregate, nil
}

func (s *board) Update(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	name *domain.BoardName,
	description *domain.BoardDescription,
) (domain.Board, error) {
	updated, err := s.boardRepo.Update(ctx, callerID, boardID, name, description)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return domain.Board{}, ErrBoardNotFound
		}
		return domain.Board{}, fmt.Errorf("board service: update: %v: %w", err, ErrInternal)
	}

	return updated, nil
}

func (s *board) Delete(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) error {
	err := s.boardRepo.Delete(ctx, callerID, boardID)
	if err != nil {
		if errors.Is(err, repository.ErrRowNotFound) {
			return ErrBoardNotFound
		}
		return fmt.Errorf("board service: delete: %v: %w", err, ErrInternal)
	}

	return nil
}
