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

func TestBoard_Create(t *testing.T) {
	board := testutil.ValidBoard()
	repo := NewMockBoardRepository(t)
	repo.CreateFunc = func(
		ctx context.Context,
		ownerID domain.UserID,
		name domain.BoardName,
		description domain.BoardDescription,
	) (domain.Board, error) {
		if ownerID != board.OwnerID {
			t.Errorf("got ownerID %v, want %v", ownerID, board.OwnerID)
		}
		if name != board.Name {
			t.Errorf("got name %v, want %v", name, board.Name)
		}
		if description != board.Description {
			t.Errorf("got description %v, want %v", description, board.Description)
		}
		return board, nil
	}

	got, err := service.NewBoard(repo).Create(context.Background(), board.OwnerID, board.Name, board.Description)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if diff := cmp.Diff(board, got, testutil.CmpAllowUnexported()); diff != "" {
		t.Errorf("Create() mismatch (-want +got):\n%s", diff)
	}
}

func TestBoard_ListByOwnerID(t *testing.T) {
	board := testutil.ValidBoard()
	want := []domain.Board{board}
	repo := NewMockBoardRepository(t)
	repo.ListByOwnerIDFunc = func(ctx context.Context, ownerID domain.UserID) ([]domain.Board, error) {
		if ownerID != board.OwnerID {
			t.Errorf("got ownerID %v, want %v", ownerID, board.OwnerID)
		}
		return want, nil
	}

	got, err := service.NewBoard(repo).ListByOwnerID(context.Background(), board.OwnerID)
	if err != nil {
		t.Fatalf("ListByOwnerID() error = %v", err)
	}
	if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
		t.Errorf("ListByOwnerID() mismatch (-want +got):\n%s", diff)
	}
}

func TestBoard_Get(t *testing.T) {
	board := testutil.ValidBoard()
	tests := []struct {
		name     string
		repoErr  error
		wantErr  error
		wantZero bool
	}{
		{name: "Success"},
		{name: "Not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound, wantZero: true},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal, wantZero: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockBoardRepository(t)
			repo.GetFunc = func(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) (domain.Board, error) {
				if callerID != board.OwnerID {
					t.Errorf("got callerID %v, want %v", callerID, board.OwnerID)
				}
				if boardID != board.ID {
					t.Errorf("got boardID %v, want %v", boardID, board.ID)
				}
				return board, tt.repoErr
			}

			got, err := service.NewBoard(repo).Get(context.Background(), board.OwnerID, board.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantZero {
				if got != (domain.Board{}) {
					t.Errorf("Get() = %v, want zero board", got)
				}
				return
			}
			if diff := cmp.Diff(board, got, testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBoard_GetAggregate(t *testing.T) {
	board := testutil.ValidBoard()
	firstColumn := testutil.NewValidColumn(t, board.ID, "First", 1)
	secondColumn := testutil.NewValidColumn(t, board.ID, "Second", 2)
	firstTask := testutil.NewValidTask(t, firstColumn.ID, "First", "First", 1)
	secondTask := testutil.NewValidTask(t, firstColumn.ID, "Second", "Second", 2)

	want := service.AggregateBoard{
		Board: board,
		Columns: []service.AggregateColumn{
			{Column: firstColumn, Tasks: []domain.Task{firstTask, secondTask}},
			{Column: secondColumn, Tasks: []domain.Task{}},
		},
	}

	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockBoardRepository(t)
			repo.GetAggregateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
			) (domain.Board, []domain.Column, []domain.Task, error) {
				if callerID != board.OwnerID {
					t.Errorf("got callerID %v, want %v", callerID, board.OwnerID)
				}
				if boardID != board.ID {
					t.Errorf("got boardID %v, want %v", boardID, board.ID)
				}
				if tt.repoErr != nil {
					return domain.Board{}, nil, nil, tt.repoErr
				}
				return board, []domain.Column{secondColumn, firstColumn}, []domain.Task{secondTask, firstTask}, nil
			}

			got, err := service.NewBoard(repo).GetAggregate(context.Background(), board.OwnerID, board.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetAggregate() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("GetAggregate() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBoard_Update(t *testing.T) {
	board := testutil.ValidBoard()
	name := board.Name
	description := board.Description
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockBoardRepository(t)
			repo.UpdateFunc = func(
				ctx context.Context,
				callerID domain.UserID,
				boardID domain.BoardID,
				gotName *domain.BoardName,
				gotDescription *domain.BoardDescription,
			) (domain.Board, error) {
				if callerID != board.OwnerID || boardID != board.ID {
					t.Errorf("got callerID=%v boardID=%v, want callerID=%v boardID=%v", callerID, boardID, board.OwnerID, board.ID)
				}
				if gotName != &name || gotDescription != &description {
					t.Errorf("got name=%v description=%v, want name=%v description=%v", gotName, gotDescription, &name, &description)
				}
				return board, tt.repoErr
			}

			got, err := service.NewBoard(repo).Update(context.Background(), board.OwnerID, board.ID, &name, &description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(board, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Update() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBoard_Delete(t *testing.T) {
	board := testutil.ValidBoard()
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "Success"},
		{name: "Not found", repoErr: repository.ErrRowNotFound, wantErr: service.ErrBoardNotFound},
		{name: "Repository error", repoErr: repository.ErrInternal, wantErr: service.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockBoardRepository(t)
			repo.DeleteFunc = func(ctx context.Context, callerID domain.UserID, boardID domain.BoardID) error {
				if callerID != board.OwnerID || boardID != board.ID {
					t.Errorf("got callerID=%v boardID=%v, want callerID=%v boardID=%v", callerID, boardID, board.OwnerID, board.ID)
				}
				return tt.repoErr
			}

			err := service.NewBoard(repo).Delete(context.Background(), board.OwnerID, board.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
