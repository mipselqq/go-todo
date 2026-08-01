//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
	"goroutine/internal/testutil"
)

func TestTaskRepository_Create(t *testing.T) {
	pool, r := taskRepoPrelude(t)

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
			emptyColumn := testutil.NewValidColumn(t, hierarchy.board.ID, "Empty", 2)
			CreateColumn(t, pool, &emptyColumn)
			validTask := testutil.ValidTask(emptyColumn.ID)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := emptyColumn
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

			task, err := r.Create(context.Background(), callerID, targetBoard.ID, targetColumn.ID, validTask.Name, validTask.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got := ListTasksByColumnID(t, pool, emptyColumn.ID); len(got) != 0 {
					t.Errorf("got %d tasks, want 0", len(got))
				}
				return
			}

			if task.ID.IsNil() {
				t.Error("got empty task id, want generated id")
			}
			if task.ColumnID != emptyColumn.ID {
				t.Errorf("got columnID %q, want %q", task.ColumnID, emptyColumn.ID)
			}
			if task.Name != validTask.Name {
				t.Errorf("got name %q, want %q", task.Name, validTask.Name)
			}
			if task.Description != validTask.Description {
				t.Errorf("got description %q, want %q", task.Description, validTask.Description)
			}
			if task.Position.Int64() != 1 {
				t.Errorf("got position %d, want 1", task.Position.Int64())
			}
			if task.CreatedAt.IsZero() {
				t.Errorf("got zero createdAt, want set value")
			}
			if task.UpdatedAt.IsZero() {
				t.Errorf("got zero updatedAt, want set value")
			}
			if !task.CreatedAt.Equal(task.UpdatedAt) {
				t.Errorf("got createdAt=%v updatedAt=%v, want equal", task.CreatedAt, task.UpdatedAt)
			}
			AssertTimestampPrecisionAtLeastMillis(t, pool, "tasks", "created_at", "updated_at")

			storedTasks := ListTasksByColumnID(t, pool, emptyColumn.ID)
			if len(storedTasks) != 1 {
				t.Fatalf("ListTasksByColumnID() returned %d tasks, want exactly 1", len(storedTasks))
			}
			if diff := cmp.Diff(task, storedTasks[0], testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("got stored task mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskRepository_Create_AppendsPosition(t *testing.T) {
	pool, r := taskRepoPrelude(t)

	testutil.TruncateAllTables(t, pool)

	hierarchy := insertBoardHierarchy(t, pool)
	column := hierarchy.column

	toCreate := testutil.NewValidTask(t, column.ID, "Second", "Second description", 2)

	second, err := r.Create(
		context.Background(),
		hierarchy.board.OwnerID,
		hierarchy.board.ID,
		column.ID,
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

func TestTaskRepository_ListByColumnID(t *testing.T) {
	pool, r := taskRepoPrelude(t)

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
		{name: "Success empty"},
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
			emptyColumn := testutil.NewValidColumn(t, hierarchy.board.ID, "Empty", 2)
			CreateColumn(t, pool, &emptyColumn)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := emptyColumn
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

			tasks, err := r.ListByColumnID(context.Background(), callerID, targetBoard.ID, targetColumn.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ListByColumnID() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && len(tasks) != 0 {
				t.Errorf("got %d tasks, want 0", len(tasks))
			}
		})
	}

	t.Run("Success ordered and filtered by column", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		columnA := hierarchy.column
		columnB := testutil.NewValidColumn(t, board.ID, "In Progress", 2)
		CreateColumn(t, pool, &columnB)

		second := testutil.NewValidTask(t, columnA.ID, "Second", "second", 2)
		otherColumnTask := testutil.NewValidTask(t, columnB.ID, "Other", "other", 1)

		CreateTask(t, pool, &second)
		CreateTask(t, pool, &otherColumnTask)

		got, err := r.ListByColumnID(context.Background(), board.OwnerID, board.ID, columnA.ID)
		if err != nil {
			t.Fatalf("ListByColumnID() error = %v", err)
		}

		want := []domain.Task{hierarchy.task, second}
		if diff := cmp.Diff(want, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("ListByColumnID() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestTaskRepository_Get(t *testing.T) {
	pool, r := taskRepoPrelude(t)

	tests := []struct {
		name                      string
		useAnotherOwner           bool
		useMissingOwner           bool
		useAnotherBoard           bool
		useMissingBoard           bool
		useAnotherColumn          bool
		useColumnFromAnotherBoard bool
		useMissingColumn          bool
		useAnotherTask            bool
		useMissingTask            bool
		wantErr                   error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Another column", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useColumnFromAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Task from another column", useAnotherTask: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing task", useMissingTask: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			var sameBoardColumn domain.Column
			var sameBoardTask domain.Task
			if tt.useAnotherColumn || tt.useAnotherTask {
				sameBoardColumn = testutil.NewValidColumn(t, hierarchy.board.ID, "Another", 2)
				CreateColumn(t, pool, &sameBoardColumn)
			}
			if tt.useAnotherTask {
				sameBoardTask = testutil.NewValidTask(t, sameBoardColumn.ID, "Another", "another", 1)
				CreateTask(t, pool, &sameBoardTask)
			}

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := hierarchy.column
			targetTask := hierarchy.task
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
				targetColumn = sameBoardColumn
			}
			if tt.useColumnFromAnotherBoard {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}
			if tt.useAnotherTask {
				targetTask = sameBoardTask
			}
			if tt.useMissingTask {
				targetTask = hierarchy.missingTask
			}

			got, err := r.Get(context.Background(), callerID, targetBoard.ID, targetColumn.ID, targetTask.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if diff := cmp.Diff(hierarchy.task, got, testutil.CmpAllowUnexported()); diff != "" {
					t.Errorf("Get() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestTaskRepository_Update(t *testing.T) {
	pool, r := taskRepoPrelude(t)

	assertUpdatedTask := func(t *testing.T, got domain.Task, want domain.Task) {
		t.Helper()

		if got.ID != want.ID {
			t.Errorf("got id %q, want %q", got.ID, want.ID)
		}
		if got.ColumnID != want.ColumnID {
			t.Errorf("got columnID %q, want %q", got.ColumnID, want.ColumnID)
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
		AssertTimestampPrecisionAtLeastMillis(t, pool, "tasks", "created_at", "updated_at")

		storedTasks := ListTasksByColumnID(t, pool, want.ColumnID)
		if len(storedTasks) != 1 {
			t.Fatalf("ListTasksByColumnID() returned %d tasks, want exactly 1", len(storedTasks))
		}
		if diff := cmp.Diff(got, storedTasks[0], testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("got stored task mismatch (-want +got):\n%s", diff)
		}
	}

	tests := []struct {
		name                      string
		useAnotherOwner           bool
		useMissingOwner           bool
		useAnotherBoard           bool
		useMissingBoard           bool
		useAnotherColumn          bool
		useColumnFromAnotherBoard bool
		useMissingColumn          bool
		useAnotherTask            bool
		useMissingTask            bool
		wantErr                   error
	}{
		{name: "Success"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Another column", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useColumnFromAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Task from another column", useAnotherTask: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing task", useMissingTask: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			var sameBoardColumn domain.Column
			var sameBoardTask domain.Task
			if tt.useAnotherColumn || tt.useAnotherTask {
				sameBoardColumn = testutil.NewValidColumn(t, hierarchy.board.ID, "Another", 2)
				CreateColumn(t, pool, &sameBoardColumn)
			}
			if tt.useAnotherTask {
				sameBoardTask = testutil.NewValidTask(t, sameBoardColumn.ID, "Another", "another", 1)
				CreateTask(t, pool, &sameBoardTask)
			}
			want := testutil.UpdateValidTask(t, &hierarchy.task, "Renamed", "Renamed description", hierarchy.task.UpdatedAt)

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := hierarchy.column
			targetTask := hierarchy.task
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
				targetColumn = sameBoardColumn
			}
			if tt.useColumnFromAnotherBoard {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}
			if tt.useAnotherTask {
				targetTask = sameBoardTask
			}
			if tt.useMissingTask {
				targetTask = hierarchy.missingTask
			}

			got, err := r.Update(context.Background(), callerID, targetBoard.ID, targetColumn.ID, targetTask.ID, &want.Name, &want.Description)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				assertUpdatedTask(t, got, want)
				return
			}

			if diff := cmp.Diff([]domain.Task{hierarchy.task}, ListTasksByColumnID(t, pool, hierarchy.column.ID), testutil.CmpAllowUnexported()); diff != "" {
				t.Errorf("stored tasks mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("Success no changes", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)

		got, err := r.Update(
			context.Background(),
			hierarchy.board.OwnerID,
			hierarchy.board.ID,
			hierarchy.column.ID,
			hierarchy.task.ID,
			nil,
			nil,
		)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if diff := cmp.Diff(hierarchy.task, got, testutil.CmpAllowUnexported()); diff != "" {
			t.Errorf("Update() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestTaskRepository_Move(t *testing.T) {
	pool, r := taskRepoPrelude(t)

	tests := []struct {
		name                            string
		useAnotherOwner                 bool
		useMissingOwner                 bool
		useAnotherBoard                 bool
		useMissingBoard                 bool
		useAnotherSourceColumn          bool
		useSourceColumnFromAnotherBoard bool
		useMissingSourceColumn          bool
		useAnotherTask                  bool
		useMissingTask                  bool
		useAnotherTargetColumn          bool
		useMissingTargetColumn          bool
		wantErr                         error
	}{
		{name: "Success move down within column"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Another source column", useAnotherSourceColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Source column from another board", useSourceColumnFromAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing source column", useMissingSourceColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Task from another column", useAnotherTask: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing task", useMissingTask: true, wantErr: repository.ErrRowNotFound},
		{name: "Target column from another board", useAnotherTargetColumn: true, wantErr: repository.ErrTargetRowNotFound},
		{name: "Missing target column", useMissingTargetColumn: true, wantErr: repository.ErrTargetRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			second := testutil.NewValidTask(t, hierarchy.column.ID, "Second", "second", 2)
			third := testutil.NewValidTask(t, hierarchy.column.ID, "Third", "third", 3)
			CreateTask(t, pool, &third)
			CreateTask(t, pool, &second)
			var sameBoardColumn domain.Column
			var sameBoardTask domain.Task
			if tt.useAnotherSourceColumn || tt.useAnotherTask {
				sameBoardColumn = testutil.NewValidColumn(t, hierarchy.board.ID, "Another", 2)
				CreateColumn(t, pool, &sameBoardColumn)
			}
			if tt.useAnotherTask {
				sameBoardTask = testutil.NewValidTask(t, sameBoardColumn.ID, "Another", "another", 1)
				CreateTask(t, pool, &sameBoardTask)
			}

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			currentColumn := hierarchy.column
			targetTask := hierarchy.task
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
			if tt.useAnotherSourceColumn {
				currentColumn = sameBoardColumn
			}
			if tt.useSourceColumnFromAnotherBoard {
				currentColumn = hierarchy.anotherColumn
			}
			if tt.useMissingSourceColumn {
				currentColumn = hierarchy.missingColumn
			}
			if tt.useAnotherTask {
				targetTask = sameBoardTask
			}
			if tt.useMissingTask {
				targetTask = hierarchy.missingTask
			}
			if tt.useAnotherTargetColumn {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingTargetColumn {
				targetColumn = hierarchy.missingColumn
			}

			targetPosition := testutil.NewValidTaskPosition(t, 3)
			gotColumn, gotPosition, err := r.Move(
				context.Background(),
				callerID,
				targetBoard.ID,
				currentColumn.ID,
				targetTask.ID,
				targetColumn.ID,
				targetPosition,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Move() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if gotColumn != hierarchy.column.ID {
					t.Errorf("Move() column = %v, want %v", gotColumn, hierarchy.column.ID)
				}
				if gotPosition != targetPosition {
					t.Errorf("Move() position = %v, want %v", gotPosition, targetPosition)
				}
			}

			got := ListTasksByColumnID(t, pool, hierarchy.column.ID)
			if len(got) != 3 {
				t.Fatalf("got %d tasks after move, want 3", len(got))
			}
			if tt.wantErr == nil {
				assertTaskIDAndPosition(t, &got[0], second.ID, 1)
				assertTaskIDAndPosition(t, &got[1], third.ID, 2)
				assertTaskIDAndPosition(t, &got[2], hierarchy.task.ID, 3)
				return
			}
			assertTaskIDAndPosition(t, &got[0], hierarchy.task.ID, 1)
			assertTaskIDAndPosition(t, &got[1], second.ID, 2)
			assertTaskIDAndPosition(t, &got[2], third.ID, 3)
		})
	}

	t.Run("Success move up within column", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		column := hierarchy.column
		second := testutil.NewValidTask(t, column.ID, "Second", "second", 2)
		third := testutil.NewValidTask(t, column.ID, "Third", "third", 3)

		CreateTask(t, pool, &second)
		CreateTask(t, pool, &third)

		targetPosition := testutil.NewValidTaskPosition(t, 1)

		gotColumn, gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, column.ID, third.ID, column.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotColumn != column.ID {
			t.Fatalf("Move() column = %v, want %v", gotColumn, column.ID)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		got := ListTasksByColumnID(t, pool, column.ID)
		if len(got) != 3 {
			t.Fatalf("got %d tasks after move, want 3", len(got))
		}
		assertTaskIDAndPosition(t, &got[0], third.ID, 1)
		assertTaskIDAndPosition(t, &got[1], hierarchy.task.ID, 2)
		assertTaskIDAndPosition(t, &got[2], second.ID, 3)
	})

	t.Run("Success no-op", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		column := hierarchy.column
		second := testutil.NewValidTask(t, column.ID, "Second", "second", 2)

		CreateTask(t, pool, &second)

		targetPosition := testutil.NewValidTaskPosition(t, 2)

		gotColumn, gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, column.ID, second.ID, column.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotColumn != column.ID {
			t.Fatalf("Move() column = %v, want %v", gotColumn, column.ID)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		got := ListTasksByColumnID(t, pool, column.ID)
		if len(got) != 2 {
			t.Fatalf("got %d tasks after no-op move, want 2", len(got))
		}
		assertTaskIDAndPosition(t, &got[0], hierarchy.task.ID, 1)
		assertTaskIDAndPosition(t, &got[1], second.ID, 2)
	})

	t.Run("Index out of bounds within column", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		column := hierarchy.column
		second := testutil.NewValidTask(t, column.ID, "Second", "second", 2)
		third := testutil.NewValidTask(t, column.ID, "Third", "third", 3)

		CreateTask(t, pool, &second)
		CreateTask(t, pool, &third)

		targetPosition := testutil.NewValidTaskPosition(t, 4)

		_, _, err := r.Move(context.Background(), board.OwnerID, board.ID, column.ID, second.ID, column.ID, targetPosition)
		if !errors.Is(err, repository.ErrIndexOutOfBounds) {
			t.Fatalf("Move() error = %v, want ErrIndexOutOfBounds", err)
		}

		got := ListTasksByColumnID(t, pool, column.ID)
		if len(got) != 3 {
			t.Fatalf("got %d tasks after failed move, want 3", len(got))
		}
		assertTaskIDAndPosition(t, &got[0], hierarchy.task.ID, 1)
		assertTaskIDAndPosition(t, &got[1], second.ID, 2)
		assertTaskIDAndPosition(t, &got[2], third.ID, 3)
	})

	t.Run("Success move across columns into middle", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		columnA := hierarchy.column
		a1 := hierarchy.task
		columnB := testutil.NewValidColumn(t, board.ID, "In Progress", 2)
		CreateColumn(t, pool, &columnB)

		a2 := testutil.NewValidTask(t, columnA.ID, "A2", "a2", 2)
		a3 := testutil.NewValidTask(t, columnA.ID, "A3", "a3", 3)

		b1 := testutil.NewValidTask(t, columnB.ID, "B1", "b1", 1)
		b2 := testutil.NewValidTask(t, columnB.ID, "B2", "b2", 2)

		CreateTask(t, pool, &a3)
		CreateTask(t, pool, &a2)
		CreateTask(t, pool, &b2)
		CreateTask(t, pool, &b1)

		targetPosition := testutil.NewValidTaskPosition(t, 2)

		gotColumn, gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, columnA.ID, a2.ID, columnB.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotColumn != columnB.ID {
			t.Fatalf("Move() column = %v, want %v", gotColumn, columnB.ID)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		gotA := ListTasksByColumnID(t, pool, columnA.ID)
		if len(gotA) != 2 {
			t.Fatalf("got %d tasks in source column after move, want 2", len(gotA))
		}
		assertTaskIDAndPosition(t, &gotA[0], a1.ID, 1)
		assertTaskIDAndPosition(t, &gotA[1], a3.ID, 2)

		gotB := ListTasksByColumnID(t, pool, columnB.ID)
		if len(gotB) != 3 {
			t.Fatalf("got %d tasks in target column after move, want 3", len(gotB))
		}
		assertTaskIDAndPosition(t, &gotB[0], b1.ID, 1)
		assertTaskIDAndPosition(t, &gotB[1], a2.ID, 2)
		assertTaskIDAndPosition(t, &gotB[2], b2.ID, 3)
	})

	t.Run("Success move across columns to append", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		columnA := hierarchy.column
		a1 := hierarchy.task
		columnB := testutil.NewValidColumn(t, board.ID, "Done", 2)
		CreateColumn(t, pool, &columnB)

		b1 := testutil.NewValidTask(t, columnB.ID, "B1", "b1", 1)

		CreateTask(t, pool, &b1)

		targetPosition := testutil.NewValidTaskPosition(t, 2)

		gotColumn, gotPosition, err := r.Move(context.Background(), board.OwnerID, board.ID, columnA.ID, a1.ID, columnB.ID, targetPosition)
		if err != nil {
			t.Fatalf("Move() error = %v", err)
		}
		if gotColumn != columnB.ID {
			t.Fatalf("Move() column = %v, want %v", gotColumn, columnB.ID)
		}
		if gotPosition != targetPosition {
			t.Fatalf("Move() position = %v, want %v", gotPosition, targetPosition)
		}

		gotA := ListTasksByColumnID(t, pool, columnA.ID)
		if len(gotA) != 0 {
			t.Fatalf("got %d tasks in source column after move, want 0", len(gotA))
		}

		gotB := ListTasksByColumnID(t, pool, columnB.ID)
		if len(gotB) != 2 {
			t.Fatalf("got %d tasks in target column after move, want 2", len(gotB))
		}
		assertTaskIDAndPosition(t, &gotB[0], b1.ID, 1)
		assertTaskIDAndPosition(t, &gotB[1], a1.ID, 2)
	})

	t.Run("Index out of bounds across columns", func(t *testing.T) {
		testutil.TruncateAllTables(t, pool)

		hierarchy := insertBoardHierarchy(t, pool)
		board := hierarchy.board
		columnA := hierarchy.column
		a1 := hierarchy.task
		columnB := testutil.NewValidColumn(t, board.ID, "Done", 2)
		CreateColumn(t, pool, &columnB)

		b1 := testutil.NewValidTask(t, columnB.ID, "B1", "b1", 1)

		CreateTask(t, pool, &b1)

		targetPosition := testutil.NewValidTaskPosition(t, 3)

		_, _, err := r.Move(context.Background(), board.OwnerID, board.ID, columnA.ID, a1.ID, columnB.ID, targetPosition)
		if !errors.Is(err, repository.ErrIndexOutOfBounds) {
			t.Fatalf("Move() error = %v, want ErrIndexOutOfBounds", err)
		}

		gotA := ListTasksByColumnID(t, pool, columnA.ID)
		if len(gotA) != 1 {
			t.Fatalf("got %d tasks in source column after failed move, want 1", len(gotA))
		}
		assertTaskIDAndPosition(t, &gotA[0], a1.ID, 1)

		gotB := ListTasksByColumnID(t, pool, columnB.ID)
		if len(gotB) != 1 {
			t.Fatalf("got %d tasks in target column after failed move, want 1", len(gotB))
		}
		assertTaskIDAndPosition(t, &gotB[0], b1.ID, 1)
	})
}

func TestTaskRepository_Delete(t *testing.T) {
	pool, r := taskRepoPrelude(t)

	tests := []struct {
		name                      string
		useAnotherOwner           bool
		useMissingOwner           bool
		useAnotherBoard           bool
		useMissingBoard           bool
		useAnotherColumn          bool
		useColumnFromAnotherBoard bool
		useMissingColumn          bool
		useAnotherTask            bool
		useMissingTask            bool
		wantErr                   error
	}{
		{name: "Success shift positions"},
		{name: "Another owner", useAnotherOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing owner", useMissingOwner: true, wantErr: repository.ErrRowNotFound},
		{name: "Another board", useAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing board", useMissingBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Another column", useAnotherColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Column from another board", useColumnFromAnotherBoard: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing column", useMissingColumn: true, wantErr: repository.ErrRowNotFound},
		{name: "Task from another column", useAnotherTask: true, wantErr: repository.ErrRowNotFound},
		{name: "Missing task", useMissingTask: true, wantErr: repository.ErrRowNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TruncateAllTables(t, pool)
			hierarchy := insertBoardHierarchy(t, pool)
			second := testutil.NewValidTask(t, hierarchy.column.ID, "Second", "second", 2)
			third := testutil.NewValidTask(t, hierarchy.column.ID, "Third", "third", 3)
			CreateTask(t, pool, &third)
			CreateTask(t, pool, &second)
			var sameBoardColumn domain.Column
			var sameBoardTask domain.Task
			if tt.useAnotherColumn || tt.useAnotherTask {
				sameBoardColumn = testutil.NewValidColumn(t, hierarchy.board.ID, "Another", 2)
				CreateColumn(t, pool, &sameBoardColumn)
			}
			if tt.useAnotherTask {
				sameBoardTask = testutil.NewValidTask(t, sameBoardColumn.ID, "Another", "another", 1)
				CreateTask(t, pool, &sameBoardTask)
			}

			callerID := hierarchy.board.OwnerID
			targetBoard := hierarchy.board
			targetColumn := hierarchy.column
			targetTask := second
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
				targetColumn = sameBoardColumn
			}
			if tt.useColumnFromAnotherBoard {
				targetColumn = hierarchy.anotherColumn
			}
			if tt.useMissingColumn {
				targetColumn = hierarchy.missingColumn
			}
			if tt.useAnotherTask {
				targetTask = sameBoardTask
			}
			if tt.useMissingTask {
				targetTask = hierarchy.missingTask
			}

			err := r.Delete(context.Background(), callerID, targetBoard.ID, targetColumn.ID, targetTask.ID)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Delete() error = %v, want %v", err, tt.wantErr)
			}

			got := ListTasksByColumnID(t, pool, hierarchy.column.ID)
			if tt.wantErr == nil {
				if len(got) != 2 {
					t.Fatalf("got %d tasks after delete, want 2", len(got))
				}
				assertTaskIDAndPosition(t, &got[0], hierarchy.task.ID, 1)
				assertTaskIDAndPosition(t, &got[1], third.ID, 2)
				return
			}
			if len(got) != 3 {
				t.Fatalf("got %d tasks after failed delete, want 3", len(got))
			}
			assertTaskIDAndPosition(t, &got[0], hierarchy.task.ID, 1)
			assertTaskIDAndPosition(t, &got[1], second.ID, 2)
			assertTaskIDAndPosition(t, &got[2], third.ID, 3)
		})
	}
}

func TestLockTaskColumns_BlocksSecondTransaction(t *testing.T) {
	pool, _ := taskRepoPrelude(t)
	testutil.TruncateAllTables(t, pool)

	hierarchy := insertBoardHierarchy(t, pool)

	beginTx := func(id string) pgx.Tx {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Fatalf("pool.Begin() tx%s error = %v", id, err)
		}
		return tx
	}

	setLockTimeoutMs := func(tx pgx.Tx, id string, ms int) {
		_, err := tx.Exec(context.Background(), fmt.Sprintf(`SET LOCAL lock_timeout = '%dms'`, ms))
		if err != nil {
			t.Fatalf("tx%s SET LOCAL lock_timeout error = %v", id, err)
		}
	}

	rollbackTx := func(tx pgx.Tx, id string) {
		err := tx.Rollback(context.Background())
		if err != nil {
			t.Fatalf("tx%s Rollback() error = %v", id, err)
		}
	}

	lockTaskColumns := func(tx pgx.Tx) error {
		return repository.LockTaskColumns(context.Background(), tx, hierarchy.board.OwnerID, hierarchy.board.ID, hierarchy.column.ID)
	}

	tx1 := beginTx("1")

	// 1. LockTaskColumns runs SELECT ... FOR UPDATE on the columns row for this board/column;
	//    that row lock blocks any other transaction from locking the same row until tx1 ends.
	err := lockTaskColumns(tx1)
	if err != nil {
		t.Fatalf("LockTaskColumns() tx1 error = %v", err)
	}

	tx2 := beginTx("2")

	// 2. tx2: the next LockTaskColumns will try the same FOR UPDATE on the same row. While tx1
	//    still holds the lock, PostgreSQL would wait forever; SET LOCAL lock_timeout limits that
	//    wait to ~100ms, then the statement fails with a lock timeout instead of hanging the test.
	setLockTimeoutMs(tx2, "2", 100)

	// 3. tx2 must fail to acquire the same lock while tx1 still holds it.
	err = lockTaskColumns(tx2)
	if err == nil {
		t.Fatal("second LockTaskColumns() unexpectedly succeeded while tx1 still held the lock")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("second LockTaskColumns: want wrapped *pgconn.PgError, got %T: %v", err, err)
	}
	if pgErr.Code != "55P03" {
		t.Fatalf("second LockTaskColumns: want SQLSTATE 55P03 (lock_not_available because of lock timeout), got %v", err)
	}

	// 4. Roll back tx2 after the lock wait timeout.
	rollbackTx(tx2, "2")
	// 5. Roll back tx1 to release the original lock.
	rollbackTx(tx1, "1")

	// 6. Start tx3 after the lock has been released.
	tx3 := beginTx("3")

	// 7. tx3 should now acquire the same lock successfully.
	err = lockTaskColumns(tx3)
	if err != nil {
		t.Fatalf("third LockTaskColumns() after release error = %v", err)
	}

	// 8. Clean up tx3.
	rollbackTx(tx3, "3")
}

func assertTaskIDAndPosition(t *testing.T, task *domain.Task, wantID domain.TaskID, wantPos int64) {
	t.Helper()

	if task.ID != wantID {
		t.Errorf("got id %q, want %q", task.ID, wantID)
	}
	if task.Position.Int64() != wantPos {
		t.Errorf("got position %d, want %d", task.Position.Int64(), wantPos)
	}
}

func taskRepoPrelude(t *testing.T) (*pgxpool.Pool, *repository.PGTask) {
	t.Helper()

	pool := testutil.SetupPostgres(t, "../../migrations")
	t.Cleanup(func() { pool.Close() })

	return pool, repository.NewPGTask(pool)
}
