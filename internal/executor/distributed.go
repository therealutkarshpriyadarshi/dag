package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/therealutkarshpriyadarshi/dag/internal/dag"
	"github.com/therealutkarshpriyadarshi/dag/internal/state"
	"github.com/therealutkarshpriyadarshi/dag/internal/storage"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

const (
	// NATS stream names
	TasksPendingStream = "TASKS_PENDING"
	TasksResultsStream = "TASKS_RESULTS"

	// Subject names
	TasksPendingSubject = "tasks.pending"
	TasksResultsSubject = "tasks.results"
	WorkerHeartbeatSubject = "workers.heartbeat"
)

// DistributedExecutor executes tasks across multiple workers using NATS
type DistributedExecutor struct {
	nc            *nats.Conn
	js            nats.JetStreamContext
	taskRepo      storage.TaskInstanceRepository
	dagRunRepo    storage.DAGRunRepository
	stateMachine  *state.Machine
	config        *ExecutorConfig

	// Worker management
	workers     map[string]*WorkerInfo
	workersMu   sync.RWMutex

	// Subscriptions
	resultSub   *nats.Subscription
	heartbeatSub *nats.Subscription

	running bool
	mu      sync.RWMutex
	wg      sync.WaitGroup

	status ExecutorStatus
}

// WorkerInfo represents information about a worker
type WorkerInfo struct {
	ID           string
	Hostname     string
	LastHeartbeat time.Time
	ActiveTasks  int
}

// TaskMessage represents a task to be executed
type TaskMessage struct {
	TaskInstanceID string        `json:"task_instance_id"`
	TaskID         string        `json:"task_id"`
	DAGRunID       string        `json:"dag_run_id"`
	DAGID          string        `json:"dag_id"`
	TaskType       string        `json:"task_type"`
	Command        string        `json:"command"`
	Timeout        time.Duration `json:"timeout"`
	Retries        int           `json:"retries"`
}

// TaskResultMessage represents the result of a task execution
type TaskResultMessage struct {
	TaskInstanceID string        `json:"task_instance_id"`
	WorkerID       string        `json:"worker_id"`
	State          string        `json:"state"`
	Output         string        `json:"output"`
	ErrorMessage   string        `json:"error_message"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Hostname       string        `json:"hostname"`
}

// WorkerHeartbeat represents a worker heartbeat message
type WorkerHeartbeat struct {
	WorkerID    string    `json:"worker_id"`
	Hostname    string    `json:"hostname"`
	ActiveTasks int       `json:"active_tasks"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewDistributedExecutor creates a new distributed executor
func NewDistributedExecutor(
	natsURL string,
	taskRepo storage.TaskInstanceRepository,
	dagRunRepo storage.DAGRunRepository,
	stateMachine *state.Machine,
	config *ExecutorConfig,
) (*DistributedExecutor, error) {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	executor := &DistributedExecutor{
		nc:           nc,
		js:           js,
		taskRepo:     taskRepo,
		dagRunRepo:   dagRunRepo,
		stateMachine: stateMachine,
		config:       config,
		workers:      make(map[string]*WorkerInfo),
		running:      false,
		status: ExecutorStatus{
			Running:       false,
			ActiveTasks:   0,
			CompletedTasks: 0,
			FailedTasks:   0,
			WorkerCount:   0,
			QueueDepth:    0,
		},
	}

	// Initialize streams
	if err := executor.initStreams(); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return executor, nil
}

// initStreams initializes NATS JetStream streams
func (e *DistributedExecutor) initStreams() error {
	// Create pending tasks stream
	_, err := e.js.AddStream(&nats.StreamConfig{
		Name:      TasksPendingStream,
		Subjects:  []string{TasksPendingSubject},
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("failed to create pending tasks stream: %w", err)
	}

	// Create results stream
	_, err = e.js.AddStream(&nats.StreamConfig{
		Name:      TasksResultsStream,
		Subjects:  []string{TasksResultsSubject},
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return fmt.Errorf("failed to create results stream: %w", err)
	}

	return nil
}

// Start initializes the executor and starts listening for results
func (e *DistributedExecutor) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("executor already running")
	}

	e.running = true
	e.status.Running = true

	// Subscribe to task results
	var err error
	e.resultSub, err = e.js.Subscribe(TasksResultsSubject, e.handleTaskResult,
		nats.Durable("executor-results"),
		nats.ManualAck(),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to results: %w", err)
	}

	// Subscribe to worker heartbeats
	e.heartbeatSub, err = e.nc.Subscribe(WorkerHeartbeatSubject, e.handleWorkerHeartbeat)
	if err != nil {
		e.resultSub.Unsubscribe()
		return fmt.Errorf("failed to subscribe to heartbeats: %w", err)
	}

	// Start worker monitoring
	e.wg.Add(1)
	go e.monitorWorkers(ctx)

	log.Println("Distributed executor started")
	return nil
}

// Stop gracefully shuts down the executor
func (e *DistributedExecutor) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()

	log.Println("Stopping distributed executor...")

	// Unsubscribe from streams
	if e.resultSub != nil {
		e.resultSub.Unsubscribe()
	}
	if e.heartbeatSub != nil {
		e.heartbeatSub.Unsubscribe()
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All monitoring stopped gracefully")
	case <-time.After(e.config.ShutdownTimeout):
		log.Println("Shutdown timeout reached")
	}

	// Close NATS connection
	e.nc.Close()

	e.mu.Lock()
	e.status.Running = false
	e.mu.Unlock()

	log.Println("Distributed executor stopped")
	return nil
}

// GetStatus returns the current executor status
func (e *DistributedExecutor) GetStatus() ExecutorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	e.workersMu.RLock()
	defer e.workersMu.RUnlock()

	status := e.status
	status.WorkerCount = len(e.workers)

	// Get queue depth
	if stream, err := e.js.StreamInfo(TasksPendingStream); err == nil {
		status.QueueDepth = int(stream.State.Msgs)
	}

	return status
}

// Execute submits a DAG run for execution
func (e *DistributedExecutor) Execute(ctx context.Context, dagRun *models.DAGRun, dag *models.DAG) error {
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
	e.wg.Add(1)
	go e.scheduleTasks(ctx, dagRun, dag, graph, taskInstances)

	return nil
}

// scheduleTasks manages the scheduling of tasks for a DAG run
func (e *DistributedExecutor) scheduleTasks(
	ctx context.Context,
	dagRun *models.DAGRun,
	dag *models.DAG,
	graph *dag.Graph,
	taskInstances map[string]*models.TaskInstance,
) {
	defer e.wg.Done()

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
					e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateQueued, models.StateUpstreamFailed)
					failedTasks[taskID] = true
					continue
				}

				// Publish task to NATS
				if err := e.publishTask(task, taskInstance, dagRun); err != nil {
					log.Printf("Failed to publish task %s: %v", taskID, err)
					continue
				}

				submittedTasks[taskID] = true
				log.Printf("Published task %s to NATS", taskID)
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

// publishTask publishes a task to NATS for execution
func (e *DistributedExecutor) publishTask(task *models.Task, taskInstance *models.TaskInstance, dagRun *models.DAGRun) error {
	msg := &TaskMessage{
		TaskInstanceID: taskInstance.ID,
		TaskID:         task.ID,
		DAGRunID:       dagRun.ID,
		DAGID:          dagRun.DAGID,
		TaskType:       string(task.Type),
		Command:        task.Command,
		Timeout:        task.Timeout,
		Retries:        task.Retries,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal task message: %w", err)
	}

	_, err = e.js.Publish(TasksPendingSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	return nil
}

// handleTaskResult processes task results from workers
func (e *DistributedExecutor) handleTaskResult(msg *nats.Msg) {
	var result TaskResultMessage
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		log.Printf("Failed to unmarshal task result: %v", err)
		msg.Nak()
		return
	}

	ctx := context.Background()

	// Update task instance
	taskInstance, err := e.taskRepo.GetByID(ctx, result.TaskInstanceID)
	if err != nil {
		log.Printf("Failed to get task instance %s: %v", result.TaskInstanceID, err)
		msg.Nak()
		return
	}

	// Update state
	taskInstance.State = models.State(result.State)
	taskInstance.StartDate = &result.StartTime
	taskInstance.EndDate = &result.EndTime
	taskInstance.Duration = result.EndTime.Sub(result.StartTime)
	taskInstance.Hostname = result.Hostname
	taskInstance.ErrorMessage = result.ErrorMessage

	if err := e.taskRepo.UpdateState(ctx, taskInstance.ID, models.StateRunning, models.State(result.State)); err != nil {
		log.Printf("Failed to update task state: %v", err)
		msg.Nak()
		return
	}

	// Update statistics
	e.mu.Lock()
	if result.State == string(models.StateSuccess) {
		e.status.CompletedTasks++
	} else {
		e.status.FailedTasks++
	}
	e.mu.Unlock()

	log.Printf("Task %s completed with state %s", result.TaskInstanceID, result.State)
	msg.Ack()
}

// handleWorkerHeartbeat processes worker heartbeat messages
func (e *DistributedExecutor) handleWorkerHeartbeat(msg *nats.Msg) {
	var heartbeat WorkerHeartbeat
	if err := json.Unmarshal(msg.Data, &heartbeat); err != nil {
		log.Printf("Failed to unmarshal heartbeat: %v", err)
		return
	}

	e.workersMu.Lock()
	e.workers[heartbeat.WorkerID] = &WorkerInfo{
		ID:           heartbeat.WorkerID,
		Hostname:     heartbeat.Hostname,
		LastHeartbeat: heartbeat.Timestamp,
		ActiveTasks:  heartbeat.ActiveTasks,
	}
	e.workersMu.Unlock()
}

// monitorWorkers monitors worker health and removes dead workers
func (e *DistributedExecutor) monitorWorkers(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.workersMu.Lock()
			now := time.Now()
			for id, worker := range e.workers {
				if now.Sub(worker.LastHeartbeat) > 30*time.Second {
					log.Printf("Worker %s is dead, removing", id)
					delete(e.workers, id)
				}
			}
			e.workersMu.Unlock()
		}
	}
}

// finalizeDagRun updates the final state of a DAG run
func (e *DistributedExecutor) finalizeDagRun(ctx context.Context, dagRun *models.DAGRun, failed bool) {
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
