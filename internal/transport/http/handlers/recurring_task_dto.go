package handlers

import (
	"fmt"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
	recurringtaskusecase "example.com/taskservice/internal/usecase/recurringtask"
)

type recurringTaskMutationDTO struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Priority    taskdomain.Priority `json:"priority"`
	Recurrence  recurrenceDTO       `json:"recurrence"`
}

type recurrenceDTO struct {
	Type       recurringtaskdomain.RuleType  `json:"type"`
	StartDate  string                        `json:"start_date"`
	EveryNDays int                           `json:"every_n_days,omitempty"`
	DayOfMonth int                           `json:"day_of_month,omitempty"`
	Parity     recurringtaskdomain.DayParity `json:"parity,omitempty"`
	Dates      []string                      `json:"dates,omitempty"`
}

type recurringTaskDTO struct {
	ID               int64               `json:"id"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Priority         taskdomain.Priority `json:"priority"`
	Recurrence       recurrenceDTO       `json:"recurrence"`
	LastGeneratedFor *string             `json:"last_generated_for,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

func (d recurringTaskMutationDTO) toCreateInput() (recurringtaskusecase.CreateInput, error) {
	startDate, err := parseDate(d.Recurrence.StartDate)
	if err != nil {
		return recurringtaskusecase.CreateInput{}, fmt.Errorf("invalid recurrence.start_date: %w", err)
	}

	dates, err := parseDates(d.Recurrence.Dates)
	if err != nil {
		return recurringtaskusecase.CreateInput{}, fmt.Errorf("invalid recurrence.dates: %w", err)
	}

	return recurringtaskusecase.CreateInput{
		Title:       d.Title,
		Description: d.Description,
		Priority:    d.Priority,
		StartDate:   startDate,
		RuleType:    d.Recurrence.Type,
		EveryNDays:  d.Recurrence.EveryNDays,
		DayOfMonth:  d.Recurrence.DayOfMonth,
		Parity:      d.Recurrence.Parity,
		Dates:       dates,
	}, nil
}

func (d recurringTaskMutationDTO) toUpdateInput() (recurringtaskusecase.UpdateInput, error) {
	input, err := d.toCreateInput()
	if err != nil {
		return recurringtaskusecase.UpdateInput{}, err
	}

	return recurringtaskusecase.UpdateInput(input), nil
}

func newRecurringTaskDTO(recurringTask *recurringtaskdomain.RecurringTask) recurringTaskDTO {
	var lastGeneratedFor *string
	if recurringTask.LastGeneratedFor != nil {
		value := formatDate(*recurringTask.LastGeneratedFor)
		lastGeneratedFor = &value
	}

	return recurringTaskDTO{
		ID:          recurringTask.ID,
		Title:       recurringTask.Title,
		Description: recurringTask.Description,
		Priority:    recurringTask.Priority,
		Recurrence: recurrenceDTO{
			Type:       recurringTask.Rule.Type,
			StartDate:  formatDate(recurringTask.StartDate),
			EveryNDays: recurringTask.Rule.EveryNDays,
			DayOfMonth: recurringTask.Rule.DayOfMonth,
			Parity:     recurringTask.Rule.Parity,
			Dates:      formatDates(recurringTask.Rule.Dates),
		},
		LastGeneratedFor: lastGeneratedFor,
		CreatedAt:        recurringTask.CreatedAt,
		UpdatedAt:        recurringTask.UpdatedAt,
	}
}

func parseDates(values []string) ([]time.Time, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]time.Time, 0, len(values))
	for _, value := range values {
		date, err := parseDate(value)
		if err != nil {
			return nil, err
		}

		result = append(result, date)
	}

	return result, nil
}

func formatDates(values []time.Time) []string {
	if len(values) == 0 {
		return nil
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, formatDate(value))
	}

	return result
}

func parseDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation(dateLayout, value, time.UTC)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC), nil
}

func formatDate(value time.Time) string {
	return value.UTC().Format(dateLayout)
}

const dateLayout = "2006-01-02"
