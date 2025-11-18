package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestBashTaskExecutor_Execute_Success(t *testing.T) {
	executor := NewBashTaskExecutor()

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeBash,
		Command: "echo 'Hello, World!'",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateSuccess {
		t.Errorf("Expected state Success, got %s", result.State)
	}

	if !strings.Contains(result.Output, "Hello, World!") {
		t.Errorf("Expected output to contain 'Hello, World!', got: %s", result.Output)
	}

	if result.ErrorMessage != "" {
		t.Errorf("Expected no error message, got: %s", result.ErrorMessage)
	}
}

func TestBashTaskExecutor_Execute_Failure(t *testing.T) {
	executor := NewBashTaskExecutor()

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeBash,
		Command: "exit 1",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateFailed {
		t.Errorf("Expected state Failed, got %s", result.State)
	}

	if result.ErrorMessage == "" {
		t.Error("Expected error message, got empty string")
	}
}

func TestBashTaskExecutor_Execute_Timeout(t *testing.T) {
	executor := NewBashTaskExecutor()

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeBash,
		Command: "sleep 5",
		Timeout: 1 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateFailed {
		t.Errorf("Expected state Failed due to timeout, got %s", result.State)
	}

	if !strings.Contains(result.ErrorMessage, "timed out") {
		t.Errorf("Expected timeout error message, got: %s", result.ErrorMessage)
	}
}

func TestBashTaskExecutor_Type(t *testing.T) {
	executor := NewBashTaskExecutor()

	if executor.Type() != models.TaskTypeBash {
		t.Errorf("Expected type Bash, got %s", executor.Type())
	}
}

func TestBashTaskExecutor_WithConfig(t *testing.T) {
	workingDir := "/tmp"
	env := []string{"TEST_VAR=test_value"}

	executor := NewBashTaskExecutorWithConfig(workingDir, env)

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeBash,
		Command: "echo $TEST_VAR",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateSuccess {
		t.Errorf("Expected state Success, got %s", result.State)
	}
}
