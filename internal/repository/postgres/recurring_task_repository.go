package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type RecurringTaskRepository struct {
	pool *pgxpool.Pool
}

func NewRecurringTaskRepository(pool *pgxpool.Pool) *RecurringTaskRepository {
	return &RecurringTaskRepository{pool: pool}
}

func (r *RecurringTaskRepository) Create(ctx context.Context, recurringTask *recurringtaskdomain.RecurringTask) (*recurringtaskdomain.RecurringTask, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		INSERT INTO recurring_tasks (
			title,
			description,
			priority,
			start_date,
			rule_type,
			every_n_days,
			day_of_month,
			parity,
			last_generated_for,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, title, description, priority, start_date, rule_type, every_n_days, day_of_month, parity, last_generated_for, created_at, updated_at
	`

	row := tx.QueryRow(
		ctx,
		query,
		recurringTask.Title,
		recurringTask.Description,
		recurringTask.Priority,
		recurringTask.StartDate,
		recurringTask.Rule.Type,
		nullInt32(recurringTask.Rule.EveryNDays),
		nullInt32(recurringTask.Rule.DayOfMonth),
		nullString(string(recurringTask.Rule.Parity)),
		recurringTask.LastGeneratedFor,
		recurringTask.CreatedAt,
		recurringTask.UpdatedAt,
	)

	created, err := scanRecurringTask(row)
	if err != nil {
		return nil, err
	}

	if err := r.replaceSpecificDates(ctx, tx, created.ID, recurringTask.Rule.Dates); err != nil {
		return nil, err
	}

	created.Rule.Dates = cloneDates(recurringTask.Rule.Dates)
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return created, nil
}

func (r *RecurringTaskRepository) GetByID(ctx context.Context, id int64) (*recurringtaskdomain.RecurringTask, error) {
	const query = `
		SELECT id, title, description, priority, start_date, rule_type, every_n_days, day_of_month, parity, last_generated_for, created_at, updated_at
		FROM recurring_tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	recurringTask, err := scanRecurringTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurringtaskdomain.ErrNotFound
		}

		return nil, err
	}

	recurringTask.Rule.Dates, err = r.listSpecificDates(ctx, id)
	if err != nil {
		return nil, err
	}

	return recurringTask, nil
}

func (r *RecurringTaskRepository) Update(ctx context.Context, recurringTask *recurringtaskdomain.RecurringTask) (*recurringtaskdomain.RecurringTask, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	const query = `
		UPDATE recurring_tasks
		SET title = $1,
			description = $2,
			priority = $3,
			start_date = $4,
			rule_type = $5,
			every_n_days = $6,
			day_of_month = $7,
			parity = $8,
			last_generated_for = $9,
			updated_at = $10
		WHERE id = $11
		RETURNING id, title, description, priority, start_date, rule_type, every_n_days, day_of_month, parity, last_generated_for, created_at, updated_at
	`

	row := tx.QueryRow(
		ctx,
		query,
		recurringTask.Title,
		recurringTask.Description,
		recurringTask.Priority,
		recurringTask.StartDate,
		recurringTask.Rule.Type,
		nullInt32(recurringTask.Rule.EveryNDays),
		nullInt32(recurringTask.Rule.DayOfMonth),
		nullString(string(recurringTask.Rule.Parity)),
		recurringTask.LastGeneratedFor,
		recurringTask.UpdatedAt,
		recurringTask.ID,
	)

	updated, err := scanRecurringTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurringtaskdomain.ErrNotFound
		}

		return nil, err
	}

	if err := r.replaceSpecificDates(ctx, tx, recurringTask.ID, recurringTask.Rule.Dates); err != nil {
		return nil, err
	}

	updated.Rule.Dates = cloneDates(recurringTask.Rule.Dates)
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return updated, nil
}

func (r *RecurringTaskRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM recurring_tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return recurringtaskdomain.ErrNotFound
	}

	return nil
}

func (r *RecurringTaskRepository) List(ctx context.Context) ([]recurringtaskdomain.RecurringTask, error) {
	const query = `
		SELECT id, title, description, priority, start_date, rule_type, every_n_days, day_of_month, parity, last_generated_for, created_at, updated_at
		FROM recurring_tasks
		ORDER BY CASE WHEN priority = 'urgent' THEN 0 ELSE 1 END, id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	recurringTasks := make([]recurringtaskdomain.RecurringTask, 0)
	for rows.Next() {
		recurringTask, err := scanRecurringTask(rows)
		if err != nil {
			return nil, err
		}

		recurringTask.Rule.Dates, err = r.listSpecificDates(ctx, recurringTask.ID)
		if err != nil {
			return nil, err
		}

		recurringTasks = append(recurringTasks, *recurringTask)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recurringTasks, nil
}

func (r *RecurringTaskRepository) SetLastGeneratedFor(ctx context.Context, id int64, date time.Time) error {
	const query = `
		UPDATE recurring_tasks
		SET last_generated_for = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.pool.Exec(ctx, query, date, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return recurringtaskdomain.ErrNotFound
	}

	return nil
}

func (r *RecurringTaskRepository) replaceSpecificDates(ctx context.Context, tx pgx.Tx, recurringTaskID int64, dates []time.Time) error {
	if _, err := tx.Exec(ctx, `DELETE FROM recurring_task_dates WHERE recurring_task_id = $1`, recurringTaskID); err != nil {
		return err
	}

	for _, date := range dates {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO recurring_task_dates (recurring_task_id, scheduled_date) VALUES ($1, $2)`,
			recurringTaskID,
			date,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r *RecurringTaskRepository) listSpecificDates(ctx context.Context, recurringTaskID int64) ([]time.Time, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT scheduled_date FROM recurring_task_dates WHERE recurring_task_id = $1 ORDER BY scheduled_date ASC`,
		recurringTaskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := make([]time.Time, 0)
	for rows.Next() {
		var scheduledDate time.Time
		if err := rows.Scan(&scheduledDate); err != nil {
			return nil, err
		}

		dates = append(dates, scheduledDate.UTC())
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dates, nil
}

type recurringTaskScanner interface {
	Scan(dest ...any) error
}

func scanRecurringTask(scanner recurringTaskScanner) (*recurringtaskdomain.RecurringTask, error) {
	var (
		recurringTask    recurringtaskdomain.RecurringTask
		priority         string
		ruleType         string
		everyNDays       sql.NullInt32
		dayOfMonth       sql.NullInt32
		parity           sql.NullString
		lastGeneratedFor sql.NullTime
	)

	if err := scanner.Scan(
		&recurringTask.ID,
		&recurringTask.Title,
		&recurringTask.Description,
		&priority,
		&recurringTask.StartDate,
		&ruleType,
		&everyNDays,
		&dayOfMonth,
		&parity,
		&lastGeneratedFor,
		&recurringTask.CreatedAt,
		&recurringTask.UpdatedAt,
	); err != nil {
		return nil, err
	}

	recurringTask.StartDate = recurringTask.StartDate.UTC()
	recurringTask.Priority = taskdomain.Priority(priority)
	recurringTask.Rule.Type = recurringtaskdomain.RuleType(ruleType)
	if everyNDays.Valid {
		recurringTask.Rule.EveryNDays = int(everyNDays.Int32)
	}

	if dayOfMonth.Valid {
		recurringTask.Rule.DayOfMonth = int(dayOfMonth.Int32)
	}

	if parity.Valid {
		recurringTask.Rule.Parity = recurringtaskdomain.DayParity(parity.String)
	}

	if lastGeneratedFor.Valid {
		value := lastGeneratedFor.Time.UTC()
		recurringTask.LastGeneratedFor = &value
	}

	return &recurringTask, nil
}

func cloneDates(dates []time.Time) []time.Time {
	if len(dates) == 0 {
		return nil
	}

	cloned := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		cloned = append(cloned, date.UTC())
	}

	return cloned
}

func nullInt32(value int) any {
	if value == 0 {
		return nil
	}

	return value
}

func nullString(value string) any {
	if value == "" {
		return nil
	}

	return value
}
