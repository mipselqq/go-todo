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

func TestColumnRepository_Create(t *testing.T) {
	pool, r := columnRepoPrelude(t)

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
			emptyBoard := testutil.ValidBoardForOwner(hierarchy.board.OwnerID)
			CreateBoard(t, pool, &emptyBoard)
			validColumn := testutil.ValidColumn(emptyBoard.ID)

			callerID := hierarchy.board.OwnerID
			targetBoard := emptyBoard
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

			column, err := r.Create(context.Background(), callerID, targetBoard.ID, validColumn.Name, validColumn.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got := ListColumnsByBoardID(t, pool, emptyBoard.ID); len(got) != 0 {
					t.Errorf("got %d columns, want 0", len(got))
				}
				return
			}

			if column.ID.IsNil() {
				t.Error("got empty column id, want generated id")
			}
			if column.BoardID != emptyBoard.ID {
				t.Errorf("got boardID %q, want %q", column.BoardID, emptyBoard.ID)
			}
			if column.Name != validColumn.Name {
				t.Errorf("got name %q, want %q", column.Name, validColumn.Name)
			}
			if column.Description != validColumn.Description {
				t.Errorf("got description %q, want %q", column.Description, validColumn.Description)
			}
			if column.Position.Int64() != 1 {
				t.Errorf("got position %d, want 1", column.Position.Int64())
			}
			if column.CreatedAt.IsZero() {
				t.Errorf("got zero createdAt, want set value")
			}
			if column.UpdatedAt.IsZero() {
				t.Errorf("got zero updatedAt, want set value")
			}
			if !column.CreatedAt.Equal(column.UpdatedAt) {
				t.Errorf("got createdAt=%v updatedAt=%v, want equal", column.CreatedAt, column.UpdatedAt)
			}
			AssertTimestampPrecisionAtLeastMillis(t, pool, "columns", "created_at", "updated_at")

			storedColumns := ListColumnsByBoardID(t, pool, emptyBoard.ID)
			if len(storedColumns) != 1 {
				t.Fatalf("ListColumnsByBoardID() returned %d columns, want exactly 1", len(storedColumns))
			}
			if diff := cmp.Diff(column, storedColumns[0], testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("got stored column mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestColumnRepository_Create_AppendsPosition(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	testutil.TruncateAllTables(t, pool)

	hierarchy := insertBoardHierarchy(t, pool)

	toCreate := testutil.NewValidColumn(t, hierarchy.board.ID, "Done", 2)

	second, err := r.Create(
		context.Background(),
		hierarchy.board.OwnerID,
		hierarchy.board.ID,
		toCreate.Name,
		toCreate.Description,
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if second.Position.Int64() != 2 {
		t.Errorf("got second position %d, want 2", second.Position.Int64())
	}
}

func TestColumnRepository_ListByBoardID(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	tests := []struct {
		name            string
		useAnotherOwner bool
		useMissingOwner bool
		useAnotherBoard bool
		useMissingBoard bool
		wantErr         error
	}{
		{name: "Success empty"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			emptyBoard := testutil.ValidBoardForOwner(hierarchy.board.OwnerID)
			CreateBoard(t, pool, &emptyBoard)

			callerID := hierarchy.board.OwnerID
			targetBoard := emptyBoard
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

			columns, err := r.ListByBoardID(context.Background(), callerID, targetBoard.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ListByBoardID() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && len(columns) != 0 {
				t.Errorf("got %d columns, want 0", len(columns))
			}
		})
	}

	t.Run("Success ordered and filtered by board", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)
		hierarchy := insertBoardHierarchy(t, pool)

		first := hierarchy.column
		second := testutil.NewValidColumn(t, hierarchy.board.ID, "In Progress", 2)

		CreateColumn(t, pool, &second)

		got, err := r.ListByBoardID(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID)
		if err != nil {
			t.Fatalf("ListByBoardID() error = %v", err)
		}

		want := []domain.Column{first, second}
		if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("ListByBoardID() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestColumnRepository_Get(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	tests := []struct {
		name             string
		useAnotherOwner  bool
		useMissingOwner  bool
		useAnotherBoard  bool
		useMissingBoard  bool
		useAnotherColumn bool
		useMissingColumn bool
		wantErr          error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := hierarchy.column
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
			if tt.useAnotherColumn {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}

			got, err := r.Get(context.Background(), callerID, targetBoard.ID, targetColumn.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(hierarchy.column, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Get() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestColumnRepository_Update(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	assertUpdatedColumn := func(t *testing.T, got domain.Column, want domain.Column) {
		t.Helper()

		if got.ID != want.ID {
			t.Errorf("got id %q, want %q", got.ID, want.ID)
		}
		if got.BoardID != want.BoardID {
			t.Errorf("got boardID %q, want %q", got.BoardID, want.BoardID)
		}
		if got.Name != want.Name {
			t.Errorf("got name %q, want %q", got.Name, want.Name)
		}
		if got.Description != want.Description {
			t.Errorf("got description %q, want %q", got.Description, want.Description)
		}
		if got.Position != want.Position {
			t.Errorf("got position %d, want %d", got.Position.Int64(), want.Position.Int64())
		}
		if !got.CreatedAt.Truncate(time.Millisecond).Equal(want.CreatedAt.Truncate(time.Millisecond)) {
			t.Errorf("got createdAt %v, want %v (at millisecond precision)", got.CreatedAt, want.CreatedAt)
		}
		if !got.UpdatedAt.After(want.UpdatedAt) {
			t.Errorf("got updatedAt %v, want after %v", got.UpdatedAt, want.UpdatedAt)
		}
		AssertTimestampPrecisionAtLeastMillis(t, pool, "columns", "created_at", "updated_at")

		storedColumns := ListColumnsByBoardID(t, pool, want.BoardID)
		if len(storedColumns) != 1 {
			t.Fatalf("ListColumnsByBoardID() returned %d columns, want exactly 1", len(storedColumns))
		}
		if diff := cmp.Diff(got, storedColumns[0], testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("got stored column mismatch (-want +got):\n%s", diff)
		}
	}

	tests := []struct {
		name             string
		useAnotherOwner  bool
		useMissingOwner  bool
		useAnotherBoard  bool
		useMissingBoard  bool
		useAnotherColumn bool
		useMissingColumn bool
		wantErr          error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			want := testutil.UpdateValidColumn(t, &hierarchy.column, "Renamed", hierarchy.column.Description.String(), hierarchy.column.UpdatedAt)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := hierarchy.column
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
			if tt.useAnotherColumn {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}

			got, err := r.Update(context.Background(), callerID, targetBoard.ID, targetColumn.ID, &want.Name, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				assertUpdatedColumn(t, got, want)
				return
			}

			if diff := cmp.Diff([]domain.Column{hierarchy.column}, ListColumnsByBoardID(t, pool, hierarchy.board.ID), testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("stored columns mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Success description only", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		column := hierarchy.column

		newDesc, err := domain.NewColumnDescription("Updated column body")
		if err != nil {
			t.Fatalf("NewColumnDescription() error = %v", err)
		}
		updated, err := r.Update(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID, column.ID, nil, &newDesc)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Name != column.Name {
			t.Errorf("got name %q, want %q", updated.Name, column.Name)
		}
		if updated.Description != newDesc {
			t.Errorf("got description %q, want %q", updated.Description, newDesc)
		}
		storedColumns := ListColumnsByBoardID(t, pool, column.BoardID)
		if len(storedColumns) != 1 {
			t.Fatalf("ListColumnsByBoardID() returned %d columns, want exactly 1", len(storedColumns))
		}
		storedColumn := storedColumns[0]
		if storedColumn.Description != newDesc {
			t.Errorf("stored description %q, want %q", storedColumn.Description, newDesc)
		}
	})

	t.Run("Success no changes", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)

		got, err := r.Update(context.Background(), hierarchy.board.OwnerID, hierarchy.board.ID, hierarchy.column.ID, nil, nil)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if diff := cmp.Diff(hierarchy.column, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("Update() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestColumnRepository_Move(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	tests := []struct {
		name             string
		useAnotherOwner  bool
		useMissingOwner  bool
		useAnotherBoard  bool
		useMissingBoard  bool
		useAnotherColumn bool
		useMissingColumn bool
		wantErr          error
	}{
		{name: "Success move down"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			first := hierarchy.column
			second := testutil.NewValidColumn(t, hierarchy.board.ID, "In Progress", 2)
			third := testutil.NewValidColumn(t, hierarchy.board.ID, "Done", 3)
			CreateColumn(t, pool, &third)
			CreateColumn(t, pool, &second)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := first
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
			if tt.useAnotherColumn {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}

			targetPosition := testutil.NewValidColumnPosition(t, 3)
			gotPosition, err := r.Move(context.Background(), callerID, targetBoard.ID, targetColumn.ID, targetPosition)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Move() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && gotPosition != targetPosition {
				t.Errorf("Move() position = %v, want %v", gotPosition, targetPosition)
			}

			got := ListColumnsByBoardID(t, pool, hierarchy.board.ID)
			if len(got) != 3 {
				t.Fatalf("got %d columns after move, want 3", len(got))
			}
			if tt.wantErr == nil {
				assertColumnIDAndPosition(t, &got[0], second.ID, 1)
				assertColumnIDAndPosition(t, &got[1], third.ID, 2)
				assertColumnIDAndPosition(t, &got[2], first.ID, 3)
				return
			}
			assertColumnIDAndPosition(t, &got[0], first.ID, 1)
			assertColumnIDAndPosition(t, &got[1], second.ID, 2)
			assertColumnIDAndPosition(t, &got[2], third.ID, 3)
		})
	}

	t.Run("Success move up", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		second := testutil.NewValidColumn(t, board.ID, "In Progress", 2)
		third := testutil.NewValidColumn(t, board.ID, "Done", 3)

		CreateColumn(t, pool, &second)
		CreateColumn(t, pool, &third)

		targetPosition := testutil.NewValidColumnPosition(t, 1)

		gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, third.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		got := ListColumnsByBoardID(t, pool, board.ID)
		if len(got) != 3 {
			t.Fatalf("got %d columns after move, want 3", len(got))
		}
		assertColumnIDAndPosition(t, &got[0], third.ID, 1)
		assertColumnIDAndPosition(t, &got[1], hierarchy.column.ID, 2)
		assertColumnIDAndPosition(t, &got[2], second.ID, 3)
	})

	t.Run("Success no-op", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		second := testutil.NewValidColumn(t, board.ID, "In Progress", 2)

		CreateColumn(t, pool, &second)

		targetPosition := testutil.NewValidColumnPosition(t, 2)

		gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, second.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		got := ListColumnsByBoardID(t, pool, board.ID)
		if len(got) != 2 {
			t.Fatalf("got %d columns after no-op move, want 2", len(got))
		}
		assertColumnIDAndPosition(t, &got[0], hierarchy.column.ID, 1)
		assertColumnIDAndPosition(t, &got[1], second.ID, 2)
	})

	t.Run("Index out of bounds", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		second := testutil.NewValidColumn(t, board.ID, "In Progress", 2)
		third := testutil.NewValidColumn(t, board.ID, "Done", 3)

		CreateColumn(t, pool, &second)
		CreateColumn(t, pool, &third)

		targetPosition := testutil.NewValidColumnPosition(t, 4)

		_, err := r.Move(context.Background(), board.OwnerID, board.ID, second.ID, targetPosition)
		if !errors.Is(err, repository.ErrIndexOutOfBounds) {
			t.Fatalf("Move() error = %v, want ErrIndexOutOfBounds", err)
		}

		got := ListColumnsByBoardID(t, pool, board.ID)
		if len(got) != 3 {
			t.Fatalf("got %d columns after failed move, want 3", len(got))
		}
		assertColumnIDAndPosition(t, &got[0], hierarchy.column.ID, 1)
		assertColumnIDAndPosition(t, &got[1], second.ID, 2)
		assertColumnIDAndPosition(t, &got[2], third.ID, 3)
	})
}

func TestColumnRepository_Delete(t *testing.T) {
	pool, r := columnRepoPrelude(t)

	tests := []struct {
		name             string
		useAnotherOwner  bool
		useMissingOwner  bool
		useAnotherBoard  bool
		useMissingBoard  bool
		useAnotherColumn bool
		useMissingColumn bool
		wantErr          error
	}{
		{name: "Success shift positions"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			first := hierarchy.column
			second := testutil.NewValidColumn(t, hierarchy.board.ID, "In Progress", 2)
			third := testutil.NewValidColumn(t, hierarchy.board.ID, "Done", 3)
			CreateColumn(t, pool, &third)
			CreateColumn(t, pool, &second)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := second
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
			if tt.useAnotherColumn {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}

			err := r.Delete(context.Background(), callerID, targetBoard.ID, targetColumn.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}

			got := ListColumnsByBoardID(t, pool, hierarchy.board.ID)
			if tt.wantErr == nil {
				if len(got) != 2 {
					t.Fatalf("got %d columns after delete, want 2", len(got))
				}
				assertColumnIDAndPosition(t, &got[0], first.ID, 1)
				assertColumnIDAndPosition(t, &got[1], third.ID, 2)
				return
			}
			if len(got) != 3 {
				t.Fatalf("got %d columns after failed delete, want 3", len(got))
			}
			assertColumnIDAndPosition(t, &got[0], first.ID, 1)
			assertColumnIDAndPosition(t, &got[1], second.ID, 2)
			assertColumnIDAndPosition(t, &got[2], third.ID, 3)
		})
	}
}

func assertColumnIDAndPosition(t *testing.T, col *domain.Column, wantID domain.ColumnID, wantPos int64) {
	t.Helper()

	if col.ID != wantID {
		t.Errorf("got id %q, want %q", col.ID, wantID)
	}
	if col.Position.Int64() != wantPos {
		t.Errorf("got position %d, want %d", col.Position.Int64(), wantPos)
	}
}

func columnRepoPrelude(t *testing.T) (*pgxpool.Pool, *repository.PGColumn) {
	t.Helper()

	pool := testutil.SetupPostgres(t, "../../migrations")
	t.Cleanup(func() { pool.Close() })

	return pool, repository.NewPGColumn(pool)
}
