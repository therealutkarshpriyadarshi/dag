package models

import "time"

// DAG represents a Directed Acyclic Graph workflow definition
type DAG struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Schedule    string     `json:"schedule"` // Cron expression
	Tasks       []Task     `json:"tasks"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Tags        []string   `json:"tags"`
	IsPaused    bool       `json:"is_paused"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Task represents a single task within a DAG
type Task struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         TaskType      `json:"type"`
	Command      string        `json:"command"`
	Dependencies []string      `json:"dependencies"`
	Retries      int           `json:"retries"`
	Timeout      time.Duration `json:"timeout"`
	SLA          time.Duration `json:"sla"`
}

// TaskType defines the type of task executor to use
type TaskType string

const (
	TaskTypeBash   TaskType = "bash"
	TaskTypeHTTP   TaskType = "http"
	TaskTypePython TaskType = "python"
	TaskTypeGo     TaskType = "go"
)

// DAGRun represents a single execution instance of a DAG
type DAGRun struct {
	ID              string     `json:"id"`
	DAGID           string     `json:"dag_id"`
	ExecutionDate   time.Time  `json:"execution_date"`
	State           State      `json:"state"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	ExternalTrigger bool       `json:"external_trigger"`
}

// TaskInstance represents a single execution instance of a task
type TaskInstance struct {
	ID           string        `json:"id"`
	TaskID       string        `json:"task_id"`
	DAGRunID     string        `json:"dag_run_id"`
	State        State         `json:"state"`
	TryNumber    int           `json:"try_number"`
	MaxTries     int           `json:"max_tries"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	Duration     time.Duration `json:"duration"`
	Hostname     string        `json:"hostname"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// State represents the execution state of a DAG or task
type State string

const (
	StateQueued         State = "queued"
	StateRunning        State = "running"
	StateSuccess        State = "success"
	StateFailed         State = "failed"
	StateRetrying       State = "retrying"
	StateSkipped        State = "skipped"
	StateUpstreamFailed State = "upstream_failed"
)

// IsTerminal returns true if the state is a terminal state (no further transitions)
func (s State) IsTerminal() bool {
	return s == StateSuccess || s == StateFailed || s == StateSkipped
}
