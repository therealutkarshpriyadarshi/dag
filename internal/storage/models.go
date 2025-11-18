package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// JSONB is a custom type for JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// StringArray is a custom type for string array columns
type StringArray []string

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, s)
}

// DAGModel represents the database model for a DAG
type DAGModel struct {
	ID          uuid.UUID   `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string      `gorm:"type:varchar(255);unique;not null;index:idx_dags_name"`
	Description string      `gorm:"type:text"`
	Schedule    string      `gorm:"type:varchar(100)"`
	IsPaused    bool        `gorm:"default:false;index:idx_dags_is_paused"`
	Tags        StringArray `gorm:"type:jsonb;default:'[]'"`
	StartDate   time.Time   `gorm:"not null"`
	EndDate     *time.Time
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for DAGModel
func (DAGModel) TableName() string {
	return "dags"
}

// DAGRunModel represents the database model for a DAG run
type DAGRunModel struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	DAGID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_dag_runs_dag_id"`
	ExecutionDate   time.Time  `gorm:"not null;index:idx_dag_runs_execution_date"`
	State           string     `gorm:"type:varchar(50);not null;default:'queued';index:idx_dag_runs_state"`
	StartDate       *time.Time
	EndDate         *time.Time
	ExternalTrigger bool      `gorm:"default:false"`
	CreatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_dag_runs_created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Version         int       `gorm:"not null;default:1"` // For optimistic locking

	// Relationships
	DAG           DAGModel        `gorm:"foreignKey:DAGID"`
	TaskInstances []TaskInstanceModel `gorm:"foreignKey:DAGRunID"`
}

// TableName specifies the table name for DAGRunModel
func (DAGRunModel) TableName() string {
	return "dag_runs"
}

// TaskInstanceModel represents the database model for a task instance
type TaskInstanceModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TaskID       string     `gorm:"type:varchar(255);not null;index:idx_task_instances_task_id"`
	DAGRunID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_task_instances_dag_run_id"`
	State        string     `gorm:"type:varchar(50);not null;default:'queued';index:idx_task_instances_state"`
	TryNumber    int        `gorm:"not null;default:1"`
	MaxTries     int        `gorm:"not null;default:1"`
	StartDate    *time.Time
	EndDate      *time.Time
	Duration     *int64     `gorm:"type:bigint"` // Duration in nanoseconds
	Hostname     string     `gorm:"type:varchar(255)"`
	ErrorMessage string     `gorm:"type:text"`
	CreatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_task_instances_created_at"`
	UpdatedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	Version      int        `gorm:"not null;default:1"` // For optimistic locking

	// Relationships
	DAGRun DAGRunModel   `gorm:"foreignKey:DAGRunID"`
	Logs   []TaskLogModel `gorm:"foreignKey:TaskInstanceID"`
}

// TableName specifies the table name for TaskInstanceModel
func (TaskInstanceModel) TableName() string {
	return "task_instances"
}

// TaskLogModel represents the database model for task logs
type TaskLogModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	TaskInstanceID uuid.UUID `gorm:"type:uuid;not null;index:idx_task_logs_task_instance_id"`
	LogData        string    `gorm:"type:text;not null"`
	Timestamp      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_task_logs_timestamp"`

	// Relationships
	TaskInstance TaskInstanceModel `gorm:"foreignKey:TaskInstanceID"`
}

// TableName specifies the table name for TaskLogModel
func (TaskLogModel) TableName() string {
	return "task_logs"
}

// ToDAG converts a DAGModel to a models.DAG
func (d *DAGModel) ToDAG() *models.DAG {
	return &models.DAG{
		ID:          d.ID.String(),
		Name:        d.Name,
		Description: d.Description,
		Schedule:    d.Schedule,
		Tasks:       []models.Task{}, // Tasks need to be loaded separately
		StartDate:   d.StartDate,
		EndDate:     d.EndDate,
		Tags:        []string(d.Tags),
		IsPaused:    d.IsPaused,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// FromDAG converts a models.DAG to a DAGModel
func FromDAG(d *models.DAG) (*DAGModel, error) {
	id, err := uuid.Parse(d.ID)
	if err != nil {
		id = uuid.New()
	}

	return &DAGModel{
		ID:          id,
		Name:        d.Name,
		Description: d.Description,
		Schedule:    d.Schedule,
		IsPaused:    d.IsPaused,
		Tags:        StringArray(d.Tags),
		StartDate:   d.StartDate,
		EndDate:     d.EndDate,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}, nil
}

// ToDAGRun converts a DAGRunModel to a models.DAGRun
func (dr *DAGRunModel) ToDAGRun() *models.DAGRun {
	return &models.DAGRun{
		ID:              dr.ID.String(),
		DAGID:           dr.DAGID.String(),
		ExecutionDate:   dr.ExecutionDate,
		State:           models.State(dr.State),
		StartDate:       dr.StartDate,
		EndDate:         dr.EndDate,
		ExternalTrigger: dr.ExternalTrigger,
	}
}

// FromDAGRun converts a models.DAGRun to a DAGRunModel
func FromDAGRun(dr *models.DAGRun) (*DAGRunModel, error) {
	id, err := uuid.Parse(dr.ID)
	if err != nil {
		id = uuid.New()
	}

	dagID, err := uuid.Parse(dr.DAGID)
	if err != nil {
		return nil, err
	}

	return &DAGRunModel{
		ID:              id,
		DAGID:           dagID,
		ExecutionDate:   dr.ExecutionDate,
		State:           string(dr.State),
		StartDate:       dr.StartDate,
		EndDate:         dr.EndDate,
		ExternalTrigger: dr.ExternalTrigger,
		Version:         1,
	}, nil
}

// ToTaskInstance converts a TaskInstanceModel to a models.TaskInstance
func (ti *TaskInstanceModel) ToTaskInstance() *models.TaskInstance {
	var duration time.Duration
	if ti.Duration != nil {
		duration = time.Duration(*ti.Duration)
	}

	return &models.TaskInstance{
		ID:           ti.ID.String(),
		TaskID:       ti.TaskID,
		DAGRunID:     ti.DAGRunID.String(),
		State:        models.State(ti.State),
		TryNumber:    ti.TryNumber,
		MaxTries:     ti.MaxTries,
		StartDate:    ti.StartDate,
		EndDate:      ti.EndDate,
		Duration:     duration,
		Hostname:     ti.Hostname,
		ErrorMessage: ti.ErrorMessage,
	}
}

// FromTaskInstance converts a models.TaskInstance to a TaskInstanceModel
func FromTaskInstance(ti *models.TaskInstance) (*TaskInstanceModel, error) {
	id, err := uuid.Parse(ti.ID)
	if err != nil {
		id = uuid.New()
	}

	dagRunID, err := uuid.Parse(ti.DAGRunID)
	if err != nil {
		return nil, err
	}

	var duration *int64
	if ti.Duration > 0 {
		d := int64(ti.Duration)
		duration = &d
	}

	return &TaskInstanceModel{
		ID:           id,
		TaskID:       ti.TaskID,
		DAGRunID:     dagRunID,
		State:        string(ti.State),
		TryNumber:    ti.TryNumber,
		MaxTries:     ti.MaxTries,
		StartDate:    ti.StartDate,
		EndDate:      ti.EndDate,
		Duration:     duration,
		Hostname:     ti.Hostname,
		ErrorMessage: ti.ErrorMessage,
		Version:      1,
	}, nil
}
