package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// Worker represents a distributed worker that processes tasks from NATS
type Worker struct {
	id            string
	hostname      string
	nc            *nats.Conn
	js            nats.JetStreamContext
	taskExecutors map[models.TaskType]TaskExecutor
	config        *ExecutorConfig

	taskSub      *nats.Subscription
	activeTasks  int
	mu           sync.RWMutex
	running      bool
	wg           sync.WaitGroup
}

// NewWorker creates a new distributed worker
func NewWorker(natsURL string, config *ExecutorConfig) (*Worker, error) {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])

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

	return &Worker{
		id:            workerID,
		hostname:      hostname,
		nc:            nc,
		js:            js,
		taskExecutors: make(map[models.TaskType]TaskExecutor),
		config:        config,
		activeTasks:   0,
		running:       false,
	}, nil
}

// RegisterTaskExecutor registers a task executor
func (w *Worker) RegisterTaskExecutor(executor TaskExecutor) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.taskExecutors[executor.Type()] = executor
}

// Start starts the worker and begins processing tasks
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("worker already running")
	}

	w.running = true

	// Subscribe to pending tasks with multiple workers
	var err error
	w.taskSub, err = w.js.QueueSubscribe(
		TasksPendingSubject,
		"workers",
		w.handleTask,
		nats.Durable("workers"),
		nats.ManualAck(),
		nats.AckWait(5*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to tasks: %w", err)
	}

	// Start heartbeat goroutine
	w.wg.Add(1)
	go w.sendHeartbeats(ctx)

	log.Printf("Worker %s started on %s", w.id, w.hostname)
	return nil
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	log.Printf("Stopping worker %s...", w.id)

	// Unsubscribe from tasks
	if w.taskSub != nil {
		w.taskSub.Unsubscribe()
	}

	// Wait for active tasks to complete
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Worker stopped gracefully")
	case <-time.After(w.config.ShutdownTimeout):
		log.Println("Worker shutdown timeout reached")
	}

	// Close NATS connection
	w.nc.Close()

	log.Printf("Worker %s stopped", w.id)
	return nil
}

// handleTask processes a single task from the queue
func (w *Worker) handleTask(msg *nats.Msg) {
	var taskMsg TaskMessage
	if err := json.Unmarshal(msg.Data, &taskMsg); err != nil {
		log.Printf("Failed to unmarshal task message: %v", err)
		msg.Nak()
		return
	}

	log.Printf("Worker %s received task %s (type: %s)", w.id, taskMsg.TaskID, taskMsg.TaskType)

	// Get task executor
	w.mu.RLock()
	executor, ok := w.taskExecutors[models.TaskType(taskMsg.TaskType)]
	w.mu.RUnlock()

	if !ok {
		log.Printf("No executor registered for task type %s", taskMsg.TaskType)
		result := &TaskResultMessage{
			TaskInstanceID: taskMsg.TaskInstanceID,
			WorkerID:       w.id,
			State:          string(models.StateFailed),
			ErrorMessage:   fmt.Sprintf("No executor for task type: %s", taskMsg.TaskType),
			StartTime:      time.Now(),
			EndTime:        time.Now(),
			Hostname:       w.hostname,
		}
		w.publishResult(result)
		msg.Ack()
		return
	}

	// Increment active tasks
	w.mu.Lock()
	w.activeTasks++
	w.mu.Unlock()

	// Execute task
	ctx := context.Background()
	if taskMsg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, taskMsg.Timeout)
		defer cancel()
	}

	task := &models.Task{
		ID:      taskMsg.TaskID,
		Type:    models.TaskType(taskMsg.TaskType),
		Command: taskMsg.Command,
		Timeout: taskMsg.Timeout,
		Retries: taskMsg.Retries,
	}

	taskInstance := &models.TaskInstance{
		ID:       taskMsg.TaskInstanceID,
		TaskID:   taskMsg.TaskID,
		DAGRunID: taskMsg.DAGRunID,
	}

	result := executor.Execute(ctx, task, taskInstance)

	// Decrement active tasks
	w.mu.Lock()
	w.activeTasks--
	w.mu.Unlock()

	// Publish result
	resultMsg := &TaskResultMessage{
		TaskInstanceID: taskMsg.TaskInstanceID,
		WorkerID:       w.id,
		State:          string(result.State),
		Output:         result.Output,
		ErrorMessage:   result.ErrorMessage,
		StartTime:      result.StartTime,
		EndTime:        result.EndTime,
		Hostname:       result.Hostname,
	}

	if err := w.publishResult(resultMsg); err != nil {
		log.Printf("Failed to publish result: %v", err)
		msg.Nak()
		return
	}

	msg.Ack()
	log.Printf("Worker %s completed task %s with state %s", w.id, taskMsg.TaskID, result.State)
}

// publishResult publishes a task result to NATS
func (w *Worker) publishResult(result *TaskResultMessage) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	_, err = w.js.Publish(TasksResultsSubject, data)
	if err != nil {
		return fmt.Errorf("failed to publish result: %w", err)
	}

	return nil
}

// sendHeartbeats sends periodic heartbeats to the executor
func (w *Worker) sendHeartbeats(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.RLock()
			if !w.running {
				w.mu.RUnlock()
				return
			}
			activeTasks := w.activeTasks
			w.mu.RUnlock()

			heartbeat := &WorkerHeartbeat{
				WorkerID:    w.id,
				Hostname:    w.hostname,
				ActiveTasks: activeTasks,
				Timestamp:   time.Now(),
			}

			data, err := json.Marshal(heartbeat)
			if err != nil {
				log.Printf("Failed to marshal heartbeat: %v", err)
				continue
			}

			if err := w.nc.Publish(WorkerHeartbeatSubject, data); err != nil {
				log.Printf("Failed to publish heartbeat: %v", err)
			}
		}
	}
}

// GetID returns the worker ID
func (w *Worker) GetID() string {
	return w.id
}

// GetActiveTasks returns the number of active tasks
func (w *Worker) GetActiveTasks() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.activeTasks
}
