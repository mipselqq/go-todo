package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"goroutine/internal/domain"
	"goroutine/internal/repository"
	"goroutine/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateFixedUser(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	id := testutil.ValidUserID()
	domainEmail := testutil.ValidEmail()
	CreateUser(t, pool, id, domainEmail, testutil.ValidPasswordHash())
}

func CreateUser(t *testing.T, pool *pgxpool.Pool, id domain.UserID, email domain.Email, hash domain.PasswordHash) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		INSERT INTO users (id, email, password_hash)
		VALUES ($1, $2, $3)`
	_, err := pool.Exec(ctx, query, id, email, hash.RevealSecret())
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
}

func CreateBoard(t *testing.T, pool *pgxpool.Pool, board *domain.Board) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		INSERT INTO boards (id, owner_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := pool.Exec(
		ctx, query,
		board.ID,
		board.OwnerID,
		board.Name,
		board.Description,
		board.CreatedAt,
		board.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateBoard() error = %v", err)
	}
}

func ListBoards(t *testing.T, pool *pgxpool.Pool) []domain.Board {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		SELECT id, owner_id, name, description, created_at, updated_at
		FROM boards
		ORDER BY created_at ASC`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		t.Fatalf("ListBoards() error = %v", err)
	}
	defer rows.Close()

	var boards []domain.Board
	for rows.Next() {
		board, scanErr := repository.ScanBoard(rows)
		if scanErr != nil {
			t.Fatalf("Board row Scan() error = %v", scanErr)
		}
		boards = append(boards, board)
	}

	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}

	return boards
}

func CreateColumn(t *testing.T, pool *pgxpool.Pool, column *domain.Column) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		INSERT INTO columns (id, board_id, name, description, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := pool.Exec(
		ctx, query,
		column.ID,
		column.BoardID,
		column.Name,
		column.Description,
		column.Position,
		column.CreatedAt,
		column.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
}

func ListColumnsByBoardID(t *testing.T, pool *pgxpool.Pool, boardID domain.BoardID) []domain.Column {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		SELECT id, board_id, name, description, position, created_at, updated_at
		FROM columns
		WHERE board_id = $1
		ORDER BY position ASC`

	rows, err := pool.Query(ctx, query, boardID)
	if err != nil {
		t.Fatalf("ListColumnsByBoardID() error = %v", err)
	}
	defer rows.Close()

	var columns []domain.Column
	for rows.Next() {
		column, scanErr := repository.ScanColumn(rows)
		if scanErr != nil {
			t.Fatalf("Column row Scan() error = %v", scanErr)
		}

		columns = append(columns, column)
	}

	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}

	return columns
}

func CreateTask(t *testing.T, pool *pgxpool.Pool, task *domain.Task) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		INSERT INTO tasks (id, column_id, name, description, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := pool.Exec(
		ctx, query,
		task.ID,
		task.ColumnID,
		task.Name,
		task.Description,
		task.Position,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
}

type testBoardHierarchy struct {
	board         domain.Board
	column        domain.Column
	task          domain.Task
	anotherBoard  domain.Board
	anotherColumn domain.Column
	anotherTask   domain.Task
	missingBoard  domain.Board
	missingColumn domain.Column
	missingTask   domain.Task
}

func insertBoardHierarchy(t *testing.T, pool *pgxpool.Pool) testBoardHierarchy {
	t.Helper()

	board := testutil.ValidBoard()
	column := testutil.ValidColumn(board.ID)
	task := testutil.ValidTask(column.ID)

	anotherBoard := testutil.ValidBoardForOwner(domain.NewUserID())
	anotherColumn := testutil.ValidColumn(anotherBoard.ID)
	anotherTask := testutil.ValidTask(anotherColumn.ID)

	CreateFixedUser(t, pool)
	insertAnotherUser(t, pool, anotherBoard.OwnerID)
	CreateBoard(t, pool, &board)
	CreateColumn(t, pool, &column)
	CreateTask(t, pool, &task)
	CreateBoard(t, pool, &anotherBoard)
	CreateColumn(t, pool, &anotherColumn)
	CreateTask(t, pool, &anotherTask)

	missingBoard := testutil.ValidBoardForOwner(domain.NewUserID())
	missingColumn := testutil.ValidColumn(missingBoard.ID)
	missingTask := testutil.ValidTask(missingColumn.ID)

	return testBoardHierarchy{
		board:         board,
		column:        column,
		task:          task,
		anotherBoard:  anotherBoard,
		anotherColumn: anotherColumn,
		anotherTask:   anotherTask,
		missingBoard:  missingBoard,
		missingColumn: missingColumn,
		missingTask:   missingTask,
	}
}

func ListTasksByColumnID(t *testing.T, pool *pgxpool.Pool, columnID domain.ColumnID) []domain.Task {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		SELECT id, column_id, name, description, position, created_at, updated_at
		FROM tasks
		WHERE column_id = $1
		ORDER BY position ASC`

	rows, err := pool.Query(ctx, query, columnID)
	if err != nil {
		t.Fatalf("ListTasksByColumnID() error = %v", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		task, scanErr := repository.ScanTask(rows)
		if scanErr != nil {
			t.Fatalf("Task row Scan() error = %v", scanErr)
		}

		tasks = append(tasks, task)
	}

	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}

	return tasks
}

func insertAnotherUser(t *testing.T, pool *pgxpool.Pool, userID domain.UserID) {
	t.Helper()

	email, err := domain.NewEmail("another@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}
	CreateUser(t, pool, userID, email, testutil.ValidPasswordHash())
}

func AssertTimestampPrecisionAtLeastMillis(t *testing.T, pool *pgxpool.Pool, tableName string, columnNames ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		SELECT datetime_precision
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		    AND table_name = $1
		    AND column_name = $2`

	for _, columnName := range columnNames {
		var precision int32
		err := pool.QueryRow(ctx, query, tableName, columnName).Scan(&precision)
		if err != nil {
			t.Fatalf("timestamp precision lookup for %s.%s: %v", tableName, columnName, err)
		}
		if precision < 3 {
			t.Errorf("got datetime_precision=%d for %s.%s, want >= 3", precision, tableName, columnName)
		}
	}
}

func ListUsers(t *testing.T, pool *pgxpool.Pool) []domain.User {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
		SELECT id, email, password_hash, telegram_chat_id, telegram_username
		FROM users
		ORDER BY id`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		user, scanErr := repository.ScanUser(rows)
		if scanErr != nil {
			t.Fatalf("User row Scan() error = %v", scanErr)
		}
		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}

	return users
}

func assertErrRowNotFound(t *testing.T, err error) {
	t.Helper()

	if !errors.Is(err, repository.ErrRowNotFound) {
		t.Errorf("got error %v, want ErrRowNotFound", err)
	}
}
