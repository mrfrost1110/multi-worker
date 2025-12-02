package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/storage"
)

// Scheduler manages task scheduling and execution
type Scheduler struct {
	cron       *cron.Cron
	taskRepo   *storage.TaskRepository
	execRepo   *storage.ExecutionRepository
	runner     *PipelineRunner
	entryMap   map[string]cron.EntryID
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewScheduler creates a new scheduler
func NewScheduler(
	taskRepo *storage.TaskRepository,
	execRepo *storage.ExecutionRepository,
	runner *PipelineRunner,
) *Scheduler {
	return &Scheduler{
		cron:     cron.New(cron.WithSeconds()),
		taskRepo: taskRepo,
		execRepo: execRepo,
		runner:   runner,
		entryMap: make(map[string]cron.EntryID),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Load all enabled tasks
	tasks, err := s.taskRepo.FindEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	for _, task := range tasks {
		if err := s.scheduleTask(task); err != nil {
			log.Printf("Failed to schedule task %s: %v", task.ID, err)
		}
	}

	s.cron.Start()
	s.running = true

	log.Printf("Scheduler started with %d tasks", len(tasks))
	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.running = false
	log.Println("Scheduler stopped")
}

// AddTask adds a new task to the scheduler
func (s *Scheduler) AddTask(task model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.scheduleTask(task)
}

// UpdateTask updates an existing task in the scheduler
func (s *Scheduler) UpdateTask(task model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old entry if exists
	if entryID, ok := s.entryMap[task.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, task.ID)
	}

	// Add new entry if enabled
	if task.Status == model.TaskStatusEnabled {
		return s.scheduleTask(task)
	}

	return nil
}

// RemoveTask removes a task from the scheduler
func (s *Scheduler) RemoveTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entryMap[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, taskID)
	}
}

// TriggerTask triggers a task manually
func (s *Scheduler) TriggerTask(ctx context.Context, taskID, triggeredBy string) (*model.Execution, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	return s.runner.Run(ctx, *task, triggeredBy)
}

// GetNextRun returns the next run time for a task
func (s *Scheduler) GetNextRun(taskID string) *time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entryID, ok := s.entryMap[taskID]; ok {
		entry := s.cron.Entry(entryID)
		if !entry.Next.IsZero() {
			return &entry.Next
		}
	}
	return nil
}

// GetScheduledTasks returns all scheduled task IDs
func (s *Scheduler) GetScheduledTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id := range s.entryMap {
		ids = append(ids, id)
	}
	return ids
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Scheduler) scheduleTask(task model.Task) error {
	if task.Status != model.TaskStatusEnabled {
		return nil
	}

	// Parse cron expression
	schedule := task.Schedule

	// Support shortcuts
	switch schedule {
	case "@hourly":
		schedule = "0 0 * * * *"
	case "@daily":
		schedule = "0 0 0 * * *"
	case "@weekly":
		schedule = "0 0 0 * * 0"
	case "@monthly":
		schedule = "0 0 0 1 * *"
	}

	// If schedule doesn't have 6 parts, assume it's a 5-part cron and add seconds
	parts := len(splitCronParts(schedule))
	if parts == 5 {
		schedule = "0 " + schedule
	}

	entryID, err := s.cron.AddFunc(schedule, func() {
		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
		defer cancel()

		// Refresh task from database
		currentTask, err := s.taskRepo.FindByID(ctx, task.ID)
		if err != nil || currentTask == nil {
			log.Printf("Task %s not found, removing from scheduler", task.ID)
			s.RemoveTask(task.ID)
			return
		}

		// Skip if task is not enabled or already running
		if currentTask.Status == model.TaskStatusRunning {
			log.Printf("Task %s is already running, skipping scheduled execution", task.ID)
			return
		}
		if currentTask.Status != model.TaskStatusEnabled {
			return
		}

		_, err = s.runner.Run(ctx, *currentTask, "schedule")
		if err != nil {
			log.Printf("Task %s execution failed: %v", task.ID, err)
		}

		// Update next run time after execution
		s.mu.RLock()
		if entryID, ok := s.entryMap[task.ID]; ok {
			entry := s.cron.Entry(entryID)
			if !entry.Next.IsZero() {
				if err := s.taskRepo.UpdateNextRun(ctx, task.ID, entry.Next); err != nil {
					log.Printf("Warning: failed to update next run time for task %s: %v", task.ID, err)
				}
			}
		}
		s.mu.RUnlock()
	})

	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", task.Schedule, err)
	}

	s.entryMap[task.ID] = entryID

	// Set initial next run time (without updating last_run_at)
	entry := s.cron.Entry(entryID)
	if !entry.Next.IsZero() {
		if err := s.taskRepo.UpdateNextRun(s.ctx, task.ID, entry.Next); err != nil {
			log.Printf("Warning: failed to set initial next run time for task %s: %v", task.ID, err)
		}
	}

	return nil
}

func splitCronParts(schedule string) []string {
	var parts []string
	current := ""
	for _, r := range schedule {
		if r == ' ' || r == '\t' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
