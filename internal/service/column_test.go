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

func TestColumn_Create(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Board not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound},
		{name: "Internal repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
		{name: "Unexpected repository error", repoErr: testutil.ErrUnexpected, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockColumnRepository(t)
			repo.CreateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				name domain.ColumnName,
				description domain.ColumnDescription,
			) (domain.Column, error) {
				if callerID != board.OwnerID || boardID != board.ID {
					t.Errorf("got callerID=%v boardID=%v, want callerID=%v boardID=%v", callerID, boardID, board.OwnerID, board.ID)
				}
				return column, tt.repoErr
			}

			got, err := service.NewColumn(repo).Create(context.Background(), board.OwnerID, board.ID, column.Name, column.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(column, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Create() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestColumn_ListByBoardID(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)

	want := []domain.Column{column}

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Board not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound},
		{name: "Internal repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
		{name: "Unexpected repository error", repoErr: testutil.ErrUnexpected, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockColumnRepository(t)
			repo.ListByBoardIDFunc = func(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) ([]domain.Column, error) {
				if callerID != board.OwnerID || boardID != board.ID {
					t.Errorf("got callerID=%v boardID=%v, want callerID=%v boardID=%v", callerID, boardID, board.OwnerID, board.ID)
				}
				return want, tt.repoErr
			}

			got, err := service.NewColumn(repo).ListByBoardID(context.Background(), board.OwnerID, board.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ListByBoardID() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("ListByBoardID() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestColumn_Update(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	name := column.Name
	description := column.Description

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Column not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Internal repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
		{name: "Unexpected repository error", repoErr: testutil.ErrUnexpected, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockColumnRepository(t)
			repo.UpdateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
				gotName *domain.ColumnName,
				gotDescription *domain.ColumnDescription,
			) (domain.Column, error) {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID {
					t.Errorf("got callerID=%v boardID=%v columnID=%v", callerID, boardID, columnID)
				}
				return column, tt.repoErr
			}

			got, err := service.NewColumn(repo).Update(context.Background(), board.OwnerID, board.ID, column.ID, &name, &description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(column, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Update() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestColumn_Move(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	position := column.Position

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Column not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Index out of bounds", repoErr: repository.ErrIndexOutOfBounds, wantErr: service.ErrIndexOutOfBounds},
		{name: "Internal repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
		{name: "Unexpected repository error", repoErr: testutil.ErrUnexpected, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockColumnRepository(t)
			repo.MoveFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				columnID domain.ColumnID,
				targetPosition domain.ColumnPosition,
			) (domain.ColumnPosition, error) {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID || targetPosition != position {
					t.Errorf("got unexpected move arguments")
				}
				return position, tt.repoErr
			}

			got, err := service.NewColumn(repo).Move(context.Background(), board.OwnerID, board.ID, column.ID, position)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Move() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got != position {
				t.Errorf("Move() = %v, want %v", got, position)
			}
		})
	}
}

func TestColumn_Delete(t *testing.T) {
	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Column not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrColumnNotFound},
		{name: "Internal repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
		{name: "Unexpected repository error", repoErr: testutil.ErrUnexpected, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockColumnRepository(t)
			repo.DeleteFunc = func(ctx context.Context, callerID domain.UserID, boardID domain.BoardID, columnID domain.ColumnID) error {
				if callerID != board.OwnerID || boardID != board.ID || columnID != column.ID {
					t.Errorf("got unexpected delete arguments")
				}
				return tt.repoErr
			}

			err := service.NewColumn(repo).Delete(context.Background(), board.OwnerID, board.ID, column.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
