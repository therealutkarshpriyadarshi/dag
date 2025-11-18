package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/internal/dag"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// SequentialExecutor executes tasks sequentially in topological order
// This is a simple executor for testing and development
type SequentialExecutor struct {
	taskRepo         storage.TaskInstanceRepository
	dagRunRepo       storage.DAGRunRepository
	stateMachine     *state.Machine
	taskExecutors    map[models.TaskType]TaskExecutor
	status           ExecutorStatus
	mu               sync.RWMutex
}

// NewSequentialExecutor creates a new sequential executor
func NewSequentialExecutor(
	taskRepo storage.TaskInstanceRepository,
	dagRunRepo storage.DAGRunRepository,
	stateMachine *state.Machine,
) *SequentialExecutor {
	return &SequentialExecutor{
		taskRepo:      taskRepo,
		dagRunRepo:    dagRunRepo,
		stateMachine:  stateMachine,
		taskExecutors: make(map[models.TaskType]TaskExecutor),
		status: ExecutorStatus{
			Running:       false,
			ActiveTasks:   0,
			CompletedTasks: 0,
			FailedTasks:   0,
			WorkerCount:   1,
			QueueDepth:    0,
		},
	}
}

// RegisterTaskExecutor registers a task executor for a specific task type
func (e *SequentialExecutor) RegisterTaskExecutor(executor TaskExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.taskExecutors[executor.Type()] = executor
}

// Start initializes the executor
func (e *SequentialExecutor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Running = true
	log.Println("Sequential executor started")
	return nil
}

// Stop gracefully shuts down the executor
func (e *SequentialExecutor) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.Running = false
	log.Println("Sequential executor stopped")
	return nil
}

// GetStatus returns the current executor status
func (e *SequentialExecutor) GetStatus() ExecutorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// Execute executes a DAG run sequentially
func (e *SequentialExecutor) Execute(ctx context.Context, dagRun *models.DAGRun, dag *models.DAG) error {
	// Update DAG run state to running
	now := time.Now()
	dagRun.StartDate = &now
	if err := e.dagRunRepo.UpdateState(ctx, dagRun.ID, models.StateQueued, models.StateRunning); err != nil {
		return fmt.Errorf("failed to update DAG run state: %w", err)
	}

	// Build dependency graph
	graph := dag.NewGraph(dag)

	// Create task instances for all tasks
	taskInstances := make(map[string]*models.TaskInstance)
	for _, task := range dag.Tasks {
		taskInstance := &models.TaskInstance{
			TaskID:    task.ID,
			DAGRunID:  dagRun.ID,
			State:     models.StateQueued,
			TryNumber: 1,
			MaxTries:  task.Retries + 1,
		}

		created, err := e.taskRepo.Create(ctx, taskInstance)
		if err != nil {
			return fmt.Errorf("failed to create task instance for %s: %w", task.ID, err)
		}
		taskInstances[task.ID] = created
	}

	// Get topological order
	order, err := dag.TopologicalSort(dag)
	if err != nil {
		return fmt.Errorf("failed to get topological order: %w", err)
	}

	// Execute tasks in topological order
	completedTasks := make(map[string]bool)
	failedDAGRun := false

	for _, taskID := range order {
		// Check if upstream tasks failed
		task := graph.GetTask(taskID)
		if task == nil {
			return fmt.Errorf("task %s not found in graph", taskID)
		}

		upstreamFailed := false
		for _, depID := range task.Dependencies {
			if !completedTasks[depID] {
				upstreamFailed = true
				break
			}
		}

		taskInstance := taskInstances[taskID]

		if upstreamFailed {
			// Skip this task due to upstream failure
			if err := e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateQueued, models.StateUpstreamFailed); err != nil {
				log.Printf("Failed to update task %s state to upstream_failed: %v", taskID, err)
			}
			continue
		}

		// Execute the task
		if err := e.executeTask(ctx, task, taskInstance); err != nil {
			log.Printf("Failed to execute task %s: %v", taskID, err)
			failedDAGRun = true
			continue
		}

		// Check if task succeeded
		updated, err := e.taskRepo.GetByID(ctx, taskInstance.ID)
		if err != nil {
			log.Printf("Failed to get updated task instance %s: %v", taskID, err)
			continue
		}

		if updated.State == models.StateSuccess {
			completedTasks[taskID] = true
		} else {
			failedDAGRun = true
		}
	}

	// Update DAG run final state
	endTime := time.Now()
	dagRun.EndDate = &endTime

	finalState := models.StateSuccess
	if failedDAGRun {
		finalState = models.StateFailed
	}

	if err := e.dagRunRepo.UpdateState(ctx, dagRun.ID, models.StateRunning, finalState); err != nil {
		return fmt.Errorf("failed to update final DAG run state: %w", err)
	}

	return nil
}

// executeTask executes a single task
func (e *SequentialExecutor) executeTask(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) error {
	e.mu.Lock()
	executor, ok := e.taskExecutors[task.Type]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("no executor registered for task type %s", task.Type)
	}

	// Update task state to running
	if err := e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateQueued, models.StateRunning); err != nil {
		return fmt.Errorf("failed to update task state to running: %w", err)
	}

	e.mu.Lock()
	e.status.ActiveTasks++
	e.mu.Unlock()

	// Execute with timeout
	taskCtx := ctx
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// Execute the task
	result := executor.Execute(taskCtx, task, taskInstance)

	e.mu.Lock()
	e.status.ActiveTasks--
	if result.State == models.StateSuccess {
		e.status.CompletedTasks++
	} else {
		e.status.FailedTasks++
	}
	e.mu.Unlock()

	// Update task instance with result
	taskInstance.State = result.State
	taskInstance.StartDate = &result.StartTime
	taskInstance.EndDate = &result.EndTime
	taskInstance.Duration = result.EndTime.Sub(result.StartTime)
	taskInstance.Hostname = result.Hostname
	taskInstance.ErrorMessage = result.ErrorMessage

	// Update state in database
	if err := e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateRunning, result.State); err != nil {
		return fmt.Errorf("failed to update task state: %w", err)
	}

	return nil
}
