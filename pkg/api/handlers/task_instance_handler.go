package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// TaskInstanceHandler handles task instance-related HTTP requests
type TaskInstanceHandler struct {
	taskInstanceRepo storage.TaskInstanceRepository
	taskLogRepo      storage.TaskLogRepository
}

// NewTaskInstanceHandler creates a new task instance handler
func NewTaskInstanceHandler(
	taskInstanceRepo storage.TaskInstanceRepository,
	taskLogRepo storage.TaskLogRepository,
) *TaskInstanceHandler {
	return &TaskInstanceHandler{
		taskInstanceRepo: taskInstanceRepo,
		taskLogRepo:      taskLogRepo,
	}
}

// ListTaskInstances handles GET /api/v1/task-instances
// @Summary List task instances
// @Description Get a paginated list of task instances with optional filters
// @Tags task-instances
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param dag_run_id query string false "Filter by DAG run ID"
// @Param task_id query string false "Filter by task ID"
// @Param state query string false "Filter by state"
// @Success 200 {object} dto.TaskInstanceListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/task-instances [get]
func (h *TaskInstanceHandler) ListTaskInstances(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Build filters
	filters := storage.TaskInstanceFilters{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	if dagRunID := c.Query("dag_run_id"); dagRunID != "" {
		filters.DAGRunID = dagRunID
	}

	if taskID := c.Query("task_id"); taskID != "" {
		filters.TaskID = taskID
	}

	if stateStr := c.Query("state"); stateStr != "" {
		state := models.State(stateStr)
		filters.State = &state
	}

	// Get task instances from database
	instances, err := h.taskInstanceRepo.List(c.Request.Context(), filters)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}

	// Convert to response
	instanceResponses := make([]dto.TaskInstanceResponse, len(instances))
	for i, ti := range instances {
		instanceResponses[i] = dto.ToTaskInstanceResponse(ti)
	}

	// TODO: Get total count for pagination
	totalCount := int64(len(instanceResponses))

	response := dto.TaskInstanceListResponse{
		TaskInstances: instanceResponses,
		Pagination:    dto.NewPaginationMeta(page, pageSize, totalCount),
	}

	c.JSON(http.StatusOK, response)
}

// GetTaskInstance handles GET /api/v1/task-instances/:id
// @Summary Get task instance details
// @Description Get details of a specific task instance
// @Tags task-instances
// @Produce json
// @Param id path string true "Task Instance ID"
// @Success 200 {object} dto.TaskInstanceResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/task-instances/{id} [get]
func (h *TaskInstanceHandler) GetTaskInstance(c *gin.Context) {
	id := c.Param("id")

	taskInstance, err := h.taskInstanceRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "TASK_INSTANCE_NOT_FOUND", "Task instance not found")
		return
	}

	response := dto.ToTaskInstanceResponse(taskInstance)
	c.JSON(http.StatusOK, response)
}

// GetTaskInstanceLogs handles GET /api/v1/task-instances/:id/logs
// @Summary Get task instance logs
// @Description Get logs for a specific task instance
// @Tags task-instances
// @Produce json
// @Param id path string true "Task Instance ID"
// @Param limit query int false "Number of log entries to return" default(100)
// @Success 200 {object} dto.TaskLogsResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/task-instances/{id}/logs [get]
func (h *TaskInstanceHandler) GetTaskInstanceLogs(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	if limit < 1 || limit > 1000 {
		limit = 100
	}

	// Check if task instance exists
	_, err := h.taskInstanceRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "TASK_INSTANCE_NOT_FOUND", "Task instance not found")
		return
	}

	// Get logs
	logs, err := h.taskLogRepo.List(c.Request.Context(), id, limit)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "GET_LOGS_FAILED", err.Error())
		return
	}

	// Convert to response
	logResponses := make([]dto.TaskLogResponse, len(logs))
	for i, log := range logs {
		logResponses[i] = dto.TaskLogResponse{
			ID:        log.ID.String(),
			LogData:   log.LogData,
			Timestamp: log.Timestamp,
		}
	}

	response := dto.TaskLogsResponse{
		Logs: logResponses,
		Pagination: dto.PaginationMeta{
			Page:       1,
			PageSize:   limit,
			TotalPages: 1,
			TotalCount: int64(len(logResponses)),
		},
	}

	c.JSON(http.StatusOK, response)
}

// RetryTaskInstance handles POST /api/v1/task-instances/:id/retry
// @Summary Retry task instance
// @Description Manually retry a failed task instance
// @Tags task-instances
// @Param id path string true "Task Instance ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/task-instances/{id}/retry [post]
func (h *TaskInstanceHandler) RetryTaskInstance(c *gin.Context) {
	id := c.Param("id")

	taskInstance, err := h.taskInstanceRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "TASK_INSTANCE_NOT_FOUND", "Task instance not found")
		return
	}

	// Check if task can be retried
	if taskInstance.State != models.StateFailed && taskInstance.State != models.StateUpstreamFailed {
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_STATE",
			"Can only retry tasks in failed or upstream_failed state")
		return
	}

	// Check if we've exceeded max retries
	if taskInstance.TryNumber >= taskInstance.MaxTries {
		middleware.AbortWithError(c, http.StatusBadRequest, "MAX_RETRIES_EXCEEDED",
			"Task has already reached maximum retry attempts")
		return
	}

	// Update state to queued and increment try number
	taskInstance.State = models.StateQueued
	taskInstance.TryNumber++
	taskInstance.ErrorMessage = ""
	taskInstance.StartDate = nil
	taskInstance.EndDate = nil

	if err := h.taskInstanceRepo.Update(c.Request.Context(), taskInstance); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "RETRY_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Task instance queued for retry",
	})
}

// ListDAGRunTasks handles GET /api/v1/dag-runs/:id/tasks
// @Summary List tasks in a DAG run
// @Description Get all task instances for a specific DAG run
// @Tags dag-runs
// @Produce json
// @Param id path string true "DAG Run ID"
// @Success 200 {object} dto.TaskInstanceListResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dag-runs/{id}/tasks [get]
func (h *TaskInstanceHandler) ListDAGRunTasks(c *gin.Context) {
	dagRunID := c.Param("id")

	taskInstances, err := h.taskInstanceRepo.ListByDAGRun(c.Request.Context(), dagRunID)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}

	// Convert to response
	instanceResponses := make([]dto.TaskInstanceResponse, len(taskInstances))
	for i, ti := range taskInstances {
		instanceResponses[i] = dto.ToTaskInstanceResponse(ti)
	}

	totalCount := int64(len(instanceResponses))

	response := dto.TaskInstanceListResponse{
		TaskInstances: instanceResponses,
		Pagination: dto.PaginationMeta{
			Page:       1,
			PageSize:   len(instanceResponses),
			TotalPages: 1,
			TotalCount: totalCount,
		},
	}

	c.JSON(http.StatusOK, response)
}
