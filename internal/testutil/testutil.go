package testutil

import (
	"time"

	"github.com/therealutkarshpriyadarshi/dag/pkg/models"
)

// CreateTestDAG creates a simple DAG for testing purposes
func CreateTestDAG(name string) *models.DAG {
	return &models.DAG{
		ID:          "dag-" + name,
		Name:        name,
		Description: "Test DAG: " + name,
		Schedule:    "0 0 * * *",
		Tasks: []models.Task{
			{
				ID:      "task1",
				Name:    "Task 1",
				Type:    models.TaskTypeBash,
				Command: "echo 'task1'",
			},
		},
		StartDate: time.Now(),
		Tags:      []string{"test"},
		IsPaused:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestDAGWithDependencies creates a DAG with task dependencies
func CreateTestDAGWithDependencies(name string) *models.DAG {
	return &models.DAG{
		ID:          "dag-" + name,
		Name:        name,
		Description: "Test DAG with dependencies: " + name,
		Schedule:    "0 0 * * *",
		Tasks: []models.Task{
			{
				ID:      "task1",
				Name:    "Task 1",
				Type:    models.TaskTypeBash,
				Command: "echo 'task1'",
			},
			{
				ID:           "task2",
				Name:         "Task 2",
				Type:         models.TaskTypeBash,
				Command:      "echo 'task2'",
				Dependencies: []string{"task1"},
			},
			{
				ID:           "task3",
				Name:         "Task 3",
				Type:         models.TaskTypeBash,
				Command:      "echo 'task3'",
				Dependencies: []string{"task1"},
			},
			{
				ID:           "task4",
				Name:         "Task 4",
				Type:         models.TaskTypeBash,
				Command:      "echo 'task4'",
				Dependencies: []string{"task2", "task3"},
			},
		},
		StartDate: time.Now(),
		Tags:      []string{"test", "complex"},
		IsPaused:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestTask creates a simple task for testing
func CreateTestTask(id, name string, deps []string) models.Task {
	return models.Task{
		ID:           id,
		Name:         name,
		Type:         models.TaskTypeBash,
		Command:      "echo '" + name + "'",
		Dependencies: deps,
		Retries:      3,
		Timeout:      30 * time.Minute,
		SLA:          1 * time.Hour,
	}
}

// CreateTestDAGRun creates a DAG run for testing
func CreateTestDAGRun(dagID string, state models.State) *models.DAGRun {
	now := time.Now()
	return &models.DAGRun{
		ID:              "run-" + dagID,
		DAGID:           dagID,
		ExecutionDate:   now,
		State:           state,
		StartDate:       &now,
		ExternalTrigger: false,
	}
}

// CreateTestTaskInstance creates a task instance for testing
func CreateTestTaskInstance(taskID, dagRunID string, state models.State) *models.TaskInstance {
	now := time.Now()
	return &models.TaskInstance{
		ID:        "ti-" + taskID,
		TaskID:    taskID,
		DAGRunID:  dagRunID,
		State:     state,
		TryNumber: 1,
		MaxTries:  3,
		StartDate: &now,
		Hostname:  "test-worker",
	}
}
