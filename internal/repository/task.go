package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"goroutine/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGTask struct {
	pgPool *pgxpool.Pool
}

func NewPGTask(pgPool *pgxpool.Pool) *PGTask {
	return &PGTask{pgPool: pgPool}
}

// LockTaskColumns acquires FOR UPDATE row locks on the given columns for boardID.
func LockTaskColumns(
	ctx context.Context,
	tx pgx.Tx,
	callerID domain.UserID,
	boardID domain.BoardID, // To ensure columns are locked in the same board.
	columnIDs ...domain.ColumnID,
) error {
	_, err := lockTaskColumns(ctx, tx, callerID, boardID, columnIDs...)
	return err
}

func lockTaskColumns(
	ctx context.Context,
	tx pgx.Tx,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnIDs ...domain.ColumnID,
) (map[domain.ColumnID]struct{}, error) {
	if len(columnIDs) == 0 {
		return nil, errors.New("BUG: LockTaskColumns called with no columns. Isn't column ID forgotten?")
	}

	seen := make(map[domain.ColumnID]struct{}, len(columnIDs))
	for _, columnID := range columnIDs {
		if _, ok := seen[columnID]; ok {
			return nil, errors.New("BUG: LockTaskColumns called so it locks the same column multiple times")
		}
		seen[columnID] = struct{}{}
	}

	// Deadlock protection in case same ids are passed in a different order:
	// if T1 locks A and then waits for B, while T2 already locked B and waits for A,
	// PostgreSQL will detect a deadlock and abort one transaction. Ordering makes all
	// callers acquire row locks in the same order.
	const lockColumnsQuery = `
		SELECT c.id
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		WHERE b.id = @board_id
		  AND b.owner_id = @caller_id
		  AND c.id = ANY(@column_ids)
		ORDER BY c.id
		FOR UPDATE OF c`

	rows, err := tx.Query(ctx, lockColumnsQuery, pgx.NamedArgs{
		"caller_id":  callerID,
		"board_id":   boardID,
		"column_ids": columnIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("lock task columns: %w", err)
	}
	defer rows.Close()

	locked := make(map[domain.ColumnID]struct{}, len(columnIDs))
	for rows.Next() {
		var rawColumnID uuid.UUID
		if err = rows.Scan(&rawColumnID); err != nil {
			return nil, fmt.Errorf("failed to scan locked column row: %w", err)
		}
		columnID, conversionErr := domain.NewColumnIDFromUUID(rawColumnID)
		if conversionErr != nil {
			return nil, fmt.Errorf("convert locked column id: %w", conversionErr)
		}
		locked[columnID] = struct{}{}
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("lock task columns rows final error: %w", err)
	}
	if len(locked) != len(columnIDs) {
		return locked, ErrRowNotFound
	}

	return locked, nil
}

func (r *PGTask) Create(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	name domain.TaskName,
	description domain.TaskDescription,
) (domain.Task, error) {
	const (
		beginQuery      = `BEGIN`
		lockColumnQuery = `
		SELECT 1
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		WHERE b.id = @board_id
		  AND b.owner_id = @caller_id
		  AND c.id = @column_id
		FOR UPDATE OF c`
		insertTaskQuery = `
		INSERT INTO tasks (column_id, name, description, position)
		SELECT
			c.id,
			@name,
			@description,
			COALESCE(MAX(t.position), 0) + 1
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		LEFT JOIN tasks t ON t.column_id = c.id
		WHERE b.id = @board_id
		  AND b.owner_id = @caller_id
		  AND c.id = @column_id
		GROUP BY c.id
		RETURNING id, column_id, name, description, position, created_at, updated_at`
		commitQuery = `COMMIT`
	)

	args := pgx.NamedArgs{
		"caller_id":   callerID,
		"board_id":    boardID,
		"column_id":   columnID,
		"name":        name,
		"description": description,
	}
	batch := &pgx.Batch{}
	batch.Queue(beginQuery)
	batch.Queue(lockColumnQuery, args)
	batch.Queue(insertTaskQuery, args)
	batch.Queue(commitQuery)

	results := r.pgPool.SendBatch(ctx, batch)
	defer func() {
		_ = results.Close()
	}()

	_, err := results.Exec()
	if err != nil {
		return domain.Task{}, fmt.Errorf("task repo: create begin tx: %v: %w", err, ErrInternal)
	}
	var locked int
	err = results.QueryRow().Scan(&locked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, ErrRowNotFound
		}
		return domain.Task{}, fmt.Errorf("task repo: create lock column: %v: %w", err, ErrInternal)
	}

	task, err := ScanTask(results.QueryRow())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, ErrRowNotFound
		}
		return domain.Task{}, fmt.Errorf("task repo: create insert: %v: %w", err, ErrInternal)
	}
	_, err = results.Exec()
	if err != nil {
		return domain.Task{}, fmt.Errorf("task repo: create commit: %v: %w", err, ErrInternal)
	}
	if err = results.Close(); err != nil {
		return domain.Task{}, fmt.Errorf("task repo: create close batch: %v: %w", err, ErrInternal)
	}

	return task, nil
}

func (r *PGTask) ListByColumnID(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
) ([]domain.Task, error) {
	const (
		columnQuery = `
		SELECT 1
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		WHERE b.id = @board_id
		  AND b.owner_id = @caller_id
		  AND c.id = @column_id`
		tasksQuery = `
		SELECT t.id, t.column_id, t.name, t.description, t.position, t.created_at, t.updated_at
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		JOIN tasks t ON t.column_id = c.id
		WHERE b.id = @board_id
		  AND b.owner_id = @caller_id
		  AND c.id = @column_id
		ORDER BY t.position ASC`
	)

	args := pgx.NamedArgs{
		"caller_id": callerID,
		"board_id":  boardID,
		"column_id": columnID,
	}
	batch := &pgx.Batch{}
	batch.Queue(columnQuery, args)
	batch.Queue(tasksQuery, args)

	results := r.pgPool.SendBatch(ctx, batch)
	defer func() {
		_ = results.Close()
	}()

	var found int
	err := results.QueryRow().Scan(&found)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRowNotFound
		}
		return nil, fmt.Errorf("task repo: list by column id check column: %v: %w", err, ErrInternal)
	}

	rows, err := results.Query()
	if err != nil {
		return nil, fmt.Errorf("task repo: list by column id: %v: %w", err, ErrInternal)
	}
	defer rows.Close()

	var result []domain.Task
	for rows.Next() {
		task, scanErr := ScanTask(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("task repo: list by column id: scan: %v: %w", scanErr, ErrInternal)
		}
		result = append(result, task)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("task repo: list by column id: rows final error: %v: %w", err, ErrInternal)
	}
	if err = results.Close(); err != nil {
		return nil, fmt.Errorf("task repo: list by column id close batch: %v: %w", err, ErrInternal)
	}

	return result, nil
}

func (r *PGTask) Get(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
) (domain.Task, error) {
	const query = `
		SELECT t.id, t.column_id, t.name, t.description, t.position, t.created_at, t.updated_at
		FROM boards b
		JOIN columns c ON c.board_id = b.id
		JOIN tasks t ON t.column_id = c.id
		WHERE b.id = $1
		  AND b.owner_id = $2
		  AND c.id = $3
		  AND t.id = $4`

	task, err := ScanTask(r.pgPool.QueryRow(ctx, query, boardID, callerID, columnID, taskID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, ErrRowNotFound
		}
		return domain.Task{}, fmt.Errorf("task repo: get: %v: %w", err, ErrInternal)
	}

	return task, nil
}

func (r *PGTask) Update(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
	name *domain.TaskName,
	description *domain.TaskDescription,
) (domain.Task, error) {
	const query = `
		UPDATE tasks t
		SET
			name = COALESCE($5, t.name),
			description = COALESCE($6, t.description),
			updated_at = CASE
				WHEN $5 IS NULL
				 AND $6 IS NULL
				THEN t.updated_at
				ELSE CURRENT_TIMESTAMP AT TIME ZONE 'UTC'
			END
		FROM columns c
		JOIN boards b ON b.id = c.board_id
		WHERE b.id = $2
		  AND b.owner_id = $1
		  AND c.id = $3
		  AND t.column_id = c.id
		  AND t.id = $4
		RETURNING t.id, t.column_id, t.name, t.description, t.position, t.created_at, t.updated_at`

	task, err := ScanTask(r.pgPool.QueryRow(ctx, query, callerID, boardID, columnID, taskID, name, description))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, ErrRowNotFound
		}
		return domain.Task{}, fmt.Errorf("task repo: update: %v: %w", err, ErrInternal)
	}

	return task, nil
}

func (r *PGTask) Move(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	currentColumnID domain.ColumnID,
	taskID domain.TaskID,
	targetColumnID domain.ColumnID,
	targetPosition domain.TaskPosition,
) (domain.ColumnID, domain.TaskPosition, error) {
	const (
		// 2. SET position order is not guaranteed, so we disable uniqueness constraint for this transaction.
		deferPositionConstraintQuery = `
		SET CONSTRAINTS tasks_column_id_position_key DEFERRED`

		// 3. Read the current position of the task we are moving in its source column.
		getCurrentPositionQuery = `
		SELECT position
		FROM tasks
		WHERE column_id = @current_column_id
		  AND id = @task_id`

		// 4. Read how many tasks the target column currently has to validate targetPosition.
		countTargetTasksQuery = `
		SELECT COUNT(*)
		FROM tasks
		WHERE column_id = @target_column_id`

		// 5a. Same column, moving down: shift neighbors from (current, target] one slot up.
		//     Example: moving 2 -> 5 means 3,4,5 become 2,3,4.
		moveNeighborsDownQuery = `
		UPDATE tasks
		SET position = position - 1
		WHERE column_id = @current_column_id
		  AND position > @current_position
		  AND position <= @target_position`

		// 5b. Same column, moving up: shift neighbors from [target, current) one slot down.
		//     Example: moving 5 -> 2 means 2,3,4 become 3,4,5.
		moveNeighborsUpQuery = `
		UPDATE tasks
		SET position = position + 1
		WHERE column_id = @current_column_id
		  AND position >= @target_position
		  AND position < @current_position`

		// 5c. Cross-column compaction of the source column: shift everything below the
		//     moved task one slot up to close the gap.
		compactSourceQuery = `
		UPDATE tasks
		SET position = position - 1
		WHERE column_id = @current_column_id
		  AND position > @current_position`

		// 5d. Cross-column slot opening in the target column: shift positions >= target
		//     one slot down to make room.
		openTargetSlotQuery = `
		UPDATE tasks
		SET position = position + 1
		WHERE column_id = @target_column_id
		  AND position >= @target_position`

		// 6a. Same-column move: place the task at the target position.
		moveTaskWithinColumnQuery = `
		UPDATE tasks
		SET position = @target_position
		WHERE id = @task_id`

		// 6b. Cross-column move: switch column_id and place the task at the target position.
		moveTaskAcrossColumnsQuery = `
		UPDATE tasks
		SET column_id = @target_column_id,
		    position = @target_position
		WHERE id = @task_id`
	)

	tx, err := r.pgPool.Begin(ctx)
	if err != nil {
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move begin tx: %v: %w", err, ErrInternal)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sameColumn := currentColumnID == targetColumnID

	// 1. Lock affected columns so concurrent operations can't interrupt the move.
	if sameColumn {
		err = LockTaskColumns(ctx, tx, callerID, boardID, currentColumnID)
	} else {
		var locked map[domain.ColumnID]struct{}
		locked, err = lockTaskColumns(ctx, tx, callerID, boardID, currentColumnID, targetColumnID)
		if errors.Is(err, ErrRowNotFound) {
			if _, currentFound := locked[currentColumnID]; currentFound {
				return domain.ColumnID{}, domain.TaskPosition{}, ErrTargetRowNotFound
			}
		}
	}
	if err != nil {
		if errors.Is(err, ErrRowNotFound) {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrRowNotFound
		}
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move lock columns: %v: %w", err, ErrInternal)
	}

	_, err = tx.Exec(ctx, deferPositionConstraintQuery)
	if err != nil {
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move defer position constraint: %v: %w", err, ErrInternal)
	}

	var currentPosition int64
	err = tx.QueryRow(ctx, getCurrentPositionQuery, pgx.NamedArgs{
		"current_column_id": currentColumnID,
		"task_id":           taskID,
	}).Scan(&currentPosition)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrRowNotFound
		}
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move get current position: %v: %w", err, ErrInternal)
	}

	targetPositionInt := targetPosition.Int64()

	var targetTasksCount int64
	err = tx.QueryRow(ctx, countTargetTasksQuery, pgx.NamedArgs{
		"target_column_id": targetColumnID,
	}).Scan(&targetTasksCount)
	if err != nil {
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move count target tasks: %v: %w", err, ErrInternal)
	}

	if sameColumn {
		// In the same column, moving the task does not grow the column, so the
		// upper bound is the current task count.
		if targetPositionInt > targetTasksCount {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrIndexOutOfBounds
		}
		if targetPositionInt == currentPosition {
			return currentColumnID, targetPosition, nil
		}

		moveNeighborsArgs := pgx.NamedArgs{
			"current_column_id": currentColumnID,
			"current_position":  currentPosition,
			"target_position":   targetPositionInt,
		}
		if currentPosition < targetPositionInt {
			_, err = tx.Exec(ctx, moveNeighborsDownQuery, moveNeighborsArgs)
			if err != nil {
				return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move neighbors down: %v: %w", err, ErrInternal)
			}
		} else {
			_, err = tx.Exec(ctx, moveNeighborsUpQuery, moveNeighborsArgs)
			if err != nil {
				return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move neighbors up: %v: %w", err, ErrInternal)
			}
		}

		_, err = tx.Exec(ctx, moveTaskWithinColumnQuery, pgx.NamedArgs{
			"task_id":         taskID,
			"target_position": targetPositionInt,
		})
		if err != nil {
			return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move task within column: %v: %w", err, ErrInternal)
		}
	} else {
		// Across columns the target column grows by one, so an append at
		// targetTasksCount+1 is valid.
		if targetPositionInt > targetTasksCount+1 {
			return domain.ColumnID{}, domain.TaskPosition{}, ErrIndexOutOfBounds
		}

		_, err = tx.Exec(ctx, compactSourceQuery, pgx.NamedArgs{
			"current_column_id": currentColumnID,
			"current_position":  currentPosition,
		})
		if err != nil {
			return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move compact source: %v: %w", err, ErrInternal)
		}

		_, err = tx.Exec(ctx, openTargetSlotQuery, pgx.NamedArgs{
			"target_column_id": targetColumnID,
			"target_position":  targetPositionInt,
		})
		if err != nil {
			return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move open target slot: %v: %w", err, ErrInternal)
		}

		_, err = tx.Exec(ctx, moveTaskAcrossColumnsQuery, pgx.NamedArgs{
			"task_id":          taskID,
			"target_column_id": targetColumnID,
			"target_position":  targetPositionInt,
		})
		if err != nil {
			return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move task across columns: %v: %w", err, ErrInternal)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return domain.ColumnID{}, domain.TaskPosition{}, fmt.Errorf("task repo: move commit: %v: %w", err, ErrInternal)
	}

	return targetColumnID, targetPosition, nil
}

func (r *PGTask) Delete(
	ctx context.Context,
	callerID domain.UserID,
	boardID domain.BoardID,
	columnID domain.ColumnID,
	taskID domain.TaskID,
) error {
	const (
		// 2. Defer the unique constraint until COMMIT for this transaction only.
		deferPositionConstraintQuery = `
		SET CONSTRAINTS tasks_column_id_position_key DEFERRED`

		// 3. Delete the target task and remember its position.
		deleteTaskQuery = `
		DELETE FROM tasks
		WHERE column_id = @column_id
		  AND id = @task_id
		RETURNING position`

		// 4. Close the gap left by the deleted task.
		compactTrailingTasksQuery = `
		UPDATE tasks
		SET position = position - 1
		WHERE column_id = @column_id
		  AND position > @deleted_position`
	)

	tx, err := r.pgPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("task repo: delete begin tx: %v: %w", err, ErrInternal)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. Lock affected columns so concurrent operations can't interrupt the delete.
	err = LockTaskColumns(ctx, tx, callerID, boardID, columnID)
	if err != nil {
		if errors.Is(err, ErrRowNotFound) {
			return ErrRowNotFound
		}
		return fmt.Errorf("task repo: delete lock column: %v: %w", err, ErrInternal)
	}

	_, err = tx.Exec(ctx, deferPositionConstraintQuery)
	if err != nil {
		return fmt.Errorf("task repo: delete defer position constraint: %v: %w", err, ErrInternal)
	}

	var deletedPosition int64
	err = tx.QueryRow(ctx, deleteTaskQuery, pgx.NamedArgs{
		"column_id": columnID,
		"task_id":   taskID,
	}).Scan(&deletedPosition)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRowNotFound
		}
		return fmt.Errorf("task repo: delete task: %v: %w", err, ErrInternal)
	}

	_, err = tx.Exec(ctx, compactTrailingTasksQuery, pgx.NamedArgs{
		"column_id":        columnID,
		"deleted_position": deletedPosition,
	})
	if err != nil {
		return fmt.Errorf("task repo: delete compact trailing tasks: %v: %w", err, ErrInternal)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("task repo: delete commit: %v: %w", err, ErrInternal)
	}

	return nil
}

func ScanTask(row interface{ Scan(...any) error }) (domain.Task, error) {
	var (
		rawID       uuid.UUID
		rawColumnID uuid.UUID
		rawName     string
		rawDesc     string
		rawPos      int64
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := row.Scan(&rawID, &rawColumnID, &rawName, &rawDesc, &rawPos, &createdAt, &updatedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}
	name, err := domain.NewTaskName(rawName)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: name: %v: %w", err, errDataCorrupted)
	}
	desc, err := domain.NewTaskDescription(rawDesc)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: description: %v: %w", err, errDataCorrupted)
	}
	pos, err := domain.NewTaskPosition(rawPos)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: position: %v: %w", err, errDataCorrupted)
	}
	id, err := domain.NewTaskIDFromUUID(rawID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: id: %v: %w", err, errDataCorrupted)
	}
	columnID, err := domain.NewColumnIDFromUUID(rawColumnID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: column id: %v: %w", err, errDataCorrupted)
	}
	return domain.Task{
		ID:          id,
		ColumnID:    columnID,
		Name:        name,
		Description: desc,
		Position:    pos,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
