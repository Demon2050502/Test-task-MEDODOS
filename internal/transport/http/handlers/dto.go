package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Status      taskdomain.Status   `json:"status"`
	Priority    taskdomain.Priority `json:"priority"`
}

type taskDTO struct {
	ID                    int64               `json:"id"`
	Title                 string              `json:"title"`
	Description           string              `json:"description"`
	Status                taskdomain.Status   `json:"status"`
	Priority              taskdomain.Priority `json:"priority"`
	Queue                 taskdomain.Queue    `json:"queue"`
	ScheduledFor          *string             `json:"scheduled_for,omitempty"`
	SourceRecurringTaskID *int64              `json:"source_recurring_task_id,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

type taskQueueDTO struct {
	UrgentQueue  []taskDTO `json:"urgent_queue"`
	RegularQueue []taskDTO `json:"regular_queue"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	var scheduledFor *string
	if task.ScheduledFor != nil {
		value := formatDate(*task.ScheduledFor)
		scheduledFor = &value
	}

	return taskDTO{
		ID:                    task.ID,
		Title:                 task.Title,
		Description:           task.Description,
		Status:                task.Status,
		Priority:              task.Priority,
		Queue:                 task.Queue(),
		ScheduledFor:          scheduledFor,
		SourceRecurringTaskID: task.SourceRecurringTaskID,
		CreatedAt:             task.CreatedAt,
		UpdatedAt:             task.UpdatedAt,
	}
}
