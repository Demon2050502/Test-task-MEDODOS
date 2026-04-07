package recurringtask

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	recurringtaskdomain "example.com/taskservice/internal/domain/recurringtask"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo     Repository
	taskRepo GeneratedTaskRepository
	now      func() time.Time
}

func NewService(repo Repository, taskRepo GeneratedTaskRepository) *Service {
	return NewServiceWithClock(repo, taskRepo, func() time.Time { return time.Now().UTC() })
}

func NewServiceWithClock(repo Repository, taskRepo GeneratedTaskRepository, now func() time.Time) *Service {
	return &Service{
		repo:     repo,
		taskRepo: taskRepo,
		now:      now,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*recurringtaskdomain.RecurringTask, error) {
	model, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	if err := s.runTemplateOnce(ctx, created, startOfDayUTC(now)); err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*recurringtaskdomain.RecurringTask, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*recurringtaskdomain.RecurringTask, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	model, err := normalizeInput(CreateInput(input))
	if err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	model.ID = id
	model.CreatedAt = current.CreatedAt
	model.LastGeneratedFor = current.LastGeneratedFor
	model.UpdatedAt = s.now().UTC()

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	if err := s.runTemplateOnce(ctx, updated, startOfDayUTC(s.now())); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]recurringtaskdomain.RecurringTask, error) {
	return s.repo.List(ctx)
}

func (s *Service) RunOnce(ctx context.Context) error {
	today := startOfDayUTC(s.now())

	recurringTasks, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	for i := range recurringTasks {
		if err := s.runTemplateOnce(ctx, &recurringTasks[i], today); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) runTemplateOnce(ctx context.Context, recurringTask *recurringtaskdomain.RecurringTask, today time.Time) error {
	if recurringTask == nil {
		return nil
	}

	startDate := startOfDayUTC(recurringTask.StartDate)
	if startDate.After(today) {
		return nil
	}

	from := startDate
	if recurringTask.LastGeneratedFor != nil {
		nextDate := startOfDayUTC(*recurringTask.LastGeneratedFor).AddDate(0, 0, 1)
		if nextDate.After(today) {
			return nil
		}

		from = nextDate
	}

	for scheduledDate := from; !scheduledDate.After(today); scheduledDate = scheduledDate.AddDate(0, 0, 1) {
		if !matchesRule(*recurringTask, scheduledDate) {
			continue
		}

		now := s.now().UTC()
		scheduledFor := scheduledDate
		sourceRecurringTaskID := recurringTask.ID

		if _, err := s.taskRepo.CreateGeneratedIfMissing(ctx, &taskdomain.Task{
			Title:                 recurringTask.Title,
			Description:           recurringTask.Description,
			Status:                generatedTaskStatus(recurringTask.Priority),
			Priority:              recurringTask.Priority,
			ScheduledFor:          &scheduledFor,
			SourceRecurringTaskID: &sourceRecurringTaskID,
			CreatedAt:             now,
			UpdatedAt:             now,
		}); err != nil {
			return err
		}
	}

	if err := s.repo.SetLastGeneratedFor(ctx, recurringTask.ID, today); err != nil {
		return err
	}

	recurringTask.LastGeneratedFor = &today

	return nil
}

func normalizeInput(input CreateInput) (*recurringtaskdomain.RecurringTask, error) {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	startDate := startOfDayUTC(input.StartDate)

	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if startDate.IsZero() {
		return nil, fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}

	priority := priorityOrDefault(input.Priority)
	if !priority.Valid() {
		return nil, fmt.Errorf("%w: invalid priority", ErrInvalidInput)
	}

	rule, err := normalizeRule(input, startDate)
	if err != nil {
		return nil, err
	}

	return &recurringtaskdomain.RecurringTask{
		Title:       title,
		Description: description,
		Priority:    priority,
		StartDate:   startDate,
		Rule:        rule,
	}, nil
}

func normalizeRule(input CreateInput, startDate time.Time) (recurringtaskdomain.Rule, error) {
	if !input.RuleType.Valid() {
		return recurringtaskdomain.Rule{}, fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}

	switch input.RuleType {
	case recurringtaskdomain.RuleTypeDaily:
		if input.EveryNDays <= 0 {
			return recurringtaskdomain.Rule{}, fmt.Errorf("%w: every_n_days must be greater than zero", ErrInvalidInput)
		}

		return recurringtaskdomain.Rule{
			Type:       input.RuleType,
			EveryNDays: input.EveryNDays,
		}, nil
	case recurringtaskdomain.RuleTypeMonthly:
		if input.DayOfMonth < 1 || input.DayOfMonth > 30 {
			return recurringtaskdomain.Rule{}, fmt.Errorf("%w: day_of_month must be between 1 and 30", ErrInvalidInput)
		}

		return recurringtaskdomain.Rule{
			Type:       input.RuleType,
			DayOfMonth: input.DayOfMonth,
		}, nil
	case recurringtaskdomain.RuleTypeSpecificDates:
		dates, err := normalizeDates(input.Dates, startDate)
		if err != nil {
			return recurringtaskdomain.Rule{}, err
		}

		return recurringtaskdomain.Rule{
			Type:  input.RuleType,
			Dates: dates,
		}, nil
	case recurringtaskdomain.RuleTypeDayParity:
		if !input.Parity.Valid() {
			return recurringtaskdomain.Rule{}, fmt.Errorf("%w: parity must be odd or even", ErrInvalidInput)
		}

		return recurringtaskdomain.Rule{
			Type:   input.RuleType,
			Parity: input.Parity,
		}, nil
	default:
		return recurringtaskdomain.Rule{}, fmt.Errorf("%w: invalid recurrence type", ErrInvalidInput)
	}
}

func priorityOrDefault(priority taskdomain.Priority) taskdomain.Priority {
	if priority == "" {
		return taskdomain.PriorityNormal
	}

	return priority
}

func generatedTaskStatus(priority taskdomain.Priority) taskdomain.Status {
	if priority == taskdomain.PriorityUrgent {
		return taskdomain.StatusInProgress
	}

	return taskdomain.StatusNew
}

func normalizeDates(dates []time.Time, startDate time.Time) ([]time.Time, error) {
	if len(dates) == 0 {
		return nil, fmt.Errorf("%w: dates are required for specific_dates", ErrInvalidInput)
	}

	normalized := make([]time.Time, 0, len(dates))
	seen := make(map[string]struct{}, len(dates))

	for _, rawDate := range dates {
		date := startOfDayUTC(rawDate)
		if date.IsZero() {
			return nil, fmt.Errorf("%w: dates must be valid", ErrInvalidInput)
		}

		if date.Before(startDate) {
			return nil, fmt.Errorf("%w: dates must be on or after start_date", ErrInvalidInput)
		}

		key := date.Format(dateLayout)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: dates must be unique", ErrInvalidInput)
		}

		seen[key] = struct{}{}
		normalized = append(normalized, date)
	}

	slices.SortFunc(normalized, func(a, b time.Time) int {
		return a.Compare(b)
	})

	return normalized, nil
}

func matchesRule(recurringTask recurringtaskdomain.RecurringTask, date time.Time) bool {
	date = startOfDayUTC(date)
	startDate := startOfDayUTC(recurringTask.StartDate)

	switch recurringTask.Rule.Type {
	case recurringtaskdomain.RuleTypeDaily:
		if recurringTask.Rule.EveryNDays <= 0 || date.Before(startDate) {
			return false
		}

		daysSinceStart := int(date.Sub(startDate).Hours() / 24)
		return daysSinceStart%recurringTask.Rule.EveryNDays == 0
	case recurringtaskdomain.RuleTypeMonthly:
		return !date.Before(startDate) && date.Day() == recurringTask.Rule.DayOfMonth
	case recurringtaskdomain.RuleTypeSpecificDates:
		for _, candidate := range recurringTask.Rule.Dates {
			if startOfDayUTC(candidate).Equal(date) {
				return true
			}
		}

		return false
	case recurringtaskdomain.RuleTypeDayParity:
		if date.Before(startDate) {
			return false
		}

		if recurringTask.Rule.Parity == recurringtaskdomain.DayParityOdd {
			return date.Day()%2 == 1
		}

		return date.Day()%2 == 0
	default:
		return false
	}
}

func startOfDayUTC(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

const dateLayout = "2006-01-02"
