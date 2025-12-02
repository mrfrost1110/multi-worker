package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

type Execution struct {
	ID          string          `json:"id" db:"id"`
	TaskID      string          `json:"task_id" db:"task_id"`
	TaskName    string          `json:"task_name" db:"task_name"`
	Status      ExecutionStatus `json:"status" db:"status"`
	StartedAt   time.Time       `json:"started_at" db:"started_at"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty" db:"finished_at"`
	Duration    *int64          `json:"duration_ms,omitempty" db:"duration_ms"`
	StepResults StepResults     `json:"step_results" db:"step_results"`
	Error       *string         `json:"error,omitempty" db:"error"`
	TriggeredBy string          `json:"triggered_by" db:"triggered_by"` // "schedule" or "manual" or user_id
}

type StepResult struct {
	StepName   string      `json:"step_name"`
	StepType   string      `json:"step_type"`
	Status     string      `json:"status"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	Output     interface{} `json:"output,omitempty"`
	Error      *string     `json:"error,omitempty"`
}

type StepResults []StepResult

func (s StepResults) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StepResults) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

type ExecutionFilter struct {
	TaskID string
	Status ExecutionStatus
	Limit  int
	Offset int
}
