package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestGoFuncTaskExecutor_Execute_Success(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	// Register a test function
	executor.RegisterFunction("test-task", func(ctx context.Context) error {
		return nil
	})

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeGo,
		Command: "test-task",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateSuccess {
		t.Errorf("Expected state Success, got %s. Error: %s", result.State, result.ErrorMessage)
	}

	if result.ErrorMessage != "" {
		t.Errorf("Expected no error message, got: %s", result.ErrorMessage)
	}
}

func TestGoFuncTaskExecutor_Execute_Failure(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	// Register a test function that returns an error
	executor.RegisterFunction("test-task", func(ctx context.Context) error {
		return errors.New("test error")
	})

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeGo,
		Command: "test-task",
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

	if !strings.Contains(result.ErrorMessage, "test error") {
		t.Errorf("Expected error message to contain 'test error', got: %s", result.ErrorMessage)
	}
}

func TestGoFuncTaskExecutor_Execute_Panic(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	// Register a test function that panics
	executor.RegisterFunction("test-task", func(ctx context.Context) error {
		panic("test panic")
	})

	task := &models.Task{
		ID:      "test-task",
		Type:    models.TaskTypeGo,
		Command: "test-task",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateFailed {
		t.Errorf("Expected state Failed due to panic, got %s", result.State)
	}

	if !strings.Contains(result.ErrorMessage, "panic") {
		t.Errorf("Expected error message to contain 'panic', got: %s", result.ErrorMessage)
	}
}

func TestGoFuncTaskExecutor_Execute_NoFunction(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	task := &models.Task{
		ID:      "nonexistent-task",
		Type:    models.TaskTypeGo,
		Command: "nonexistent-task",
		Timeout: 10 * time.Second,
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "nonexistent-task",
	}

	ctx := context.Background()
	result := executor.Execute(ctx, task, taskInstance)

	if result.State != models.StateFailed {
		t.Errorf("Expected state Failed for missing function, got %s", result.State)
	}

	if !strings.Contains(result.ErrorMessage, "No function registered") {
		t.Errorf("Expected error about missing function, got: %s", result.ErrorMessage)
	}
}

func TestGoFuncTaskExecutor_Type(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	if executor.Type() != models.TaskTypeGo {
		t.Errorf("Expected type Go, got %s", executor.Type())
	}
}

func TestGoFuncTaskExecutor_RegisterFunction(t *testing.T) {
	executor := NewGoFuncTaskExecutor()

	callCount := 0
	executor.RegisterFunction("test", func(ctx context.Context) error {
		callCount++
		return nil
	})

	task := &models.Task{
		ID:      "test",
		Type:    models.TaskTypeGo,
		Command: "test",
	}

	taskInstance := &models.TaskInstance{
		ID:     "test-instance",
		TaskID: "test",
	}

	executor.Execute(context.Background(), task, taskInstance)

	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
}
