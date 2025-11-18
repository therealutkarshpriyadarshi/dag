package dto

import (
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// TaskInstanceResponse represents the response for a task instance
type TaskInstanceResponse struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"task_id"`
	DAGRunID     string     `json:"dag_run_id"`
	State        string     `json:"state"`
	TryNumber    int        `json:"try_number"`
	MaxTries     int        `json:"max_tries"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Duration     string     `json:"duration,omitempty"`
	Hostname     string     `json:"hostname,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// TaskInstanceListResponse represents a paginated list of task instances
type TaskInstanceListResponse struct {
	TaskInstances []TaskInstanceResponse `json:"task_instances"`
	Pagination    PaginationMeta         `json:"pagination"`
}

// TaskLogResponse represents a task log entry
type TaskLogResponse struct {
	ID        string    `json:"id"`
	LogData   string    `json:"log_data"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskLogsResponse represents a list of task logs
type TaskLogsResponse struct {
	Logs       []TaskLogResponse `json:"logs"`
	Pagination PaginationMeta    `json:"pagination"`
}

// ToTaskInstanceResponse converts a models.TaskInstance to a TaskInstanceResponse
func ToTaskInstanceResponse(ti *models.TaskInstance) TaskInstanceResponse {
	var duration string
	if ti.Duration > 0 {
		duration = ti.Duration.String()
	}

	return TaskInstanceResponse{
		ID:           ti.ID,
		TaskID:       ti.TaskID,
		DAGRunID:     ti.DAGRunID,
		State:        string(ti.State),
		TryNumber:    ti.TryNumber,
		MaxTries:     ti.MaxTries,
		StartDate:    ti.StartDate,
		EndDate:      ti.EndDate,
		Duration:     duration,
		Hostname:     ti.Hostname,
		ErrorMessage: ti.ErrorMessage,
	}
}
