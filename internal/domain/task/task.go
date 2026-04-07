package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityUrgent Priority = "urgent"
)

type Queue string

const (
	QueueRegular Queue = "regular"
	QueueUrgent  Queue = "urgent"
)

type Task struct {
	ID                    int64      `json:"id"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	Status                Status     `json:"status"`
	Priority              Priority   `json:"priority"`
	ScheduledFor          *time.Time `json:"scheduled_for,omitempty"`
	SourceRecurringTaskID *int64     `json:"source_recurring_task_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

func (p Priority) Valid() bool {
	switch p {
	case PriorityNormal, PriorityUrgent:
		return true
	default:
		return false
	}
}

func (t Task) Queue() Queue {
	if t.Priority == PriorityUrgent {
		return QueueUrgent
	}

	return QueueRegular
}
