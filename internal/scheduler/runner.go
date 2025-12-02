package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/multi-worker/internal/executor/ai"
	"github.com/multi-worker/internal/executor/discord"
	"github.com/multi-worker/internal/executor/filter"
	"github.com/multi-worker/internal/executor/rss"
	"github.com/multi-worker/internal/executor/scraper"
	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// PipelineRunner executes task pipelines
type PipelineRunner struct {
	taskRepo    *storage.TaskRepository
	execRepo    *storage.ExecutionRepository
	cacheRepo   *storage.CacheRepository
	discordRepo *storage.DiscordRepository
	aiExecutor  *ai.Executor
	scraperExec *scraper.Executor
	rssExec     *rss.Executor
	discordExec *discord.Executor
	filterExec  *filter.Executor
}

// NewPipelineRunner creates a new pipeline runner
func NewPipelineRunner(
	taskRepo *storage.TaskRepository,
	execRepo *storage.ExecutionRepository,
	cacheRepo *storage.CacheRepository,
	discordRepo *storage.DiscordRepository,
	aiExec *ai.Executor,
	scraperExec *scraper.Executor,
	rssExec *rss.Executor,
	discordExec *discord.Executor,
	filterExec *filter.Executor,
) *PipelineRunner {
	return &PipelineRunner{
		taskRepo:    taskRepo,
		execRepo:    execRepo,
		cacheRepo:   cacheRepo,
		discordRepo: discordRepo,
		aiExecutor:  aiExec,
		scraperExec: scraperExec,
		rssExec:     rssExec,
		discordExec: discordExec,
		filterExec:  filterExec,
	}
}

// Run executes a task's pipeline
func (r *PipelineRunner) Run(ctx context.Context, task model.Task, triggeredBy string) (*model.Execution, error) {
	// Create execution record
	execution, err := r.execRepo.Create(ctx, task.ID, task.Name, triggeredBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Update task status to running
	if err := r.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusRunning); err != nil {
		log.Printf("Warning: failed to update task status to running: %v", err)
	}

	// Execute pipeline
	stepResults, finalErr := r.executePipeline(ctx, task, execution.ID)

	// Update execution with results
	if finalErr != nil {
		errMsg := finalErr.Error()
		if err := r.execRepo.Fail(ctx, execution.ID, stepResults, errMsg); err != nil {
			log.Printf("Warning: failed to mark execution as failed: %v", err)
		}
	} else {
		if err := r.execRepo.Complete(ctx, execution.ID, stepResults); err != nil {
			log.Printf("Warning: failed to mark execution as complete: %v", err)
		}
	}

	// Update task status back to enabled
	if err := r.taskRepo.UpdateStatus(ctx, task.ID, model.TaskStatusEnabled); err != nil {
		log.Printf("Warning: failed to update task status to enabled: %v", err)
	}

	// Update last run time (next_run_at is managed by scheduler)
	if err := r.taskRepo.UpdateLastRunOnly(ctx, task.ID, time.Now()); err != nil {
		log.Printf("Warning: failed to update last run time: %v", err)
	}

	// Fetch updated execution
	execution, _ = r.execRepo.FindByID(ctx, execution.ID)

	return execution, finalErr
}

func (r *PipelineRunner) executePipeline(ctx context.Context, task model.Task, execID string) (model.StepResults, error) {
	var stepResults model.StepResults
	var currentResult *model.ExecutorResult

	for i, step := range task.Pipeline {
		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("Step %d: %s", i+1, step.Type)
		}

		stepResult := model.StepResult{
			StepName:  stepName,
			StepType:  step.Type,
			Status:    "running",
			StartedAt: time.Now(),
		}

		// Add task_id to config for caching
		if step.Config == nil {
			step.Config = make(map[string]interface{})
		}
		step.Config["task_id"] = task.ID

		// Execute the step
		result, err := r.executeStep(ctx, step, currentResult)

		now := time.Now()
		stepResult.FinishedAt = &now

		if err != nil {
			// Check if it's a skip error
			if _, ok := err.(filter.SkipPipelineError); ok {
				stepResult.Status = "skipped"
				stepResult.Error = stringPtr(err.Error())
				stepResults = append(stepResults, stepResult)
				log.Printf("Task %s: pipeline skipped at step %d: %v", task.ID, i+1, err)
				return stepResults, nil
			}

			stepResult.Status = "failed"
			stepResult.Error = stringPtr(err.Error())
			stepResults = append(stepResults, stepResult)

			// Update execution with partial results
			if updateErr := r.execRepo.UpdateStepResults(ctx, execID, stepResults); updateErr != nil {
				log.Printf("Warning: failed to update step results: %v", updateErr)
			}

			return stepResults, fmt.Errorf("step %d (%s) failed: %w", i+1, step.Type, err)
		}

		// Check if we should skip remaining steps (empty results)
		if filter.SkipEmpty(result) {
			stepResult.Status = "completed"
			stepResult.Output = "No new items found"
			stepResults = append(stepResults, stepResult)
			log.Printf("Task %s: no new items at step %d, skipping notification", task.ID, i+1)
			return stepResults, nil
		}

		stepResult.Status = "completed"
		stepResult.Output = map[string]interface{}{
			"item_count": result.ItemCount,
			"metadata":   result.Metadata,
		}
		stepResults = append(stepResults, stepResult)

		currentResult = result

		// Update execution with progress
		if updateErr := r.execRepo.UpdateStepResults(ctx, execID, stepResults); updateErr != nil {
			log.Printf("Warning: failed to update step results: %v", updateErr)
		}
	}

	return stepResults, nil
}

func (r *PipelineRunner) executeStep(ctx context.Context, step model.PipelineStep, input *model.ExecutorResult) (*model.ExecutorResult, error) {
	switch step.Type {
	case "scraper":
		return r.scraperExec.Execute(ctx, input, step.Config)

	case "rss":
		return r.rssExec.Execute(ctx, input, step.Config)

	case "ai_processor", "ai":
		return r.aiExecutor.Execute(ctx, input, step.Config)

	case "discord":
		// Resolve webhook URL from database if not in config
		if _, hasWebhook := step.Config["webhook_url"]; !hasWebhook {
			taskID, _ := step.Config["task_id"].(string)
			if taskID != "" && r.discordRepo != nil {
				webhookURL, err := r.discordRepo.GetWebhookForTask(ctx, taskID)
				if err == nil && webhookURL != "" {
					step.Config["webhook_url"] = webhookURL
				}
			}
		}
		return r.discordExec.Execute(ctx, input, step.Config)

	case "filter":
		return r.filterExec.Execute(ctx, input, step.Config)

	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// ValidatePipeline validates a pipeline configuration
func (r *PipelineRunner) ValidatePipeline(pipeline []model.PipelineStep) []error {
	var errors []error

	for i, step := range pipeline {
		var err error
		switch step.Type {
		case "scraper":
			err = r.scraperExec.Validate(step.Config)
		case "rss":
			err = r.rssExec.Validate(step.Config)
		case "ai_processor", "ai":
			err = r.aiExecutor.Validate(step.Config)
		case "discord":
			err = r.discordExec.Validate(step.Config)
		case "filter":
			err = r.filterExec.Validate(step.Config)
		default:
			err = fmt.Errorf("unknown step type: %s", step.Type)
		}

		if err != nil {
			errors = append(errors, fmt.Errorf("step %d: %w", i+1, err))
		}
	}

	return errors
}

func stringPtr(s string) *string {
	return &s
}
