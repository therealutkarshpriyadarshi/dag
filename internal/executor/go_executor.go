package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// GoTaskFunc is a function type for Go task execution
type GoTaskFunc func(ctx context.Context) error

// GoFuncTaskExecutor executes Go functions
type GoFuncTaskExecutor struct {
	functions map[string]GoTaskFunc
	mu        sync.RWMutex
}

// NewGoFuncTaskExecutor creates a new Go function task executor
func NewGoFuncTaskExecutor() *GoFuncTaskExecutor {
	return &GoFuncTaskExecutor{
		functions: make(map[string]GoTaskFunc),
	}
}

// RegisterFunction registers a Go function that can be executed by task ID
func (e *GoFuncTaskExecutor) RegisterFunction(taskID string, fn GoTaskFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.functions[taskID] = fn
}

// Type returns the task type this executor handles
func (e *GoFuncTaskExecutor) Type() models.TaskType {
	return models.TaskTypeGo
}

// Execute runs a registered Go function and returns the result
func (e *GoFuncTaskExecutor) Execute(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) *TaskResult {
	startTime := time.Now()
	hostname, _ := os.Hostname()

	result := &TaskResult{
		State:     models.StateSuccess,
		StartTime: startTime,
		Hostname:  hostname,
	}

	log.Printf("Executing Go function task: %s", task.ID)

	// Get the function
	e.mu.RLock()
	fn, exists := e.functions[task.ID]
	e.mu.RUnlock()

	if !exists {
		// Try using command as function name
		e.mu.RLock()
		fn, exists = e.functions[task.Command]
		e.mu.RUnlock()

		if !exists {
			result.EndTime = time.Now()
			result.State = models.StateFailed
			result.ErrorMessage = fmt.Sprintf("No function registered for task ID: %s or command: %s", task.ID, task.Command)
			log.Printf("Go function task %s failed: no function registered", task.ID)
			return result
		}
	}

	// Execute the function with panic recovery
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic occurred: %v", r)
			}
		}()
		err = fn(ctx)
	}()

	result.EndTime = time.Now()

	if err != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Function execution failed: %v", err)
		result.Output = fmt.Sprintf("Error: %v", err)
		log.Printf("Go function task %s failed: %v", task.ID, err)
	} else {
		result.Output = "Function executed successfully"
		log.Printf("Go function task %s completed successfully", task.ID)
	}

	return result
}
