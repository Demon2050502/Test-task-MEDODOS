package recurringtask

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type RuleType string

const (
	RuleTypeDaily         RuleType = "daily"
	RuleTypeMonthly       RuleType = "monthly"
	RuleTypeSpecificDates RuleType = "specific_dates"
	RuleTypeDayParity     RuleType = "day_parity"
)

type DayParity string

const (
	DayParityOdd  DayParity = "odd"
	DayParityEven DayParity = "even"
)

type Rule struct {
	Type       RuleType    `json:"type"`
	EveryNDays int         `json:"every_n_days,omitempty"`
	DayOfMonth int         `json:"day_of_month,omitempty"`
	Parity     DayParity   `json:"parity,omitempty"`
	Dates      []time.Time `json:"dates,omitempty"`
}

type RecurringTask struct {
	ID               int64               `json:"id"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Priority         taskdomain.Priority `json:"priority"`
	StartDate        time.Time           `json:"start_date"`
	Rule             Rule                `json:"rule"`
	LastGeneratedFor *time.Time          `json:"last_generated_for,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

func (t RuleType) Valid() bool {
	switch t {
	case RuleTypeDaily, RuleTypeMonthly, RuleTypeSpecificDates, RuleTypeDayParity:
		return true
	default:
		return false
	}
}

func (p DayParity) Valid() bool {
	switch p {
	case DayParityOdd, DayParityEven:
		return true
	default:
		return false
	}
}
