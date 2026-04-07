package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (
			title,
			description,
			status,
			priority,
			scheduled_for,
			source_recurring_task_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, status, priority, scheduled_for, source_recurring_task_id, created_at, updated_at
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.ScheduledFor,
		task.SourceRecurringTaskID,
		task.CreatedAt,
		task.UpdatedAt,
	)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, priority, scheduled_for, source_recurring_task_id, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			priority = $4,
			updated_at = $5
		WHERE id = $6
		RETURNING id, title, description, status, priority, scheduled_for, source_recurring_task_id, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.Priority, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, priority, scheduled_for, source_recurring_task_id, created_at, updated_at
		FROM tasks
		ORDER BY CASE WHEN priority = 'urgent' THEN 0 ELSE 1 END,
			COALESCE(scheduled_for, DATE(created_at)) DESC,
			id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) CreateGeneratedIfMissing(ctx context.Context, task *taskdomain.Task) (bool, error) {
	if task.ScheduledFor == nil || task.SourceRecurringTaskID == nil {
		return false, fmt.Errorf("generated task requires scheduled_for and source_recurring_task_id")
	}

	const query = `
		INSERT INTO tasks (
			title,
			description,
			status,
			priority,
			scheduled_for,
			source_recurring_task_id,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (source_recurring_task_id, scheduled_for) DO NOTHING
		RETURNING id, title, description, status, priority, scheduled_for, source_recurring_task_id, created_at, updated_at
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		task.ScheduledFor,
		task.SourceRecurringTaskID,
		task.CreatedAt,
		task.UpdatedAt,
	)

	_, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task                  taskdomain.Task
		status                string
		priority              string
		scheduledFor          sql.NullTime
		sourceRecurringTaskID sql.NullInt64
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&priority,
		&scheduledFor,
		&sourceRecurringTaskID,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)
	task.Priority = taskdomain.Priority(priority)
	if scheduledFor.Valid {
		value := scheduledFor.Time.UTC()
		task.ScheduledFor = &value
	}

	if sourceRecurringTaskID.Valid {
		value := sourceRecurringTaskID.Int64
		task.SourceRecurringTaskID = &value
	}

	return &task, nil
}
