package dto

import (
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// CreateDAGRequest represents the request to create a new DAG
type CreateDAGRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=255"`
	Description string     `json:"description"`
	Schedule    string     `json:"schedule" validate:"omitempty,cron"`
	Tasks       []TaskDTO  `json:"tasks" validate:"required,min=1,dive"`
	StartDate   time.Time  `json:"start_date" validate:"required"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Tags        []string   `json:"tags"`
	IsPaused    bool       `json:"is_paused"`
}

// UpdateDAGRequest represents the request to update an existing DAG
type UpdateDAGRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Schedule    *string    `json:"schedule,omitempty" validate:"omitempty,cron"`
	Tasks       []TaskDTO  `json:"tasks,omitempty" validate:"omitempty,min=1,dive"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	IsPaused    *bool      `json:"is_paused,omitempty"`
}

// TaskDTO represents a task in a DAG
type TaskDTO struct {
	ID           string        `json:"id" validate:"required"`
	Name         string        `json:"name" validate:"required"`
	Type         string        `json:"type" validate:"required,oneof=bash http python go docker"`
	Command      string        `json:"command" validate:"required"`
	Dependencies []string      `json:"dependencies"`
	Retries      int           `json:"retries" validate:"min=0,max=10"`
	Timeout      time.Duration `json:"timeout" validate:"min=0"`
	SLA          time.Duration `json:"sla" validate:"min=0"`
}

// DAGResponse represents the response for a DAG
type DAGResponse struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Schedule    string        `json:"schedule"`
	Tasks       []TaskDTO     `json:"tasks"`
	StartDate   time.Time     `json:"start_date"`
	EndDate     *time.Time    `json:"end_date,omitempty"`
	Tags        []string      `json:"tags"`
	IsPaused    bool          `json:"is_paused"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// DAGListResponse represents a paginated list of DAGs
type DAGListResponse struct {
	DAGs       []DAGResponse  `json:"dags"`
	Pagination PaginationMeta `json:"pagination"`
}

// ToTaskDTO converts a models.Task to a TaskDTO
func ToTaskDTO(task models.Task) TaskDTO {
	return TaskDTO{
		ID:           task.ID,
		Name:         task.Name,
		Type:         string(task.Type),
		Command:      task.Command,
		Dependencies: task.Dependencies,
		Retries:      task.Retries,
		Timeout:      task.Timeout,
		SLA:          task.SLA,
	}
}

// ToTask converts a TaskDTO to a models.Task
func (t TaskDTO) ToTask() models.Task {
	return models.Task{
		ID:           t.ID,
		Name:         t.Name,
		Type:         models.TaskType(t.Type),
		Command:      t.Command,
		Dependencies: t.Dependencies,
		Retries:      t.Retries,
		Timeout:      t.Timeout,
		SLA:          t.SLA,
	}
}

// ToDAGResponse converts a models.DAG to a DAGResponse
func ToDAGResponse(dag *models.DAG) DAGResponse {
	tasks := make([]TaskDTO, len(dag.Tasks))
	for i, task := range dag.Tasks {
		tasks[i] = ToTaskDTO(task)
	}

	return DAGResponse{
		ID:          dag.ID,
		Name:        dag.Name,
		Description: dag.Description,
		Schedule:    dag.Schedule,
		Tasks:       tasks,
		StartDate:   dag.StartDate,
		EndDate:     dag.EndDate,
		Tags:        dag.Tags,
		IsPaused:    dag.IsPaused,
		CreatedAt:   dag.CreatedAt,
		UpdatedAt:   dag.UpdatedAt,
	}
}

// ToDAG converts a CreateDAGRequest to a models.DAG
func (r CreateDAGRequest) ToDAG() *models.DAG {
	tasks := make([]models.Task, len(r.Tasks))
	for i, taskDTO := range r.Tasks {
		tasks[i] = taskDTO.ToTask()
	}

	return &models.DAG{
		Name:        r.Name,
		Description: r.Description,
		Schedule:    r.Schedule,
		Tasks:       tasks,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
		Tags:        r.Tags,
		IsPaused:    r.IsPaused,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
