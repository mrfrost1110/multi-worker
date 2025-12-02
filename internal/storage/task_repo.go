package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/multi-worker/internal/model"
)

type TaskRepository struct {
	db *Database
}

func NewTaskRepository(db *Database) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, req *model.CreateTaskRequest, userID string) (*model.Task, error) {
	pipeline := model.PipelineSteps(req.Pipeline)

	var task model.Task
	query := `
		INSERT INTO tasks (name, description, schedule, pipeline, created_by, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
	`
	err := r.db.QueryRowxContext(ctx, query, req.Name, req.Description, req.Schedule, pipeline, userID, model.TaskStatusEnabled).
		StructScan(&task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	var task model.Task
	query := `
		SELECT id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}
	return &task, nil
}

func (r *TaskRepository) FindAll(ctx context.Context, status *model.TaskStatus, limit, offset int) ([]model.Task, error) {
	var tasks []model.Task
	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
			FROM tasks WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`
		args = []interface{}{*status, limit, offset}
	} else {
		query = `
			SELECT id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
			FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	err := r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) FindEnabled(ctx context.Context) ([]model.Task, error) {
	var tasks []model.Task
	query := `
		SELECT id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
		FROM tasks WHERE status = $1
	`
	err := r.db.SelectContext(ctx, &tasks, query, model.TaskStatusEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled tasks: %w", err)
	}
	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, id string, req *model.UpdateTaskRequest) (*model.Task, error) {
	task, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Schedule != nil {
		task.Schedule = *req.Schedule
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Pipeline != nil {
		task.Pipeline = req.Pipeline
	}

	query := `
		UPDATE tasks SET name = $1, description = $2, schedule = $3, status = $4, pipeline = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, name, description, schedule, status, pipeline, last_run_at, next_run_at, created_by, created_at, updated_at
	`
	err = r.db.QueryRowxContext(ctx, query, task.Name, task.Description, task.Schedule, task.Status, model.PipelineSteps(task.Pipeline), time.Now(), id).
		StructScan(task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateLastRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	query := `UPDATE tasks SET last_run_at = $1, next_run_at = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, lastRun, nextRun, time.Now(), id)
	return err
}

func (r *TaskRepository) UpdateNextRun(ctx context.Context, id string, nextRun time.Time) error {
	query := `UPDATE tasks SET next_run_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, nextRun, time.Now(), id)
	return err
}

func (r *TaskRepository) UpdateLastRunOnly(ctx context.Context, id string, lastRun time.Time) error {
	query := `UPDATE tasks SET last_run_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, lastRun, time.Now(), id)
	return err
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	query := `UPDATE tasks SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *TaskRepository) Count(ctx context.Context, status *model.TaskStatus) (int, error) {
	var count int
	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT COUNT(*) FROM tasks WHERE status = $1`
		args = []interface{}{*status}
	} else {
		query = `SELECT COUNT(*) FROM tasks`
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}
