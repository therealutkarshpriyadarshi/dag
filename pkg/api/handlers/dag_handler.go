package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/therealutkarshpriyadarshi/dag/internal/dag"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/dto"
	"github.com/therealutkarshpriyadarshi/dag/pkg/api/middleware"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// DAGHandler handles DAG-related HTTP requests
type DAGHandler struct {
	dagRepo storage.DAGRepository
	engine  *dag.Engine
}

// NewDAGHandler creates a new DAG handler
func NewDAGHandler(dagRepo storage.DAGRepository, engine *dag.Engine) *DAGHandler {
	return &DAGHandler{
		dagRepo: dagRepo,
		engine:  engine,
	}
}

// CreateDAG handles POST /api/v1/dags
// @Summary Create a new DAG
// @Description Create a new DAG workflow
// @Tags dags
// @Accept json
// @Produce json
// @Param dag body dto.CreateDAGRequest true "DAG definition"
// @Success 201 {object} dto.DAGResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags [post]
func (h *DAGHandler) CreateDAG(c *gin.Context) {
	var req dto.CreateDAGRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	// Convert DTO to model
	dagModel := req.ToDAG()

	// Validate DAG structure using the DAG engine
	builder := dag.NewDAGBuilder(dagModel.Name).
		WithDescription(dagModel.Description).
		WithSchedule(dagModel.Schedule)

	for _, task := range dagModel.Tasks {
		builder.AddTask(task.ID, task.Name, task.Type, task.Command, task.Dependencies...)
	}

	dagInstance, err := builder.Build()
	if err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_DAG", err.Error())
		return
	}

	// Validate the DAG (check for cycles, etc.)
	if err := dagInstance.Validate(); err != nil {
		middleware.AbortWithError(c, http.StatusBadRequest, "DAG_VALIDATION_FAILED", err.Error())
		return
	}

	// Save to database
	dagModel.Tasks = dagInstance.GetTasks()
	if err := h.dagRepo.Create(c.Request.Context(), dagModel); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	// Return response
	response := dto.ToDAGResponse(dagModel)
	c.JSON(http.StatusCreated, response)
}

// ListDAGs handles GET /api/v1/dags
// @Summary List DAGs
// @Description Get a paginated list of DAGs with optional filters
// @Tags dags
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param is_paused query bool false "Filter by paused status"
// @Param tags query []string false "Filter by tags"
// @Success 200 {object} dto.DAGListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags [get]
func (h *DAGHandler) ListDAGs(c *gin.Context) {
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
	filters := storage.DAGFilters{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}

	if isPausedStr := c.Query("is_paused"); isPausedStr != "" {
		isPaused := isPausedStr == "true"
		filters.IsPaused = &isPaused
	}

	if tags := c.QueryArray("tags"); len(tags) > 0 {
		filters.Tags = tags
	}

	// Get DAGs from database
	dags, err := h.dagRepo.List(c.Request.Context(), filters)
	if err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}

	// Convert to response
	dagResponses := make([]dto.DAGResponse, len(dags))
	for i, d := range dags {
		dagResponses[i] = dto.ToDAGResponse(d)
	}

	// TODO: Get total count for pagination
	totalCount := int64(len(dagResponses))

	response := dto.DAGListResponse{
		DAGs:       dagResponses,
		Pagination: dto.NewPaginationMeta(page, pageSize, totalCount),
	}

	c.JSON(http.StatusOK, response)
}

// GetDAG handles GET /api/v1/dags/:id
// @Summary Get DAG details
// @Description Get details of a specific DAG
// @Tags dags
// @Produce json
// @Param id path string true "DAG ID"
// @Success 200 {object} dto.DAGResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id} [get]
func (h *DAGHandler) GetDAG(c *gin.Context) {
	id := c.Param("id")

	dag, err := h.dagRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "DAG_NOT_FOUND", "DAG not found")
		return
	}

	response := dto.ToDAGResponse(dag)
	c.JSON(http.StatusOK, response)
}

// UpdateDAG handles PATCH /api/v1/dags/:id
// @Summary Update DAG
// @Description Update an existing DAG
// @Tags dags
// @Accept json
// @Produce json
// @Param id path string true "DAG ID"
// @Param dag body dto.UpdateDAGRequest true "DAG update"
// @Success 200 {object} dto.DAGResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id} [patch]
func (h *DAGHandler) UpdateDAG(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateDAGRequest
	if !middleware.BindAndValidate(c, &req) {
		return
	}

	// Get existing DAG
	dag, err := h.dagRepo.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, http.StatusNotFound, "DAG_NOT_FOUND", "DAG not found")
		return
	}

	// Update fields
	if req.Name != nil {
		dag.Name = *req.Name
	}
	if req.Description != nil {
		dag.Description = *req.Description
	}
	if req.Schedule != nil {
		dag.Schedule = *req.Schedule
	}
	if req.Tasks != nil {
		tasks := make([]models.Task, len(req.Tasks))
		for i, taskDTO := range req.Tasks {
			tasks[i] = taskDTO.ToTask()
		}
		dag.Tasks = tasks

		// Validate the updated DAG
		builder := dag.NewDAGBuilder(dag.Name)
		for _, task := range dag.Tasks {
			builder.AddTask(task.ID, task.Name, task.Type, task.Command, task.Dependencies...)
		}
		dagInstance, err := builder.Build()
		if err != nil {
			middleware.AbortWithError(c, http.StatusBadRequest, "INVALID_DAG", err.Error())
			return
		}
		if err := dagInstance.Validate(); err != nil {
			middleware.AbortWithError(c, http.StatusBadRequest, "DAG_VALIDATION_FAILED", err.Error())
			return
		}
	}
	if req.StartDate != nil {
		dag.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		dag.EndDate = req.EndDate
	}
	if req.Tags != nil {
		dag.Tags = req.Tags
	}
	if req.IsPaused != nil {
		dag.IsPaused = *req.IsPaused
	}

	// Save to database
	if err := h.dagRepo.Update(c.Request.Context(), dag); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}

	response := dto.ToDAGResponse(dag)
	c.JSON(http.StatusOK, response)
}

// DeleteDAG handles DELETE /api/v1/dags/:id
// @Summary Delete DAG
// @Description Delete a DAG
// @Tags dags
// @Param id path string true "DAG ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id} [delete]
func (h *DAGHandler) DeleteDAG(c *gin.Context) {
	id := c.Param("id")

	if err := h.dagRepo.Delete(c.Request.Context(), id); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// PauseDAG handles POST /api/v1/dags/:id/pause
// @Summary Pause DAG
// @Description Pause a DAG to prevent new runs
// @Tags dags
// @Param id path string true "DAG ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id}/pause [post]
func (h *DAGHandler) PauseDAG(c *gin.Context) {
	id := c.Param("id")

	if err := h.dagRepo.Pause(c.Request.Context(), id); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "PAUSE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "DAG paused successfully",
	})
}

// UnpauseDAG handles POST /api/v1/dags/:id/unpause
// @Summary Unpause DAG
// @Description Unpause a DAG to allow new runs
// @Tags dags
// @Param id path string true "DAG ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/dags/{id}/unpause [post]
func (h *DAGHandler) UnpauseDAG(c *gin.Context) {
	id := c.Param("id")

	if err := h.dagRepo.Unpause(c.Request.Context(), id); err != nil {
		middleware.AbortWithError(c, http.StatusInternalServerError, "UNPAUSE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "DAG unpaused successfully",
	})
}
