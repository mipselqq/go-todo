//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
	"goroutine/internal/testutil"
)

func TestBoardRepository_Create(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	boardName := testutil.ValidBoardName()
	boardDescription := testutil.ValidBoardDescription()

	t.Run("Success", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)
		hierarchy := insertBoardHierarchy(t, pool)

		board, err := r.Create(context.Background(), hierarchy.board.OwnerID, boardName, boardDescription)
		if err != nil {
			t.Errorf("Create() error = %v", err)
		}
		if board.ID.IsNil() {
			t.Errorf("got empty board ID, want generated ID")
		}
		if board.OwnerID != hierarchy.board.OwnerID {
			t.Errorf("got owner ID %q, want %q", board.OwnerID, hierarchy.board.OwnerID)
		}
		if board.Name != boardName {
			t.Errorf("got name %q, want %q", board.Name, boardName)
		}
		if board.Description != boardDescription {
			t.Errorf("got description %q, want %q", board.Description, boardDescription)
		}
		if board.CreatedAt.IsZero() {
			t.Errorf("got zero createdAt, want set value")
		}
		if board.UpdatedAt.IsZero() {
			t.Errorf("got zero updatedAt, want set value")
		}
		if !board.CreatedAt.Equal(board.UpdatedAt) {
			t.Errorf("got createdAt=%v updatedAt=%v, want equal", board.CreatedAt, board.UpdatedAt)
		}
		AssertTimestampPrecisionAtLeastMillis(t, pool, "boards", "created_at", "updated_at")

		for _, storedBoard := range ListBoards(t, pool) {
			if storedBoard.ID == board.ID {
				if diff := cmp.Diff(board, storedBoard, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("stored board mismatch (-returned +stored):\n%s", diff)
				}
				return
			}
		}
		t.Errorf("created board %v not found", board.ID)
	})
}

func TestBoardRepository_Get(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	tests := []struct {
		name            string
		useAnotherOwner bool
		useMissingOwner bool
		useAnotherBoard bool
		useMissingBoard bool
		wantErr         error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			if tt.useAnotherOwner {
				callerID = hierarchy.anotherBoard.OwnerID
			}
			if tt.useMissingOwner {
				callerID = hierarchy.missingBoard.OwnerID
			}
			if tt.useAnotherBoard {
				targetBoard = hierarchy.anotherBoard
			}
			if tt.useMissingBoard {
				targetBoard = hierarchy.missingBoard
			}

			got, err := r.Get(context.Background(), callerID, targetBoard.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(hierarchy.board, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Get() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBoardRepository_ListByOwnerID(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	tests := []struct {
		name           string
		useAnotherUser bool
		useMissingUser bool
	}{
		{name: "Success"},
		{name: "Another user", useAnotherUser: true},
		{name: "Missing user", useMissingUser: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			ownerID := hierarchy.board.OwnerID
			want := []domain.Board{hierarchy.board}
			if tt.useAnotherUser {
				ownerID = hierarchy.anotherBoard.OwnerID
				want = []domain.Board{hierarchy.anotherBoard}
			}
			if tt.useMissingUser {
				ownerID = hierarchy.missingBoard.OwnerID
				want = nil
			}

			got, err := r.ListByOwnerID(context.Background(), ownerID)
			if err != nil {
				t.Errorf("ListByOwnerID() error = %v", err)
			}
			if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("ListByOwnerID() mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Success ordered and filtered by owner", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)
		hierarchy := insertBoardHierarchy(t, pool)

		laterBoard := testutil.ValidBoardForOwner(hierarchy.board.OwnerID)
		laterBoard.CreatedAt = testutil.Fixed5mFromNow()
		laterBoard.UpdatedAt = testutil.Fixed5mFromNow()
		CreateBoard(t, pool, &laterBoard)

		got, err := r.ListByOwnerID(context.Background(), hierarchy.board.OwnerID)
		if err != nil {
			t.Fatalf("ListByOwnerID() error = %v", err)
		}

		want := []domain.Board{hierarchy.board, laterBoard}
		if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("ListByOwnerID() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestBoardRepository_Update(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	assertUpdatedBoard := func(t *testing.T, got domain.Board, want domain.Board) {
		t.Helper()

		if got.ID != want.ID {
			t.Errorf("got id %q, want %q", got.ID, want.ID)
		}
		if got.OwnerID != want.OwnerID {
			t.Errorf("got ownerID %q, want %q", got.OwnerID, want.OwnerID)
		}
		if got.Name != want.Name {
			t.Errorf("got name %q, want %q", got.Name, want.Name)
		}
		if got.Description != want.Description {
			t.Errorf("got description %q, want %q", got.Description, want.Description)
		}
		if !got.CreatedAt.Truncate(time.Millisecond).Equal(want.CreatedAt.Truncate(time.Millisecond)) {
			t.Errorf("got createdAt %v, want %v (at millisecond precision)", got.CreatedAt, want.CreatedAt)
		}
		if !got.UpdatedAt.After(want.UpdatedAt) {
			t.Errorf("got updatedAt %v, want after %v", got.UpdatedAt, want.UpdatedAt)
		}
		AssertTimestampPrecisionAtLeastMillis(t, pool, "boards", "created_at", "updated_at")

		for _, storedBoard := range ListBoards(t, pool) {
			if storedBoard.ID == got.ID {
				if diff := cmp.Diff(got, storedBoard, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("got stored board mismatch (-want +got):\n%s", diff)
				}
				return
			}
		}
		t.Errorf("updated board %v not found", got.ID)
	}

	tests := []struct {
		name            string
		useAnotherOwner bool
		useMissingOwner bool
		useAnotherBoard bool
		useMissingBoard bool
		wantErr         error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			if tt.useAnotherOwner {
				callerID = hierarchy.anotherBoard.OwnerID
			}
			if tt.useMissingOwner {
				callerID = hierarchy.missingBoard.OwnerID
			}
			if tt.useAnotherBoard {
				targetBoard = hierarchy.anotherBoard
			}
			if tt.useMissingBoard {
				targetBoard = hierarchy.missingBoard
			}

			want := testutil.UpdateValidBoard(t, &hierarchy.board, "Updated Board Name", "Updated Board Description", hierarchy.board.UpdatedAt)
			got, err := r.Update(context.Background(), callerID, targetBoard.ID, &want.Name, &want.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				assertUpdatedBoard(t, got, want)
				return
			}

			if diff := cmp.Diff([]domain.Board{hierarchy.board, hierarchy.anotherBoard}, ListBoards(t, pool), testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("stored boards mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Success partial name only", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		want := testutil.UpdateValidBoard(
			t,
			&hierarchy.board,
			"Updated Board Name Only",
			hierarchy.board.Description.String(),
			hierarchy.board.UpdatedAt,
		)

		got, err := r.Update(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID, &want.Name, nil)
		if err != nil {
			t.Errorf("Update() error = %v", err)
		}
		assertUpdatedBoard(t, got, want)
	})

	t.Run("Success partial description only", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		want := testutil.UpdateValidBoard(
			t,
			&hierarchy.board,
			hierarchy.board.Name.String(),
			"Updated Board Description Only",
			hierarchy.board.UpdatedAt,
		)

		got, err := r.Update(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID, nil, &want.Description)
		if err != nil {
			t.Errorf("Update() error = %v", err)
		}
		assertUpdatedBoard(t, got, want)
	})

	t.Run("Success no changes", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)

		got, err := r.Update(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID, nil, nil)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if diff := cmp.Diff(hierarchy.board, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("Update() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestBoardRepository_Delete(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	tests := []struct {
		name            string
		useAnotherOwner bool
		useMissingOwner bool
		useAnotherBoard bool
		useMissingBoard bool
		wantErr         error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			if tt.useAnotherOwner {
				callerID = hierarchy.anotherBoard.OwnerID
			}
			if tt.useMissingOwner {
				callerID = hierarchy.missingBoard.OwnerID
			}
			if tt.useAnotherBoard {
				targetBoard = hierarchy.anotherBoard
			}
			if tt.useMissingBoard {
				targetBoard = hierarchy.missingBoard
			}

			err := r.Delete(context.Background(), callerID, targetBoard.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}

			want := []domain.Board{hierarchy.board, hierarchy.anotherBoard}
			if tt.wantErr == nil {
				want = []domain.Board{hierarchy.anotherBoard}
			}
			if diff := cmp.Diff(want, ListBoards(t, pool), testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("stored boards mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBoardRepository_GetAggregate(t *testing.T) {
	pool, r := boardRepoPrelude(t)

	tests := []struct {
		name            string
		useAnotherOwner bool
		useMissingOwner bool
		useAnotherBoard bool
		useMissingBoard bool
		wantErr         error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			if tt.useAnotherOwner {
				callerID = hierarchy.anotherBoard.OwnerID
			}
			if tt.useMissingOwner {
				callerID = hierarchy.missingBoard.OwnerID
			}
			if tt.useAnotherBoard {
				targetBoard = hierarchy.anotherBoard
			}
			if tt.useMissingBoard {
				targetBoard = hierarchy.missingBoard
			}

			gotBoard, gotColumns, gotTasks, err := r.GetAggregate(context.Background(), callerID, targetBoard.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("GetAggregate() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(hierarchy.board, gotBoard, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("board mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff([]domain.Column{hierarchy.column}, gotColumns, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("columns mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff([]domain.Task{hierarchy.task}, gotTasks, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("tasks mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func boardRepoPrelude(t *testing.T) (*pgxpool.Pool, *repository.PGBoard) {
	t.Helper()

	pool := testutil.SetupPostgres(t, "../../migrations")
	t.Cleanup(func() { pool.Close() })

	return pool, repository.NewPGBoard(pool)
}
