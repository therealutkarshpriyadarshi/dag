package executor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// BashTaskExecutor executes bash commands
type BashTaskExecutor struct {
	workingDir string
	env        []string
}

// NewBashTaskExecutor creates a new bash task executor
func NewBashTaskExecutor() *BashTaskExecutor {
	return &BashTaskExecutor{
		workingDir: "",
		env:        os.Environ(),
	}
}

// NewBashTaskExecutorWithConfig creates a bash executor with custom config
func NewBashTaskExecutorWithConfig(workingDir string, env []string) *BashTaskExecutor {
	if env == nil {
		env = os.Environ()
	}
	return &BashTaskExecutor{
		workingDir: workingDir,
		env:        env,
	}
}

// Type returns the task type this executor handles
func (e *BashTaskExecutor) Type() models.TaskType {
	return models.TaskTypeBash
}

// Execute runs a bash command and returns the result
func (e *BashTaskExecutor) Execute(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance) *TaskResult {
	startTime := time.Now()
	hostname, _ := os.Hostname()

	result := &TaskResult{
		State:     models.StateSuccess,
		StartTime: startTime,
		Hostname:  hostname,
	}

	log.Printf("Executing bash task: %s, command: %s", task.ID, task.Command)

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", task.Command)

	// Set working directory if specified
	if e.workingDir != "" {
		cmd.Dir = e.workingDir
	}

	// Set environment variables
	cmd.Env = e.env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	result.EndTime = time.Now()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	result.Output = output

	// Check execution result
	if err != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Command failed: %v\nOutput: %s", err, output)
		log.Printf("Bash task %s failed: %v", task.ID, err)
	} else {
		log.Printf("Bash task %s completed successfully", task.ID)
	}

	// Check for context cancellation (timeout)
	if ctx.Err() != nil {
		result.State = models.StateFailed
		result.ErrorMessage = fmt.Sprintf("Task timed out: %v\nOutput: %s", ctx.Err(), output)
		log.Printf("Bash task %s timed out", task.ID)
	}

	return result
}
