package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
	"goroutine/internal/service"
	"goroutine/internal/testutil"
)

func TestTask_Create(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	task := testutil.ValidTask(column.ID)

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Column not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTaskRepository(t)
			repo.CreateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
				name domain.TaskName,
				description domain.TaskDescription,
			) (domain.Task, error) {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID {
					t.Errorf("got unexpected create arguments")
				}
				return task, tt.repoErr
			}

			got, err := service.NewTask(repo).Create(context.Background(), board.OwnerID, board.ID, column.ID, task.Name, task.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(task, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Create() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestTask_ListByColumnID(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	task := testutil.ValidTask(column.ID)

	want := []domain.Task{task}

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Column not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTaskRepository(t)
			repo.ListByColumnIDFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
			) ([]domain.Task, error) {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID {
					t.Errorf("got unexpected list arguments")
				}
				return want, tt.repoErr
			}

			got, err := service.NewTask(repo).ListByColumnID(context.Background(), board.OwnerID, board.ID, column.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ListByColumnID() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("ListByColumnID() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestTask_Update(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	task := testutil.ValidTask(column.ID)
	name := task.Name
	description := task.Description

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Task not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrTaskNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTaskRepository(t)
			repo.UpdateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
				taskID domain.TaskID,
				gotName *domain.TaskName,
				gotDescription *domain.TaskDescription,
			) (domain.Task, error) {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID || taskID != task.ID {
					t.Errorf("got unexpected update arguments")
				}
				return task, tt.repoErr
			}

			got, err := service.NewTask(repo).Update(context.Background(), board.OwnerID, board.ID, column.ID, task.ID, &name, &description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(task, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Update() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestTask_Move(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	targetColumn := testutil.NewValidColumn(t, board.ID, "Target", 2)
	task := testutil.ValidTask(column.ID)
	position := task.Position

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Task not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrTaskNotFound},
		{name: "Target column not found", repoErr: repository.ErrTargetRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Index out of bounds", repoErr: repository.ErrIndexOutOfBounds, wantErr: service.ErrIndexOutOfBounds},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTaskRepository(t)
			repo.MoveFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				currentColumnID domain.ColumnID,
				taskID domain.TaskID,
				targetColumnID domain.ColumnID,
				targetPosition domain.TaskPosition,
			) (domain.ColumnID, domain.TaskPosition, error) {
				if callerID != board.OwnerID || boardID != board.ID || currentColumnID != column.ID ||
					taskID != task.ID || targetColumnID != targetColumn.ID || targetPosition != position {
					t.Errorf("got unexpected move arguments")
				}
				return targetColumn.ID, position, tt.repoErr
			}

			gotColumnID, gotPosition, err := service.NewTask(repo).Move(
				context.Background(),
				board.OwnerID,
				board.ID,
				column.ID,
				task.ID,
				targetColumn.ID,
				position,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Move() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if gotColumnID != targetColumn.ID || gotPosition != position {
					t.Errorf("Move() = (%v, %v), want (%v, %v)", gotColumnID, gotPosition, targetColumn.ID, position)
				}
			}
		})
	}
}

func TestTask_Delete(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	task := testutil.ValidTask(column.ID)

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Task not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrTaskNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockTaskRepository(t)
			repo.DeleteFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
				taskID domain.TaskID,
			) error {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID || taskID != task.ID {
					t.Errorf("got unexpected delete arguments")
				}
				return tt.repoErr
			}

			err := service.NewTask(repo).Delete(context.Background(), board.OwnerID, board.ID, column.ID, task.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
