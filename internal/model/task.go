package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type TaskStatus string

const (
	TaskStatusEnabled  TaskStatus = "enabled"
	TaskStatusDisabled TaskStatus = "disabled"
	TaskStatusRunning  TaskStatus = "running"
)

type Task struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Schedule    string         `json:"schedule" db:"schedule"` // Cron expression
	Status      TaskStatus     `json:"status" db:"status"`
	Pipeline    PipelineSteps  `json:"pipeline" db:"pipeline"`
	LastRunAt   *time.Time     `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt   *time.Time     `json:"next_run_at,omitempty" db:"next_run_at"`
	CreatedBy   string         `json:"created_by" db:"created_by"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

type PipelineStep struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name,omitempty"`
	Config map[string]interface{} `json:"config"`
}

type PipelineSteps []PipelineStep

func (p PipelineSteps) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *PipelineSteps) Scan(value interface{}) error {
	if value == nil {
		*p = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, p)
}

type CreateTaskRequest struct {
	Name        string         `json:"name" validate:"required,min=3,max=100"`
	Description string         `json:"description" validate:"max=500"`
	Schedule    string         `json:"schedule" validate:"required"`
	Pipeline    []PipelineStep `json:"pipeline" validate:"required,min=1"`
}

type UpdateTaskRequest struct {
	Name        *string        `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Description *string        `json:"description,omitempty" validate:"omitempty,max=500"`
	Schedule    *string        `json:"schedule,omitempty"`
	Status      *TaskStatus    `json:"status,omitempty"`
	Pipeline    []PipelineStep `json:"pipeline,omitempty"`
}
