package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/multi-worker/internal/middleware"
	"github.com/multi-worker/internal/model"
	"github.com/multi-worker/internal/scheduler"
	"github.com/multi-worker/internal/storage"
)

// Handler contains all API handlers
type Handler struct {
	userRepo  *storage.UserRepository
	taskRepo  *storage.TaskRepository
	execRepo  *storage.ExecutionRepository
	scheduler *scheduler.Scheduler
	runner    *scheduler.PipelineRunner
	auth      *middleware.AuthMiddleware
}

// NewHandler creates a new API handler
func NewHandler(
	userRepo *storage.UserRepository,
	taskRepo *storage.TaskRepository,
	execRepo *storage.ExecutionRepository,
	sched *scheduler.Scheduler,
	runner *scheduler.PipelineRunner,
	auth *middleware.AuthMiddleware,
) *Handler {
	return &Handler{
		userRepo:  userRepo,
		taskRepo:  taskRepo,
		execRepo:  execRepo,
		scheduler: sched,
		runner:    runner,
		auth:      auth,
	}
}

// Response helpers
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Auth handlers

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "Registration details"
// @Success 201 {object} model.LoginResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 409 {object} map[string]string "Email already registered"
// @Failure 500 {object} map[string]string "Server error"
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}

	// Validate email format (basic check)
	if !isValidEmail(req.Email) {
		respondError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Validate password length
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// Check if email exists
	existing, _ := h.userRepo.FindByEmail(r.Context(), req.Email)
	if existing != nil {
		respondError(w, http.StatusConflict, "email already registered")
		return
	}

	user, err := h.userRepo.Create(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, expiresAt, err := h.auth.GenerateToken(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondJSON(w, http.StatusCreated, model.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login credentials"
// @Success 200 {object} model.LoginResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Server error"
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.userRepo.FindByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !h.userRepo.ValidatePassword(user, req.Password) {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, expiresAt, err := h.auth.GenerateToken(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.userRepo.UpdateLastLogin(r.Context(), user.ID)

	respondJSON(w, http.StatusOK, model.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the current user's profile information
// @Tags Authentication
// @Produce json
// @Success 200 {object} model.User
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /auth/profile [get]
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.userRepo.FindByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// RegenerateAPIKey godoc
// @Summary Regenerate API key
// @Description Generate a new API key for the current user
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]string "New API key"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /auth/api-key/regenerate [post]
func (h *Handler) RegenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	apiKey, err := h.userRepo.RegenerateAPIKey(r.Context(), claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to regenerate API key")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"api_key": apiKey})
}

// Task handlers

// CreateTask godoc
// @Summary Create a new task
// @Description Create a new scheduled task with pipeline configuration
// @Tags Tasks
// @Accept json
// @Produce json
// @Param request body model.CreateTaskRequest true "Task configuration"
// @Success 201 {object} model.Task
// @Failure 400 {object} map[string]string "Invalid request or pipeline"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "task name is required")
		return
	}
	if req.Schedule == "" {
		respondError(w, http.StatusBadRequest, "schedule is required")
		return
	}
	if len(req.Pipeline) == 0 {
		respondError(w, http.StatusBadRequest, "at least one pipeline step is required")
		return
	}

	// Validate pipeline
	if errs := h.runner.ValidatePipeline(req.Pipeline); len(errs) > 0 {
		respondError(w, http.StatusBadRequest, errs[0].Error())
		return
	}

	task, err := h.taskRepo.Create(r.Context(), &req, claims.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	// Add to scheduler
	h.scheduler.AddTask(*task)

	respondJSON(w, http.StatusCreated, task)
}

// GetTasks godoc
// @Summary List all tasks
// @Description Get a paginated list of tasks with optional status filter
// @Tags Tasks
// @Produce json
// @Param limit query int false "Number of tasks to return" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Param status query string false "Filter by status (enabled, disabled, running)"
// @Success 200 {object} map[string]interface{} "Tasks list with pagination"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks [get]
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	var status *model.TaskStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := model.TaskStatus(s)
		status = &st
	}

	tasks, err := h.taskRepo.FindAll(r.Context(), status, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch tasks")
		return
	}

	// Add next run times
	for i := range tasks {
		if next := h.scheduler.GetNextRun(tasks[i].ID); next != nil {
			tasks[i].NextRunAt = next
		}
	}

	total, _ := h.taskRepo.Count(r.Context(), status)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetTask godoc
// @Summary Get a task
// @Description Get details of a specific task by ID
// @Tags Tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} model.Task
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id} [get]
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	task, err := h.taskRepo.FindByID(r.Context(), taskID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch task")
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	if next := h.scheduler.GetNextRun(task.ID); next != nil {
		task.NextRunAt = next
	}

	respondJSON(w, http.StatusOK, task)
}

// UpdateTask godoc
// @Summary Update a task
// @Description Update an existing task's configuration
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body model.UpdateTaskRequest true "Updated task configuration"
// @Success 200 {object} model.Task
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Task not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id} [put]
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate pipeline if provided
	if req.Pipeline != nil {
		if errs := h.runner.ValidatePipeline(req.Pipeline); len(errs) > 0 {
			respondError(w, http.StatusBadRequest, errs[0].Error())
			return
		}
	}

	task, err := h.taskRepo.Update(r.Context(), taskID, &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	// Update scheduler
	h.scheduler.UpdateTask(*task)

	respondJSON(w, http.StatusOK, task)
}

// DeleteTask godoc
// @Summary Delete a task
// @Description Delete a task and remove it from the scheduler
// @Tags Tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} map[string]string "Deletion status"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Task not found"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	if err := h.taskRepo.Delete(r.Context(), taskID); err != nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	// Remove from scheduler
	h.scheduler.RemoveTask(taskID)

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// TriggerTask godoc
// @Summary Trigger a task manually
// @Description Execute a task immediately regardless of its schedule
// @Tags Tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} model.Execution
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Execution error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id}/run [post]
func (h *Handler) TriggerTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	claims := middleware.GetUserFromContext(r.Context())
	triggeredBy := "api"
	if claims != nil {
		triggeredBy = claims.UserID
	}

	execution, err := h.scheduler.TriggerTask(r.Context(), taskID, triggeredBy)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, execution)
}

// Execution handlers

// GetTaskExecutions godoc
// @Summary Get task executions
// @Description Get paginated list of executions for a specific task
// @Tags Executions
// @Produce json
// @Param id path string true "Task ID"
// @Param limit query int false "Number of executions to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{} "Executions list"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id}/executions [get]
func (h *Handler) GetTaskExecutions(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		respondError(w, http.StatusBadRequest, "task ID required")
		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	executions, err := h.execRepo.FindByTaskID(r.Context(), taskID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch executions")
		return
	}

	total, _ := h.execRepo.CountByTaskID(r.Context(), taskID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// GetExecution godoc
// @Summary Get execution details
// @Description Get details of a specific execution
// @Tags Executions
// @Produce json
// @Param id path string true "Task ID"
// @Param execId path string true "Execution ID"
// @Success 200 {object} model.Execution
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Execution not found"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /tasks/{id}/executions/{execId} [get]
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	execID := r.PathValue("execId")
	if execID == "" {
		respondError(w, http.StatusBadRequest, "execution ID required")
		return
	}

	execution, err := h.execRepo.FindByID(r.Context(), execID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch execution")
		return
	}
	if execution == nil {
		respondError(w, http.StatusNotFound, "execution not found")
		return
	}

	respondJSON(w, http.StatusOK, execution)
}

// GetRecentExecutions godoc
// @Summary Get recent executions
// @Description Get the most recent executions across all tasks
// @Tags Executions
// @Produce json
// @Param limit query int false "Number of executions to return" default(20)
// @Success 200 {object} map[string]interface{} "Recent executions"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /executions/recent [get]
func (h *Handler) GetRecentExecutions(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	executions, err := h.execRepo.FindRecent(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch executions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
	})
}

// Health and status handlers

// Health godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "Health status"
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"scheduler": h.scheduler.IsRunning(),
	})
}

// Status godoc
// @Summary System status
// @Description Get detailed system status including task and execution counts
// @Tags System
// @Produce json
// @Success 200 {object} map[string]interface{} "System status"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /status [get]
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	taskCount, _ := h.taskRepo.Count(r.Context(), nil)
	enabledStatus := model.TaskStatusEnabled
	enabledCount, _ := h.taskRepo.Count(r.Context(), &enabledStatus)
	scheduledTasks := h.scheduler.GetScheduledTasks()

	runningStatus := model.ExecutionStatusRunning
	runningCount, _ := h.execRepo.CountByStatus(r.Context(), runningStatus)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scheduler_running": h.scheduler.IsRunning(),
		"total_tasks":       taskCount,
		"enabled_tasks":     enabledCount,
		"scheduled_tasks":   len(scheduledTasks),
		"running_executions": runningCount,
	})
}

// isValidEmail performs a basic email validation
func isValidEmail(email string) bool {
	// Basic email validation: contains @ and has text on both sides
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	// Check domain has at least one dot
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}
