package storage

import (
	"context"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// DAGRepository defines the interface for DAG persistence
type DAGRepository interface {
	Create(ctx context.Context, dag *models.DAG) error
	Get(ctx context.Context, id string) (*models.DAG, error)
	GetByName(ctx context.Context, name string) (*models.DAG, error)
	List(ctx context.Context, filters DAGFilters) ([]*models.DAG, error)
	Update(ctx context.Context, dag *models.DAG) error
	Delete(ctx context.Context, id string) error
	Pause(ctx context.Context, id string) error
	Unpause(ctx context.Context, id string) error
}

// DAGFilters defines filters for listing DAGs
type DAGFilters struct {
	IsPaused *bool
	Tags     []string
	Limit    int
	Offset   int
}

// DAGRunRepository defines the interface for DAG run persistence
type DAGRunRepository interface {
	Create(ctx context.Context, run *models.DAGRun) error
	Get(ctx context.Context, id string) (*models.DAGRun, error)
	List(ctx context.Context, filters DAGRunFilters) ([]*models.DAGRun, error)
	Update(ctx context.Context, run *models.DAGRun) error
	UpdateState(ctx context.Context, id string, oldState, newState models.State) error
	Delete(ctx context.Context, id string) error
	GetLatestRun(ctx context.Context, dagID string) (*models.DAGRun, error)
}

// DAGRunFilters defines filters for listing DAG runs
type DAGRunFilters struct {
	DAGID  string
	State  *models.State
	After  *time.Time
	Before *time.Time
	Limit  int
	Offset int
}

// TaskInstanceRepository defines the interface for task instance persistence
type TaskInstanceRepository interface {
	Create(ctx context.Context, instance *models.TaskInstance) error
	Get(ctx context.Context, id string) (*models.TaskInstance, error)
	GetByTaskID(ctx context.Context, dagRunID, taskID string) (*models.TaskInstance, error)
	List(ctx context.Context, filters TaskInstanceFilters) ([]*models.TaskInstance, error)
	Update(ctx context.Context, instance *models.TaskInstance) error
	UpdateState(ctx context.Context, id string, oldState, newState models.State) error
	Delete(ctx context.Context, id string) error
	ListByDAGRun(ctx context.Context, dagRunID string) ([]*models.TaskInstance, error)
}

// TaskInstanceFilters defines filters for listing task instances
type TaskInstanceFilters struct {
	DAGRunID string
	TaskID   string
	State    *models.State
	Limit    int
	Offset   int
}

// TaskLogRepository defines the interface for task log persistence
type TaskLogRepository interface {
	Create(ctx context.Context, taskInstanceID, logData string) error
	List(ctx context.Context, taskInstanceID string, limit int) ([]TaskLogModel, error)
	Delete(ctx context.Context, taskInstanceID string) error
}
