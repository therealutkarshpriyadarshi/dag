package models

import (
	"testing"
	"time"
)

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"Success is terminal", StateSuccess, true},
		{"Failed is terminal", StateFailed, true},
		{"Skipped is terminal", StateSkipped, true},
		{"Queued is not terminal", StateQueued, false},
		{"Running is not terminal", StateRunning, false},
		{"Retrying is not terminal", StateRetrying, false},
		{"Upstream failed is not terminal", StateUpstreamFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.IsTerminal()
			if got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDAG_Creation(t *testing.T) {
	now := time.Now()
	dag := &DAG{
		ID:          "dag-123",
		Name:        "test-dag",
		Description: "Test DAG",
		Schedule:    "0 0 * * *",
		Tasks: []Task{
			{
				ID:      "task1",
				Name:    "Task 1",
				Type:    TaskTypeBash,
				Command: "echo hello",
			},
		},
		StartDate: now,
		Tags:      []string{"test", "example"},
		IsPaused:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if dag.ID != "dag-123" {
		t.Errorf("Expected DAG ID 'dag-123', got '%s'", dag.ID)
	}
	if dag.Name != "test-dag" {
		t.Errorf("Expected DAG name 'test-dag', got '%s'", dag.Name)
	}
	if len(dag.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(dag.Tasks))
	}
	if len(dag.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(dag.Tags))
	}
}

func TestTask_Creation(t *testing.T) {
	task := Task{
		ID:           "task-1",
		Name:         "Extract Data",
		Type:         TaskTypeBash,
		Command:      "python extract.py",
		Dependencies: []string{"task-0"},
		Retries:      3,
		Timeout:      30 * time.Minute,
		SLA:          1 * time.Hour,
	}

	if task.ID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got '%s'", task.ID)
	}
	if task.Type != TaskTypeBash {
		t.Errorf("Expected task type 'bash', got '%s'", task.Type)
	}
	if task.Retries != 3 {
		t.Errorf("Expected 3 retries, got %d", task.Retries)
	}
	if len(task.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(task.Dependencies))
	}
}

func TestTaskType_Values(t *testing.T) {
	types := []TaskType{
		TaskTypeBash,
		TaskTypeHTTP,
		TaskTypePython,
		TaskTypeGo,
	}

	expected := []string{"bash", "http", "python", "go"}

	for i, tt := range types {
		if string(tt) != expected[i] {
			t.Errorf("Expected task type '%s', got '%s'", expected[i], tt)
		}
	}
}

func TestDAGRun_Creation(t *testing.T) {
	now := time.Now()
	dagRun := &DAGRun{
		ID:              "run-123",
		DAGID:           "dag-123",
		ExecutionDate:   now,
		State:           StateRunning,
		StartDate:       &now,
		ExternalTrigger: false,
	}

	if dagRun.State != StateRunning {
		t.Errorf("Expected state 'running', got '%s'", dagRun.State)
	}
	if dagRun.ExternalTrigger {
		t.Error("Expected ExternalTrigger to be false")
	}
	if dagRun.DAGID != "dag-123" {
		t.Errorf("Expected DAG ID 'dag-123', got '%s'", dagRun.DAGID)
	}
}

func TestTaskInstance_Creation(t *testing.T) {
	now := time.Now()
	duration := 5 * time.Minute

	taskInstance := &TaskInstance{
		ID:        "ti-123",
		TaskID:    "task-1",
		DAGRunID:  "run-123",
		State:     StateSuccess,
		TryNumber: 1,
		MaxTries:  3,
		StartDate: &now,
		EndDate:   &now,
		Duration:  duration,
		Hostname:  "worker-1",
	}

	if taskInstance.State != StateSuccess {
		t.Errorf("Expected state 'success', got '%s'", taskInstance.State)
	}
	if taskInstance.TryNumber != 1 {
		t.Errorf("Expected try number 1, got %d", taskInstance.TryNumber)
	}
	if taskInstance.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, taskInstance.Duration)
	}
	if taskInstance.Hostname != "worker-1" {
		t.Errorf("Expected hostname 'worker-1', got '%s'", taskInstance.Hostname)
	}
}
