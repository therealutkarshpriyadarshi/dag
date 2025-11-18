package errorhandling

import (
	"context"
	"errors"
	"testing"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

func TestPropagationHandler_HandleTaskFailure_FailPolicy(t *testing.T) {
	config := &PropagationConfig{
		Policy: PropagationPolicyFail,
	}
	handler := NewPropagationHandler(config)

	task := &models.Task{ID: "task1", Name: "Task 1"}
	taskInstance := &models.TaskInstance{ID: "ti1", TaskID: "task1", State: models.StateFailed}
	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
	err := errors.New("task error")

	resultErr := handler.HandleTaskFailure(context.Background(), task, taskInstance, dag, err)
	if resultErr == nil {
		t.Error("Expected error for fail policy, got nil")
	}
}

func TestPropagationHandler_HandleTaskFailure_SkipDownstreamPolicy(t *testing.T) {
	config := &PropagationConfig{
		Policy: PropagationPolicySkipDownstream,
	}
	handler := NewPropagationHandler(config)

	task := &models.Task{ID: "task1", Name: "Task 1"}
	taskInstance := &models.TaskInstance{ID: "ti1", TaskID: "task1", State: models.StateFailed}
	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
	err := errors.New("task error")

	resultErr := handler.HandleTaskFailure(context.Background(), task, taskInstance, dag, err)
	if resultErr == nil {
		t.Error("Expected error for skip downstream policy, got nil")
	}
}

func TestPropagationHandler_HandleTaskFailure_AllowPartialPolicy(t *testing.T) {
	config := &PropagationConfig{
		Policy:              PropagationPolicyAllowPartial,
		AllowPartialSuccess: true,
	}
	handler := NewPropagationHandler(config)

	task := &models.Task{ID: "task1", Name: "Task 1"}
	taskInstance := &models.TaskInstance{ID: "ti1", TaskID: "task1", State: models.StateFailed}
	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
	err := errors.New("task error")

	resultErr := handler.HandleTaskFailure(context.Background(), task, taskInstance, dag, err)
	if resultErr != nil {
		t.Errorf("Expected nil error for allow partial policy, got %v", resultErr)
	}
}

func TestPropagationHandler_HandleTaskFailure_CriticalTask(t *testing.T) {
	config := &PropagationConfig{
		Policy:              PropagationPolicyAllowPartial,
		AllowPartialSuccess: true,
		CriticalTasks:       []string{"task1"},
	}
	handler := NewPropagationHandler(config)

	task := &models.Task{ID: "task1", Name: "Task 1"}
	taskInstance := &models.TaskInstance{ID: "ti1", TaskID: "task1", State: models.StateFailed}
	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
	err := errors.New("task error")

	resultErr := handler.HandleTaskFailure(context.Background(), task, taskInstance, dag, err)
	if resultErr == nil {
		t.Error("Expected error for critical task failure, got nil")
	}
}

func TestPropagationHandler_HandleTaskFailure_WithCallback(t *testing.T) {
	callbackCalled := false
	config := &PropagationConfig{
		Policy: PropagationPolicySkipDownstream,
		OnTaskFailure: func(ctx context.Context, task *models.Task, taskInstance *models.TaskInstance, err error) error {
			callbackCalled = true
			return nil
		},
	}
	handler := NewPropagationHandler(config)

	task := &models.Task{ID: "task1", Name: "Task 1"}
	taskInstance := &models.TaskInstance{ID: "ti1", TaskID: "task1", State: models.StateFailed}
	dag := &models.DAG{ID: "dag1", Name: "Test DAG"}
	err := errors.New("task error")

	handler.HandleTaskFailure(context.Background(), task, taskInstance, dag, err)

	if !callbackCalled {
		t.Error("Task failure callback was not called")
	}
}

func TestPropagationHandler_ShouldMarkDownstreamFailed(t *testing.T) {
	tests := []struct {
		name     string
		policy   PropagationPolicy
		expected bool
	}{
		{"fail policy", PropagationPolicyFail, true},
		{"skip downstream policy", PropagationPolicySkipDownstream, true},
		{"allow partial policy", PropagationPolicyAllowPartial, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &PropagationConfig{Policy: tt.policy}
			handler := NewPropagationHandler(config)

			task := &models.Task{ID: "task1"}
			result := handler.ShouldMarkDownstreamFailed(task)

			if result != tt.expected {
				t.Errorf("ShouldMarkDownstreamFailed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPropagationHandler_CanDAGSucceed(t *testing.T) {
	tests := []struct {
		name          string
		config        *PropagationConfig
		taskInstances []*models.TaskInstance
		expected      bool
	}{
		{
			name: "all tasks succeeded",
			config: &PropagationConfig{
				AllowPartialSuccess: false,
			},
			taskInstances: []*models.TaskInstance{
				{TaskID: "task1", State: models.StateSuccess},
				{TaskID: "task2", State: models.StateSuccess},
			},
			expected: true,
		},
		{
			name: "one task failed, no partial success",
			config: &PropagationConfig{
				AllowPartialSuccess: false,
			},
			taskInstances: []*models.TaskInstance{
				{TaskID: "task1", State: models.StateSuccess},
				{TaskID: "task2", State: models.StateFailed},
			},
			expected: false,
		},
		{
			name: "non-critical task failed, partial success allowed",
			config: &PropagationConfig{
				AllowPartialSuccess: true,
				CriticalTasks:       []string{"task1"},
			},
			taskInstances: []*models.TaskInstance{
				{TaskID: "task1", State: models.StateSuccess},
				{TaskID: "task2", State: models.StateFailed},
			},
			expected: true,
		},
		{
			name: "critical task failed, partial success allowed",
			config: &PropagationConfig{
				AllowPartialSuccess: true,
				CriticalTasks:       []string{"task1"},
			},
			taskInstances: []*models.TaskInstance{
				{TaskID: "task1", State: models.StateFailed},
				{TaskID: "task2", State: models.StateSuccess},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPropagationHandler(tt.config)
			result := handler.CanDAGSucceed(tt.taskInstances)

			if result != tt.expected {
				t.Errorf("CanDAGSucceed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorClassifier_IsRetryable(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		errorCode string
		expected  bool
	}{
		{"timeout", true},
		{"connection_refused", true},
		{"temporary", true},
		{"rate_limit", true},
		{"service_unavailable", true},
		{"network", true},
		{"validation_error", false},
		{"not_found", false},
		{"permission_denied", false},
	}

	for _, tt := range tests {
		t.Run(tt.errorCode, func(t *testing.T) {
			result := classifier.IsRetryable(tt.errorCode)
			if result != tt.expected {
				t.Errorf("IsRetryable(%s) = %v, want %v", tt.errorCode, result, tt.expected)
			}
		})
	}
}

func TestErrorClassifier_AddRemoveRetryableError(t *testing.T) {
	classifier := NewErrorClassifier()

	// Add a custom error code
	classifier.AddRetryableError("custom_error")
	if !classifier.IsRetryable("custom_error") {
		t.Error("Added error code should be retryable")
	}

	// Remove the error code
	classifier.RemoveRetryableError("custom_error")
	if classifier.IsRetryable("custom_error") {
		t.Error("Removed error code should not be retryable")
	}
}

func TestPropagationHandler_HandleDAGFailure(t *testing.T) {
	callbackCalled := false
	config := &PropagationConfig{
		OnDAGFailure: func(ctx context.Context, dagRun *models.DAGRun, err error) error {
			callbackCalled = true
			return nil
		},
	}
	handler := NewPropagationHandler(config)

	dagRun := &models.DAGRun{ID: "dr1", DAGID: "dag1", State: models.StateFailed}
	err := errors.New("DAG error")

	handler.HandleDAGFailure(context.Background(), dagRun, err)

	if !callbackCalled {
		t.Error("DAG failure callback was not called")
	}
}
