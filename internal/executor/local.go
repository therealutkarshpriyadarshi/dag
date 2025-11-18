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

// TaskExecution represents a task ready for execution
type TaskExecution struct {
	Task         *models.Task
	TaskInstance *models.TaskInstance
	DAGRun       *models.DAGRun
	DAG          *models.DAG
}

// LocalExecutor executes tasks using a local worker pool
type LocalExecutor struct {
	taskRepo      storage.TaskInstanceRepository
	dagRunRepo    storage.DAGRunRepository
	stateMachine  *state.Machine
	taskExecutors map[models.TaskType]TaskExecutor
	config        *ExecutorConfig

	taskQueue chan *TaskExecution
	workers   []*worker
	running   bool
	mu        sync.RWMutex
	wg        sync.WaitGroup

	status ExecutorStatus
}

// worker represents a worker goroutine
type worker struct {
	id       int
	executor *LocalExecutor
	stopChan chan struct{}
}

// NewLocalExecutor creates a new local executor with worker pool
func NewLocalExecutor(
	taskRepo storage.TaskInstanceRepository,
	dagRunRepo storage.DAGRunRepository,
	stateMachine *state.Machine,
	config *ExecutorConfig,
) *LocalExecutor {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	return &LocalExecutor{
		taskRepo:      taskRepo,
		dagRunRepo:    dagRunRepo,
		stateMachine:  stateMachine,
		taskExecutors: make(map[models.TaskType]TaskExecutor),
		config:        config,
		taskQueue:     make(chan *TaskExecution, config.QueueSize),
		workers:       make([]*worker, 0, config.WorkerCount),
		running:       false,
		status: ExecutorStatus{
			Running:       false,
			ActiveTasks:   0,
			CompletedTasks: 0,
			FailedTasks:   0,
			WorkerCount:   config.WorkerCount,
			QueueDepth:    0,
		},
	}
}

// RegisterTaskExecutor registers a task executor for a specific task type
func (e *LocalExecutor) RegisterTaskExecutor(executor TaskExecutor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.taskExecutors[executor.Type()] = executor
}

// Start initializes the executor and starts worker goroutines
func (e *LocalExecutor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("executor already running")
	}

	e.running = true
	e.status.Running = true

	// Start worker goroutines
	for i := 0; i < e.config.WorkerCount; i++ {
		w := &worker{
			id:       i,
			executor: e,
			stopChan: make(chan struct{}),
		}
		e.workers = append(e.workers, w)
		e.wg.Add(1)
		go w.run(ctx)
	}

	log.Printf("Local executor started with %d workers", e.config.WorkerCount)
	return nil
}

// Stop gracefully shuts down the executor
func (e *LocalExecutor) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()

	log.Println("Stopping local executor...")

	// Stop all workers
	for _, w := range e.workers {
		close(w.stopChan)
	}

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
	case <-time.After(e.config.ShutdownTimeout):
		log.Println("Shutdown timeout reached, forcing stop")
	}

	close(e.taskQueue)

	e.mu.Lock()
	e.status.Running = false
	e.mu.Unlock()

	log.Println("Local executor stopped")
	return nil
}

// GetStatus returns the current executor status
func (e *LocalExecutor) GetStatus() ExecutorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := e.status
	status.QueueDepth = len(e.taskQueue)
	return status
}

// Execute submits a DAG run for execution
func (e *LocalExecutor) Execute(ctx context.Context, dagRun *models.DAGRun, dag *models.DAG) error {
	if !e.running {
		return fmt.Errorf("executor is not running")
	}

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

	// Start a goroutine to manage task scheduling for this DAG run
	go e.scheduleTasks(ctx, dagRun, dag, graph, taskInstances)

	return nil
}

// scheduleTasks manages the scheduling of tasks for a DAG run
func (e *LocalExecutor) scheduleTasks(
	ctx context.Context,
	dagRun *models.DAGRun,
	dag *models.DAG,
	graph *dag.Graph,
	taskInstances map[string]*models.TaskInstance,
) {
	completedTasks := make(map[string]bool)
	failedTasks := make(map[string]bool)
	submittedTasks := make(map[string]bool)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("DAG run %s scheduling cancelled", dagRun.ID)
			return
		case <-ticker.C:
			// Check if all tasks are done
			if len(completedTasks)+len(failedTasks) == len(dag.Tasks) {
				// Update final DAG run state
				e.finalizeDagRun(ctx, dagRun, len(failedTasks) > 0)
				return
			}

			// Find tasks ready to execute
			readyTasks := graph.GetParallelTasks(completedTasks)

			for _, taskID := range readyTasks {
				if submittedTasks[taskID] || failedTasks[taskID] {
					continue
				}

				task := graph.GetTask(taskID)
				if task == nil {
					continue
				}

				// Check if upstream tasks failed
				upstreamFailed := false
				for _, depID := range task.Dependencies {
					if failedTasks[depID] {
						upstreamFailed = true
						break
					}
				}

				taskInstance := taskInstances[taskID]

				if upstreamFailed {
					// Mark as upstream failed
					e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateQueued, models.StateUpstreamFailed)
					failedTasks[taskID] = true
					continue
				}

				// Submit task for execution
				execution := &TaskExecution{
					Task:         task,
					TaskInstance: taskInstance,
					DAGRun:       dagRun,
					DAG:          dag,
				}

				select {
				case e.taskQueue <- execution:
					submittedTasks[taskID] = true
					log.Printf("Submitted task %s for execution", taskID)
				case <-ctx.Done():
					return
				}
			}

			// Update completed/failed tasks status
			for taskID := range submittedTasks {
				if completedTasks[taskID] || failedTasks[taskID] {
					continue
				}

				taskInstance, err := e.taskRepo.GetByID(ctx, taskInstances[taskID].ID)
				if err != nil {
					continue
				}

				if taskInstance.State == models.StateSuccess {
					completedTasks[taskID] = true
				} else if taskInstance.State.IsTerminal() {
					failedTasks[taskID] = true
				}
			}
		}
	}
}

// finalizeDagRun updates the final state of a DAG run
func (e *LocalExecutor) finalizeDagRun(ctx context.Context, dagRun *models.DAGRun, failed bool) {
	endTime := time.Now()
	dagRun.EndDate = &endTime

	finalState := models.StateSuccess
	if failed {
		finalState = models.StateFailed
	}

	if err := e.dagRunRepo.UpdateState(ctx, dagRun.ID, models.StateRunning, finalState); err != nil {
		log.Printf("Failed to update final DAG run state: %v", err)
	}

	log.Printf("DAG run %s completed with state %s", dagRun.ID, finalState)
}

// worker.run executes tasks from the queue
func (w *worker) run(ctx context.Context) {
	defer w.executor.wg.Done()

	log.Printf("Worker %d started", w.id)

	for {
		select {
		case <-w.stopChan:
			log.Printf("Worker %d stopped", w.id)
			return
		case execution, ok := <-w.executor.taskQueue:
			if !ok {
				log.Printf("Worker %d: task queue closed", w.id)
				return
			}

			w.executeTask(ctx, execution)
		}
	}
}

// executeTask executes a single task
func (w *worker) executeTask(ctx context.Context, execution *TaskExecution) {
	log.Printf("Worker %d executing task %s", w.id, execution.Task.ID)

	w.executor.mu.Lock()
	executor, ok := w.executor.taskExecutors[execution.Task.Type]
	w.executor.mu.Unlock()

	if !ok {
		log.Printf("No executor registered for task type %s", execution.Task.Type)
		w.executor.taskRepo.UpdateState(ctx, execution.TaskInstance.ID, models.StateQueued, models.StateFailed)
		return
	}

	// Update task state to running
	if err := w.executor.taskRepo.UpdateState(ctx, execution.TaskInstance.ID, models.StateQueued, models.StateRunning); err != nil {
		log.Printf("Failed to update task state to running: %v", err)
		return
	}

	w.executor.mu.Lock()
	w.executor.status.ActiveTasks++
	w.executor.mu.Unlock()

	// Execute with timeout
	taskCtx := ctx
	if execution.Task.Timeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, execution.Task.Timeout)
		defer cancel()
	}

	// Execute the task
	result := executor.Execute(taskCtx, execution.Task, execution.TaskInstance)

	w.executor.mu.Lock()
	w.executor.status.ActiveTasks--
	if result.State == models.StateSuccess {
		w.executor.status.CompletedTasks++
	} else {
		w.executor.status.FailedTasks++
	}
	w.executor.mu.Unlock()

	// Update task instance with result
	execution.TaskInstance.State = result.State
	execution.TaskInstance.StartDate = &result.StartTime
	execution.TaskInstance.EndDate = &result.EndTime
	execution.TaskInstance.Duration = result.EndTime.Sub(result.StartTime)
	execution.TaskInstance.Hostname = result.Hostname
	execution.TaskInstance.ErrorMessage = result.ErrorMessage

	// Update state in database
	if err := w.executor.taskRepo.UpdateState(ctx, execution.TaskInstance.ID, models.StateRunning, result.State); err != nil {
		log.Printf("Failed to update task state: %v", err)
	}

	log.Printf("Worker %d completed task %s with state %s", w.id, execution.Task.ID, result.State)
}
