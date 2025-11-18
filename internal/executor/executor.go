package executor

import (
	"context"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Executor is the main interface for executing DAG runs
type Executor interface {
	// Execute submits a DAG run for execution
	Execute(ctx context.Context, dagRun *models.DAGRun, dag *models.DAG) error

	// Start initializes the executor and starts processing tasks
	Start(ctx context.Context) error

	// Stop gracefully shuts down the executor
	Stop(ctx context.Context) error

	// GetStatus returns the current status of the executor
	GetStatus() ExecutorStatus
}

// TaskExecutor executes individual tasks
type TaskExecutor interface {
	// Execute runs a single task and returns the result
	Execute(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) *TaskResult

	// Type returns the task type this executor handles
	Type() models.TaskType
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	State        models.State
	Output       string
	ErrorMessage string
	StartTime    time.Time
	EndTime      time.Time
	Hostname     string
}

// ExecutorStatus represents the current status of an executor
type ExecutorStatus struct {
	Running       bool
	ActiveTasks   int
	CompletedTasks int
	FailedTasks   int
	WorkerCount   int
	QueueDepth    int
}

// ExecutorConfig holds configuration for executor initialization
type ExecutorConfig struct {
	WorkerCount      int
	QueueSize        int
	TaskTimeout      time.Duration
	ShutdownTimeout  time.Duration
	EnableDocker     bool
	MaxMemoryMB      int64
	MaxCPUPercent    int
}

// DefaultExecutorConfig returns default configuration
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		WorkerCount:     5,
		QueueSize:       100,
		TaskTimeout:     30 * time.Minute,
		ShutdownTimeout: 30 * time.Second,
		EnableDocker:    false,
		MaxMemoryMB:     1024,
		MaxCPUPercent:   100,
	}
}
