package dto

import (
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// TriggerDAGRequest represents the request to manually trigger a DAG
type TriggerDAGRequest struct {
	ExecutionDate *time.Time             `json:"execution_date,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// DAGRunResponse represents the response for a DAG run
type DAGRunResponse struct {
	ID              string     `json:"id"`
	DAGID           string     `json:"dag_id"`
	ExecutionDate   time.Time  `json:"execution_date"`
	State           string     `json:"state"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	ExternalTrigger bool       `json:"external_trigger"`
}

// DAGRunListResponse represents a paginated list of DAG runs
type DAGRunListResponse struct {
	DAGRuns    []DAGRunResponse `json:"dag_runs"`
	Pagination PaginationMeta   `json:"pagination"`
}

// DAGRunDetailResponse represents detailed information about a DAG run
type DAGRunDetailResponse struct {
	DAGRunResponse
	Tasks []TaskInstanceResponse `json:"tasks"`
}

// ToDAGRunResponse converts a models.DAGRun to a DAGRunResponse
func ToDAGRunResponse(run *models.DAGRun) DAGRunResponse {
	return DAGRunResponse{
		ID:              run.ID,
		DAGID:           run.DAGID,
		ExecutionDate:   run.ExecutionDate,
		State:           string(run.State),
		StartDate:       run.StartDate,
		EndDate:         run.EndDate,
		ExternalTrigger: run.ExternalTrigger,
	}
}
