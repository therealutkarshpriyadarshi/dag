package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/internal/executor"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// DAGRunHandler handles DAG run-related HTTP requests
type DAGRunHandler struct {
	dagRepo        storage.DAGRepository
	dagRunRepo     storage.DAGRunRepository
	taskInstanceRepo storage.TaskInstanceRepository
	executor       executor.Executor
}

// NewDAGRunHandler creates a new DAG run handler
func NewDAGRunHandler(
	dagRepo storage.DAGRepository,
	dagRunRepo storage.DAGRunRepository,
	taskInstanceRepo storage.TaskInstanceRepository,
	exec executor.Executor,
) *DAGRunHandler {
	return &DAGRunHandler{
		dagRepo:          dagRepo,
		dagRunRepo:       dagRunRepo,
		taskInstanceRepo: taskInstanceRepo,
		executor:         exec,
	}
}

// TriggerDAG handles POST /api/v1/dags/:id/trigger
// @Summary Trigger a DAG run
// @Description Manually trigger a DAG execution
// @Tags dag-runs
// @Accept json
// @Produce json
// @Param id path string true "DAG ID"
// @Param request body dto.TriggerDAGRequest false "Trigger options"
// @Success 201 {object} dto.DAGRunResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id}/trigger [post]
func (h *DAGRunHandler) TriggerDAG(c *gin.Context) {
	dagID := c.Param("id")

	var req dto.TriggerDAGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = dto.TriggerDAGRequest{}
	}

	// Get DAG
	dag, err := h.dagRepo.Get(c.Request.Context(), dagID)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "DAG_NOT_FOUND", "DAG not found")
		return
	}

	// Check if DAG is paused
	if dag.IsPaused {
		middleware.AbortWithError(c, http.StatusBadRequest, "DAG_PAUSED", "Cannot trigger a paused DAG")
		return
	}

	// Set execution date
	executionDate := time.Now()
	if req.ExecutionDate != nil {
		executionDate = *req.ExecutionDate
	}

	// Create DAG run
	dagRun := &models.DAGRun{
		ID:              uuid.New().String(),
		DAGID:           dagID,
		ExecutionDate:   executionDate,
		State:           models.StateQueued,
		ExternalTrigger: true,
	}

	if err := h.dagRunRepo.Create(c.Request.Context(), dagRun); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "CREATE_RUN_FAILED", err.Error())
		return
	}

	// Submit to executor (asynchronously)
	go func() {
		ctx := context.Background()
		_ = h.executor.Execute(ctx, dagRun, dag)
	}()

	response := dto.ToDAGRunResponse(dagRun)
	c.JSON(http.StatusCreated, response)
}

// ListDAGRuns handles GET /api/v1/dag-runs
// @Summary List DAG runs
// @Description Get a paginated list of DAG runs with optional filters
// @Tags dag-runs
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param dag_id query string false "Filter by DAG ID"
// @Param state query string false "Filter by state"
// @Success 200 {object} dto.DAGRunListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dag-runs [get]
func (h *DAGRunHandler) ListDAGRuns(c *gin.Context) {
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
	filters := storage.DAGRunFilters{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	if dagID := c.Query("dag_id"); dagID != "" {
		filters.DAGID = dagID
	}

	if stateStr := c.Query("state"); stateStr != "" {
		state := models.State(stateStr)
		filters.State = &state
	}

	// Get DAG runs from database
	runs, err := h.dagRunRepo.List(c.Request.Context(), filters)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}

	// Convert to response
	runResponses := make([]dto.DAGRunResponse, len(runs))
	for i, run := range runs {
		runResponses[i] = dto.ToDAGRunResponse(run)
	}

	// TODO: Get total count for pagination
	totalCount := int64(len(runResponses))

	response := dto.DAGRunListResponse{
		DAGRuns:    runResponses,
		Pagination: dto.NewPaginationMeta(page, pageSize, totalCount),
	}

	c.JSON(http.StatusOK, response)
}

// GetDAGRun handles GET /api/v1/dag-runs/:id
// @Summary Get DAG run details
// @Description Get details of a specific DAG run including task instances
// @Tags dag-runs
// @Produce json
// @Param id path string true "DAG Run ID"
// @Success 200 {object} dto.DAGRunDetailResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dag-runs/{id} [get]
func (h *DAGRunHandler) GetDAGRun(c *gin.Context) {
	id := c.Param("id")

	dagRun, err := h.dagRunRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "DAG_RUN_NOT_FOUND", "DAG run not found")
		return
	}

	// Get task instances for this run
	taskInstances, err := h.taskInstanceRepo.ListByDAGRun(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "LIST_TASKS_FAILED", err.Error())
		return
	}

	// Convert to response
	taskResponses := make([]dto.TaskInstanceResponse, len(taskInstances))
	for i, ti := range taskInstances {
		taskResponses[i] = dto.ToTaskInstanceResponse(ti)
	}

	response := dto.DAGRunDetailResponse{
		DAGRunResponse: dto.ToDAGRunResponse(dagRun),
		Tasks:          taskResponses,
	}

	c.JSON(http.StatusOK, response)
}

// CancelDAGRun handles POST /api/v1/dag-runs/:id/cancel
// @Summary Cancel DAG run
// @Description Cancel a running DAG run
// @Tags dag-runs
// @Param id path string true "DAG Run ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dag-runs/{id}/cancel [post]
func (h *DAGRunHandler) CancelDAGRun(c *gin.Context) {
	id := c.Param("id")

	dagRun, err := h.dagRunRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "DAG_RUN_NOT_FOUND", "DAG run not found")
		return
	}

	// Check if run is in a cancellable state
	if dagRun.State == models.StateSuccess || dagRun.State == models.StateFailed {
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_STATE",
			"Cannot cancel a DAG run in terminal state")
		return
	}

	// Update state to failed (cancel)
	if err := h.dagRunRepo.UpdateState(c.Request.Context(), id, dagRun.State, models.StateFailed); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		return
	}

	// Cancel all running task instances
	taskInstances, err := h.taskInstanceRepo.ListByDAGRun(c.Request.Context(), id)
	if err == nil {
		for _, ti := range taskInstances {
			if ti.State == models.StateRunning || ti.State == models.StateQueued {
				_ = h.taskInstanceRepo.UpdateState(c.Request.Context(), ti.ID, ti.State, models.StateFailed)
			}
		}
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "DAG run cancelled successfully",
	})
}
