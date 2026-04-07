package recurringtask

import (
	"context"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, recurringTask *recurringtaskdomain.RecurringTask) (*recurringtaskdomain.RecurringTask, error)
	GetByID(ctx context.Context, id int64) (*recurringtaskdomain.RecurringTask, error)
	Update(ctx context.Context, recurringTask *recurringtaskdomain.RecurringTask) (*recurringtaskdomain.RecurringTask, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]recurringtaskdomain.RecurringTask, error)
	SetLastGeneratedFor(ctx context.Context, id int64, date time.Time) error
}

type GeneratedTaskRepository interface {
	CreateGeneratedIfMissing(ctx context.Context, task *taskdomain.Task) (bool, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*recurringtaskdomain.RecurringTask, error)
	GetByID(ctx context.Context, id int64) (*recurringtaskdomain.RecurringTask, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*recurringtaskdomain.RecurringTask, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]recurringtaskdomain.RecurringTask, error)
	RunOnce(ctx context.Context) error
}

type CreateInput struct {
	Title       string
	Description string
	Priority    taskdomain.Priority
	StartDate   time.Time
	RuleType    recurringtaskdomain.RuleType
	EveryNDays  int
	DayOfMonth  int
	Parity      recurringtaskdomain.DayParity
	Dates       []time.Time
}

type UpdateInput struct {
	Title       string
	Description string
	Priority    taskdomain.Priority
	StartDate   time.Time
	RuleType    recurringtaskdomain.RuleType
	EveryNDays  int
	DayOfMonth  int
	Parity      recurringtaskdomain.DayParity
	Dates       []time.Time
}
