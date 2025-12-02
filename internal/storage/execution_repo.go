package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/multi-worker/internal/model"
)

type ExecutionRepository struct {
	db *Database
}

func NewExecutionRepository(db *Database) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

func (r *ExecutionRepository) Create(ctx context.Context, taskID, taskName, triggeredBy string) (*model.Execution, error) {
	var execution model.Execution
	query := `
		INSERT INTO executions (task_id, task_name, status, triggered_by, step_results)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, task_id, task_name, status, started_at, finished_at, duration_ms, step_results, error, triggered_by
	`
	err := r.db.QueryRowxContext(ctx, query, taskID, taskName, model.ExecutionStatusRunning, triggeredBy, model.StepResults{}).
		StructScan(&execution)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	return &execution, nil
}

func (r *ExecutionRepository) FindByID(ctx context.Context, id string) (*model.Execution, error) {
	var execution model.Execution
	query := `
		SELECT id, task_id, task_name, status, started_at, finished_at, duration_ms, step_results, error, triggered_by
		FROM executions WHERE id = $1
	`
	err := r.db.GetContext(ctx, &execution, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}
	return &execution, nil
}

func (r *ExecutionRepository) FindByTaskID(ctx context.Context, taskID string, limit, offset int) ([]model.Execution, error) {
	var executions []model.Execution
	query := `
		SELECT id, task_id, task_name, status, started_at, finished_at, duration_ms, step_results, error, triggered_by
		FROM executions WHERE task_id = $1
		ORDER BY started_at DESC LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &executions, query, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	return executions, nil
}

func (r *ExecutionRepository) FindRecent(ctx context.Context, limit int) ([]model.Execution, error) {
	var executions []model.Execution
	query := `
		SELECT id, task_id, task_name, status, started_at, finished_at, duration_ms, step_results, error, triggered_by
		FROM executions ORDER BY started_at DESC LIMIT $1
	`
	err := r.db.SelectContext(ctx, &executions, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find recent executions: %w", err)
	}
	return executions, nil
}

func (r *ExecutionRepository) UpdateStatus(ctx context.Context, id string, status model.ExecutionStatus, errMsg *string) error {
	now := time.Now()
	var duration int64

	exec, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if exec != nil {
		duration = now.Sub(exec.StartedAt).Milliseconds()
	}

	query := `UPDATE executions SET status = $1, finished_at = $2, duration_ms = $3, error = $4 WHERE id = $5`
	_, err = r.db.ExecContext(ctx, query, status, now, duration, errMsg, id)
	return err
}

func (r *ExecutionRepository) UpdateStepResults(ctx context.Context, id string, results model.StepResults) error {
	query := `UPDATE executions SET step_results = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, results, id)
	return err
}

func (r *ExecutionRepository) Complete(ctx context.Context, id string, results model.StepResults) error {
	now := time.Now()
	exec, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	var duration int64
	if exec != nil {
		duration = now.Sub(exec.StartedAt).Milliseconds()
	}

	query := `UPDATE executions SET status = $1, finished_at = $2, duration_ms = $3, step_results = $4 WHERE id = $5`
	_, err = r.db.ExecContext(ctx, query, model.ExecutionStatusCompleted, now, duration, results, id)
	return err
}

func (r *ExecutionRepository) Fail(ctx context.Context, id string, results model.StepResults, errMsg string) error {
	now := time.Now()
	exec, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	var duration int64
	if exec != nil {
		duration = now.Sub(exec.StartedAt).Milliseconds()
	}

	query := `UPDATE executions SET status = $1, finished_at = $2, duration_ms = $3, step_results = $4, error = $5 WHERE id = $6`
	_, err = r.db.ExecContext(ctx, query, model.ExecutionStatusFailed, now, duration, results, errMsg, id)
	return err
}

func (r *ExecutionRepository) CountByStatus(ctx context.Context, status model.ExecutionStatus) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM executions WHERE status = $1`
	err := r.db.GetContext(ctx, &count, query, status)
	return count, err
}

func (r *ExecutionRepository) CountByTaskID(ctx context.Context, taskID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM executions WHERE task_id = $1`
	err := r.db.GetContext(ctx, &count, query, taskID)
	return count, err
}

func (r *ExecutionRepository) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM executions WHERE started_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
